package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/ipc"
	"github.com/jonyduque/Gobsidian/internal/vaulttest"
)

// TestPonteNaoIniciaDaemonOndeOSocketNaoConecta e G4.
//
// Medido em 2026-09-14: no processo que o Claude Desktop cria, o socket do
// daemon nao conecta, e a ponte subia um daemon por partida -- que morria
// logando `daemon nao pode abrir o socket`, 53 vezes so no cofre Estudo. Um
// daemon que nenhum processo deste contexto alcanca nao deve ser iniciado.
func TestPonteNaoIniciaDaemonOndeOSocketNaoConecta(t *testing.T) {
	semDaemonParaTeste(t)

	var iniciou atomic.Bool
	iniciarDaemonFn = func(config.Config) error {
		iniciou.Store(true)
		return nil
	}
	originalSonda := sondarSocketFn
	sondarSocketFn = func(string) error {
		return fmt.Errorf("%w: simulado como no processo do Claude Desktop", ipc.ErrDiretorioSemSocket)
	}
	t.Cleanup(func() { sondarSocketFn = originalSonda })

	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, nil))
	cfg := config.Config{VaultPath: filepath.Join(t.TempDir(), "nao-existe")}

	ctx, cancel := context.WithTimeout(context.Background(), vaulttest.Prazo)
	defer cancel()
	_ = servePonte(ctx, cfg, log)

	if iniciou.Load() {
		t.Fatal("a ponte iniciou um daemon num diretorio onde nem o proprio socket conecta")
	}
	saida := logBuf.String()
	if !strings.Contains(saida, "level=WARN") {
		t.Errorf("a queda saiu sem WARN:\n%s", saida)
	}
	if !strings.Contains(saida, "motivo=diretorio-sem-socket") {
		t.Errorf("a queda saiu sem motivo=diretorio-sem-socket:\n%s", saida)
	}
}
