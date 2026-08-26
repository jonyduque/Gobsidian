package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
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
