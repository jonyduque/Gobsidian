package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
)

// TestDaemonComCofreInexistenteRegistraCausa cobre a morte muda medida em
// 2026-08-26 na maquina do dono: dois daemons registraram "daemon iniciado" e
// NADA mais. O processo saiu, a ponte esperou o prazo inteiro do
// EnsureStarted, caiu para o modo em processo, e o unico rastro no host foi
// "Server transport closed unexpectedly".
//
// A causa existia e era especifica -- vault.New (internal/vault/vault.go:90-95)
// devolve "raiz do cofre inacessivel %q" para caminho ausente. Ela so nunca
// chegava a lugar nenhum: runDaemon fazia `return err` depois de ja ter
// logado "daemon iniciado", e o stderr de um processo detachado nao vai a
// lugar nenhum que alguem possa ler.
//
// O teste afirma o que um humano precisaria para diagnosticar: que houve um
// ERROR, e que ele nomeia o caminho recusado.
func TestDaemonComCofreInexistenteRegistraCausa(t *testing.T) {
	inexistente := filepath.Join(t.TempDir(), "cofre-que-nao-existe")

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := config.Config{
		VaultPath: inexistente,
		CacheDir:  t.TempDir(),
		LogLevel:  slog.LevelDebug,
	}

	err := runDaemon(context.Background(), cfg, time.Second, log)
	if err == nil {
		t.Fatal("runDaemon devolveu nil para cofre inexistente")
	}

	saida := buf.String()
	if !strings.Contains(saida, "level=ERROR") {
		t.Errorf("nenhum log de ERROR antes da saida.\nlog foi:\n%s", saida)
	}
	if !strings.Contains(saida, inexistente) {
		t.Errorf("o log nao nomeia o caminho recusado %q.\nlog foi:\n%s", inexistente, saida)
	}
}

// TestServeEmProcessoComCofreInexistenteRegistraCausa e o irmao do teste
// acima para o outro caminho de boot. Ele importa porque o fallback em
// processo e obrigatorio quando o daemon nao sobe: se os DOIS morrem calados,
// o cofre mal configurado nao produz mensagem acionavel em lugar nenhum -- foi
// exatamente o que aconteceu com gobsidian-jurisprudencia por dois dias.
func TestServeEmProcessoComCofreInexistenteRegistraCausa(t *testing.T) {
	inexistente := filepath.Join(t.TempDir(), "outro-cofre-ausente")

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := config.Config{
		VaultPath: inexistente,
		CacheDir:  t.TempDir(),
		LogLevel:  slog.LevelDebug,
	}

	// stdin fechado: serveEmProcesso monta o servico ANTES de tocar em stdin,
	// e e na montagem que ele tem de falhar. Se um dia ele passar da montagem
	// neste teste, o EOF encerra em vez de pendurar.
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("abrindo %s: %v", os.DevNull, err)
	}
	defer func() { _ = devNull.Close() }()

	if err := serveEmProcesso(context.Background(), cfg, log); err == nil {
		t.Fatal("serveEmProcesso devolveu nil para cofre inexistente")
	}

	saida := buf.String()
	if !strings.Contains(saida, "level=ERROR") {
		t.Errorf("nenhum log de ERROR antes da saida.\nlog foi:\n%s", saida)
	}
	if !strings.Contains(saida, inexistente) {
		t.Errorf("o log nao nomeia o caminho recusado %q.\nlog foi:\n%s", inexistente, saida)
	}
}

// TestLoggerDoDaemonCarimbaPidEVersao: o arquivo de log e UNICO por cofre e
// recebe append de N instancias ao longo de meses -- 727 261 bytes na maquina
// do dono, medido em 2026-09-08.
//
// Sem pid e versao em cada linha, descobrir qual processo escreveu uma linha
// exige cruzar mtime de arquivo com StartTime de processo. Foi o que a
// investigacao de 2026-09-08 teve de fazer, e o resultado ficou ambiguo
// justamente no caso que importava: duas instancias do mesmo cofre convivendo.
//
// Com o instalador (spec D-06/D-07) versoes deixam de conviver por desenho,
// mas o log continua sendo lido DEPOIS do fato, quando quem le nao sabe o que
// havia instalado na epoca.
func TestLoggerDoDaemonCarimbaPidEVersao(t *testing.T) {
	cofre := t.TempDir()
	// Desvia do diretorio de runtime REAL do usuario: os.UserCacheDir le
	// %LocalAppData% no Windows, e um teste nao pode sujar
	// %LocalAppData%\gobsidian\run -- que ja acumulou 960 arquivos .lock
	// exatamente assim (medido em 2026-09-08).
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	log, fechar, err := novoLoggerDoDaemon(cofre, slog.LevelInfo)
	if err != nil {
		t.Fatalf("novoLoggerDoDaemon() error = %v", err)
	}
	log.Info("linha de teste")
	if err := fechar(); err != nil {
		t.Fatalf("fechar() error = %v", err)
	}

	caminho, err := daemonLogPath(cofre)
	if err != nil {
		t.Fatalf("daemonLogPath() error = %v", err)
	}
	conteudo, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("lendo o log: %v", err)
	}

	linha := string(conteudo)
	esperado := "pid=" + strconv.Itoa(os.Getpid())
	if !strings.Contains(linha, esperado) {
		t.Errorf("log sem %s -- nao da para saber qual processo escreveu:\n%s", esperado, linha)
	}
	if !strings.Contains(linha, "versao=") {
		t.Errorf("log sem versao= -- nao da para saber qual binario escreveu:\n%s", linha)
	}
}

// TestLoggerDoDaemonRotacionaAcimaDoTeto: medido em 2026-09-08, o log do cofre
// Estudo tinha 727 261 bytes e nenhum limite -- ele so cresce, para sempre.
//
// Rotaciona, nunca apaga. O log e a UNICA memoria do daemon, e a investigacao
// de 2026-09-08 dependeu de linhas de 2026-08-24 para reconstruir a sequencia
// de partidas e mortes. Um arquivo anterior e o suficiente para nao perder a
// janela recente sem crescer sem limite.
func TestLoggerDoDaemonRotacionaAcimaDoTeto(t *testing.T) {
	cofre := t.TempDir()
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	// Teto minusculo: o que se prova e o COMPORTAMENTO, nao o numero. Escrever
	// 5 MB num teste so gastaria disco para provar a mesma coisa.
	tetoOriginal := tetoDoLogDoDaemon
	tetoDoLogDoDaemon = 32
	t.Cleanup(func() { tetoDoLogDoDaemon = tetoOriginal })

	caminho, err := daemonLogPath(cofre)
	if err != nil {
		t.Fatalf("daemonLogPath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
		t.Fatal(err)
	}
	antigo := strings.Repeat("linha antiga que precisa sobreviver\n", 4)
	if err := os.WriteFile(caminho, []byte(antigo), 0o600); err != nil {
		t.Fatal(err)
	}

	log, fechar, err := novoLoggerDoDaemon(cofre, slog.LevelInfo)
	if err != nil {
		t.Fatalf("novoLoggerDoDaemon() error = %v", err)
	}
	log.Info("linha nova")
	if err := fechar(); err != nil {
		t.Fatalf("fechar() error = %v", err)
	}

	corrente, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("lendo o log corrente: %v", err)
	}
	if strings.Contains(string(corrente), "linha antiga") {
		t.Errorf("o log corrente ainda tem o conteudo velho; nao rotacionou:\n%s", corrente)
	}
	if !strings.Contains(string(corrente), "linha nova") {
		t.Errorf("o log corrente perdeu a linha nova:\n%s", corrente)
	}

	guardado, err := os.ReadFile(caminho + ".1")
	if err != nil {
		t.Fatalf("o arquivo anterior nao existe -- rotacionar virou APAGAR, e o log e a unica memoria do daemon: %v", err)
	}
	if string(guardado) != antigo {
		t.Errorf("o arquivo anterior nao preservou o conteudo:\n%s", guardado)
	}
}

// TestLoggerDoDaemonNaoRotacionaAbaixoDoTeto: sem este caso, uma rotacao que
// acontecesse SEMPRE passaria no teste acima -- e o daemon perderia a janela
// recente a cada partida.
func TestLoggerDoDaemonNaoRotacionaAbaixoDoTeto(t *testing.T) {
	cofre := t.TempDir()
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	caminho, err := daemonLogPath(cofre)
	if err != nil {
		t.Fatalf("daemonLogPath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caminho, []byte("pequeno\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, fechar, err := novoLoggerDoDaemon(cofre, slog.LevelInfo)
	if err != nil {
		t.Fatalf("novoLoggerDoDaemon() error = %v", err)
	}
	_ = fechar()

	if _, err := os.Stat(caminho + ".1"); !os.IsNotExist(err) {
		t.Errorf("rotacionou um log de 8 bytes; o teto nao esta sendo respeitado")
	}
	corrente, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(corrente), "pequeno") {
		t.Errorf("o log corrente perdeu o conteudo anterior:\n%s", corrente)
	}
}
