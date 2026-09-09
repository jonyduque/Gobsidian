package boot_test

import (
	"context"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/boot"
	"github.com/jonyd/gobsidian/internal/vaulttest"
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
