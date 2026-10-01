package boot_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyduque/Gobsidian/internal/boot"
	"github.com/jonyduque/Gobsidian/internal/vault"
	"github.com/jonyduque/Gobsidian/internal/vaulttest"
)

func TestMontarDevolveServicoPronto(t *testing.T) {
	_, cfg := cofreDeTeste(t)
	cfg.EagerSearch = true
	cfg.DebounceMS = 50
	ctx, cancel := context.WithCancel(context.Background())
	c, err := boot.Montar(ctx, cfg, "em-processo", logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	if c.Service == nil || c.Index == nil || c.Inverted == nil || c.Watcher == nil || c.Vault == nil {
		t.Fatalf("Componentes incompleto: %+v", c)
	}
	if c.Index.NoteCount() != 2 {
		t.Fatalf("NoteCount = %d, quero 2", c.Index.NoteCount())
	}
	cancel()
	_ = c.Watcher.Close()
	feito := make(chan struct{})
	go func() { c.Esperar(); close(feito) }()
	select {
	case <-feito:
	case <-time.After(vaulttest.Prazo):
		t.Fatal("Esperar nao voltou depois de cancelar ctx e fechar o watcher")
	}
}

// TestMontarVarreTemporarioVelhoDoCache: o boot de serve e do daemon e quem
// limpa o diretorio de cache, pela regra de idade de
// vault.VarrerTemporariosAntigos. Sem esta chamada a funcao existiria e o lixo
// continuaria la -- os 230 MB de Estudo medidos em 2026-10-01.
func TestMontarVarreTemporarioVelhoDoCache(t *testing.T) {
	_, cfg := cofreDeTeste(t)
	cfg.DebounceMS = 50
	if err := os.MkdirAll(cfg.CacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	velho := filepath.Join(cfg.CacheDir, vault.TempFilePrefix+"orfao")
	if err := os.WriteFile(velho, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	antigo := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(velho, antigo, antigo); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	c, err := boot.Montar(ctx, cfg, "em-processo", logSilencioso())
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	_ = c.Watcher.Close()
	c.Esperar()

	if _, err := os.Stat(velho); !os.IsNotExist(err) {
		t.Errorf("o temporario de 48 h no cache sobreviveu ao boot (stat: %v)", err)
	}
}
