package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jonyduque/Gobsidian/internal/boot"
	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/ipc"
	"github.com/jonyduque/Gobsidian/internal/vaulttest"
)

// TestDaemonSaudaEnquantoMonta: o daemon saúda a ponte enquanto boot.Montar
// ainda constroi o indice.
//
// Medido em 2026-09-26 no log do daemon de Estudo: "daemon iniciado" as
// 13:29:59, "servidor pronto" (index_ms=37291) as 13:30:38, e as sessoes que
// esperavam a saudacao encerradas no mesmo milissegundo -- a ponte desistira aos
// 10 s e servira em processo, construindo o mesmo indice uma segunda vez. A
// montagem aqui fica presa ate o teste solta-la; a saudacao tem de sair antes,
// dentro do prazo curto que a ponte usa no primeiro dial.
func TestDaemonSaudaEnquantoMonta(t *testing.T) {
	cofre := t.TempDir()
	cfg := config.Config{VaultPath: cofre, CacheDir: t.TempDir()}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	soltar := make(chan struct{})
	montando := make(chan struct{})
	original := montarFn
	montarFn = func(ctx context.Context, _ config.Config, _ string, _ *slog.Logger) (*boot.Componentes, error) {
		close(montando)
		select {
		case <-soltar:
		case <-ctx.Done():
		}
		return nil, context.Canceled
	}
	t.Cleanup(func() { montarFn = original })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fim := make(chan error, 1)
	go func() { fim <- runDaemon(ctx, cfg, time.Hour, log) }()

	select {
	case <-montando:
	case <-time.After(vaulttest.Prazo):
		t.Fatal("runDaemon nao chegou a montagem")
	}

	conn, err := ipc.DialAndHandshake(ctx, cofre, false, 0, ipcDialTimeout)
	if err != nil {
		t.Errorf("sem saudacao com a montagem em curso: %v", err)
	} else {
		_ = conn.Close()
	}

	// A montagem falha: o daemon tem de encerrar, e nao ficar aceitando
	// conexoes que nunca vao ser servidas.
	close(soltar)
	select {
	case err := <-fim:
		if err == nil {
			t.Error("runDaemon devolveu nil com a montagem falhando")
		}
	case <-time.After(vaulttest.Prazo):
		t.Fatal("runDaemon nao voltou depois de a montagem falhar")
	}
}
