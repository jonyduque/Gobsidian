package main

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/vaulttest"
)

// TestPonteNaoIniciaDaemonComDaemonAnteriorVivo e G2.3.
//
// O diretorio de runtime mudou em 2026-09-14. Um daemon de versao anterior,
// vivo no diretorio antigo, continua gravando o cache do cofre; subir outro no
// diretorio novo poria dois gravadores no mesmo cache. A ponte serve em
// processo ate o anterior sair, e diz o porque.
func TestPonteNaoIniciaDaemonComDaemonAnteriorVivo(t *testing.T) {
	semDaemonParaTeste(t)

	var iniciou atomic.Bool
	iniciarDaemonFn = func(config.Config) error {
		iniciou.Store(true)
		return nil
	}
	original := daemonAnteriorFn
	daemonAnteriorFn = func(string) (int, bool) { return 4242, true }
	t.Cleanup(func() { daemonAnteriorFn = original })

	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, nil))
	cfg := config.Config{VaultPath: filepath.Join(t.TempDir(), "nao-existe")}

	ctx, cancel := context.WithTimeout(context.Background(), vaulttest.Prazo)
	defer cancel()
	_ = servePonte(ctx, cfg, log)

	if iniciou.Load() {
		t.Fatal("a ponte iniciou um daemon com um daemon de versao anterior vivo no diretorio antigo")
	}
	saida := logBuf.String()
	for _, quer := range []string{"level=WARN", "motivo=daemon-de-versao-anterior", "pid=4242"} {
		if !strings.Contains(saida, quer) {
			t.Errorf("log sem %q:\n%s", quer, saida)
		}
	}
}
