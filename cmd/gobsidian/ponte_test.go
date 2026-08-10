package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/ipc"
)

// TestPonteCaiParaModoEmProcesso e o teste nomeado na prova de mutacao da
// Task 91: se a chamada "return serveEmProcesso(ctx, cfg, log)" em
// servePonte for trocada por "return err", este teste tem de reprovar.
//
// A distincao nao depende de levantar um servidor MCP de verdade -- isso
// bloquearia em os.Stdin, que o processo de teste nao controla. Em vez
// disso, o cofre e um caminho que nao existe: serveEmProcesso falha DENTRO
// dele, no primeiro passo (vault.New), antes de tocar em stdin nenhum, com
// uma mensagem que so aparece se o fallback de fato chamou serveEmProcesso.
// A mutacao devolveria o erro da conexao recusada em vez disso -- um erro
// com uma mensagem completamente diferente -- e este teste distingue os
// dois.

// semDaemonParaTeste desliga a tentativa de iniciar um daemon de verdade
// nos testes deste arquivo: sob "go test", os.Executable() devolveria o
// BINARIO DE TESTE, e deixar EnsureStarted (chamada por servePonte) lanca-lo
// com argumentos de subcomando lancaria um processo imprevisivel em vez de
// nada. Encolhe daemonStartTimeout tambem, senao EnsureStarted insiste por
// segundos tentando um socket que este teste nunca deixa existir.
func semDaemonParaTeste(t *testing.T) {
	t.Helper()
	origIniciar := iniciarDaemonFn
	origTimeout := daemonStartTimeout
	iniciarDaemonFn = func(config.Config) error { return nil }
	daemonStartTimeout = 60 * time.Millisecond
	t.Cleanup(func() {
		iniciarDaemonFn = origIniciar
		daemonStartTimeout = origTimeout
	})
}

func TestPonteCaiParaModoEmProcesso(t *testing.T) {
	semDaemonParaTeste(t)

	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, nil))

	vaultInexistente := filepath.Join(t.TempDir(), "nao-existe")
	cfg := config.Config{VaultPath: vaultInexistente}

	// Nenhum socket existe para este cofre (nenhum daemon rodou nunca para
	// um diretorio temporario recem-criado), entao DialAndHandshake falha
	// de verdade -- sem precisar de um dublê. iniciarDaemonFn acima nao faz
	// nada, entao EnsureStarted tambem falha em esperarSocket, e o
	// fallback obrigatorio e alcancado do mesmo jeito que antes da Task 92.
	ctx, cancel := context.WithTimeout(context.Background(), boundedWait)
	defer cancel()

	err := servePonte(ctx, cfg, log)
	if err == nil {
		t.Fatal("servePonte() error = nil, esperado erro (cofre inexistente)")
	}
	if !strings.Contains(err.Error(), "raiz do cofre inacessivel") {
		t.Fatalf("servePonte() error = %q, esperado mencionar %q -- essa mensagem so vem de dentro de serveEmProcesso -> vault.New; "+
			"se o texto for outro (por exemplo, o erro de conectar ao socket), o fallback nao foi chamado",
			err, "raiz do cofre inacessivel")
	}

	if !strings.Contains(logBuf.String(), "servindo em processo") {
		t.Fatalf("log nao registrou a queda para o modo em processo: %s", logBuf.String())
	}
}

// TestServePonteRespeitaGobsidianNoDaemon prova que GOBSIDIAN_NO_DAEMON pula
// a decisao inteira -- nem tenta discar um daemon, nem tenta iniciar um. Um
// socket presente e respondendo (o oposto do caso "ausente") teria que ser
// IGNORADO com a variavel definida; sem essa prova, um "-> servindo em
// processo" no log nao distingue "pulou por causa da variavel" de "tentou e
// falhou por outro motivo".
func TestServePonteRespeitaGobsidianNoDaemon(t *testing.T) {
	t.Setenv("GOBSIDIAN_NO_DAEMON", "1")

	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, nil))

	vaultInexistente := filepath.Join(t.TempDir(), "nao-existe")
	cfg := config.Config{VaultPath: vaultInexistente}

	// Um socket de verdade, respondendo com a saudacao certa -- se a
	// variavel nao fosse respeitada, este teste conectaria e
	// servePonteRemota ficaria presa em os.Stdin, travando o teste.
	ln, _, err := ipc.Listen(cfg.VaultPath)
	if err != nil {
		t.Fatalf("ipc.Listen() error = %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		_ = ipc.Greet(c, ipc.HandshakeConfig{VaultKey: config.VaultKey(cfg.VaultPath)})
		time.Sleep(200 * time.Millisecond)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), boundedWait)
	defer cancel()

	err = servePonte(ctx, cfg, log)
	if err == nil {
		t.Fatal("servePonte() error = nil, esperado erro (cofre inexistente, modo em processo forcado)")
	}
	if !strings.Contains(err.Error(), "raiz do cofre inacessivel") {
		t.Fatalf("servePonte() error = %q, esperado mencionar %q -- essa mensagem so vem de serveEmProcesso -> vault.New; "+
			"se o texto for outro, GOBSIDIAN_NO_DAEMON nao pulou a tentativa de socket",
			err, "raiz do cofre inacessivel")
	}
	if !strings.Contains(logBuf.String(), "GOBSIDIAN_NO_DAEMON") {
		t.Fatalf("log nao registrou que GOBSIDIAN_NO_DAEMON forcou o modo em processo: %s", logBuf.String())
	}
}

// TestServePonteVersaoDiferenteCaiParaProcesso cobre a segunda verificacao
// que o brief da Task 91 pede: um socket presente, mas de uma versao que a
// ponte nao fala, tem de cair para o mesmo lugar que um socket ausente --
// sem precisar da Task 92 (o daemon de verdade) para existir, simulando o
// lado do servidor com um listener de teste.
func TestServePonteVersaoDiferenteCaiParaProcesso(t *testing.T) {
	semDaemonParaTeste(t)

	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, nil))

	vaultInexistente := filepath.Join(t.TempDir(), "nao-existe")
	cfg := config.Config{VaultPath: vaultInexistente}

	ln, _, err := ipc.Listen(cfg.VaultPath)
	if err != nil {
		t.Fatalf("ipc.Listen() error = %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		_, _ = fmt.Fprint(c, "GOBSIDIAN-IPC 999\n")
		time.Sleep(200 * time.Millisecond)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), boundedWait)
	defer cancel()

	err = servePonte(ctx, cfg, log)
	if err == nil {
		t.Fatal("servePonte() error = nil, esperado erro (cofre inexistente, apos o fallback)")
	}
	if !strings.Contains(err.Error(), "raiz do cofre inacessivel") {
		t.Fatalf("servePonte() error = %q, esperado mencionar %q (fallback nao alcancado)", err, "raiz do cofre inacessivel")
	}
	if !strings.Contains(logBuf.String(), "servindo em processo") {
		t.Fatalf("log nao registrou a queda para o modo em processo: %s", logBuf.String())
	}
}

// TestServePonteRemotaFazProxyDeBytes prova que, quando o handshake da
// certo, a ponte de fato copia bytes do daemon para o stdout do host -- e
// que fechar a conexao do lado do daemon termina servePonteRemota
// normalmente. Usa duplexPipe (io.Pipe, nao net.Pipe) de proposito:
// cmd/gobsidian nao importa "net" em lugar nenhum, e este teste nao e
// excecao.
func TestServePonteRemotaFazProxyDeBytes(t *testing.T) {
	conn, outroLado := newDuplexPipe()
	t.Cleanup(func() { _ = conn.Close() })

	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	done := make(chan error, 1)
	go func() {
		done <- servePonteRemota(context.Background(), conn, log)
	}()

	// outroLado faz o papel do daemon: escrever nele tem que aparecer do
	// lado da ponte sem alteracao (ela copia para os.Stdout, mas o que este
	// teste consegue observar diretamente e que a leitura nao trava e o
	// fechamento propaga), e fecha-lo tem que terminar servePonteRemota --
	// o equivalente, na ponte, ao serve loop de serveEmProcesso retornando
	// quando o host desconecta.
	if _, err := outroLado.Write([]byte("resposta do daemon")); err != nil {
		t.Fatalf("escrevendo do lado do daemon: %v", err)
	}
	if err := outroLado.Close(); err != nil {
		t.Fatalf("fechando o lado do daemon: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("servePonteRemota() error = %v, esperado nil (fechamento normal)", err)
		}
	case <-time.After(boundedWait):
		t.Fatal("servePonteRemota nao retornou apos o daemon fechar a conexao")
	}
}

// duplexPipe implementa ipc.Conn (Read, Write, Close, CloseWrite) em cima de
// dois io.Pipe -- um por sentido -- para os testes deste pacote poderem
// simular o lado do daemon sem importar "net". CloseWrite fecha so a
// escrita, como o net.UnixConn de verdade faria: o outro lado ainda pode
// mandar o resto do que tinha para dizer antes de a leitura tambem acabar.
type duplexPipe struct {
	r      *io.PipeReader
	w      *io.PipeWriter
	closeR func() error
}

func (d duplexPipe) Read(p []byte) (int, error)  { return d.r.Read(p) }
func (d duplexPipe) Write(p []byte) (int, error) { return d.w.Write(p) }
func (d duplexPipe) CloseWrite() error           { return d.w.Close() }
func (d duplexPipe) Close() error {
	_ = d.w.Close()
	return d.closeR()
}

// newDuplexPipe devolve os dois lados de um par ligado: o que "a" escreve,
// "b" le, e vice-versa.
func newDuplexPipe() (a, b duplexPipe) {
	ar, aw := io.Pipe() // a escreve aqui, b le
	br, bw := io.Pipe() // b escreve aqui, a le
	a = duplexPipe{r: br, w: aw, closeR: br.Close}
	b = duplexPipe{r: ar, w: bw, closeR: ar.Close}
	return a, b
}
