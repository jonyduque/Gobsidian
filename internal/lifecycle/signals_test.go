package lifecycle_test

import (
	"context"
	"io"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/jonyduque/Gobsidian/internal/lifecycle"
)

func TestSignalCancelsContext(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })

	ctx, lc := lifecycle.New(context.Background(), lifecycle.Options{Stdin: pr})

	// Da tempo de o handler ser instalado antes de disparar o sinal.
	time.Sleep(100 * time.Millisecond)

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess: %v", err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		t.Skipf("sinal nao entregavel nesta plataforma: %v", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context nao foi cancelado apos SIGTERM")
	}

	if got := lc.Reason(); got != "signal" {
		t.Errorf("Reason() = %q, quer %q", got, "signal")
	}
	lc.Wait()
}

// TestSinalRealAntesDoLifecycle: o mesmo de TestSinalAntesDoLifecycleNaoSePerde
// (sinais_cedo_test.go), com um SIGTERM de verdade, mandado ANTES de New. Sem
// ArmarSinais, o sinal mata o binario de teste pela acao padrao -- a forma do
// "encerrou sem registrar reason=" do gate de orfaos. Pula no Windows, onde o
// processo nao pode mandar sinal a si mesmo sem derrubar o go test.
func TestSinalRealAntesDoLifecycle(t *testing.T) {
	lifecycle.ArmarSinais()

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess: %v", err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		t.Skipf("sinal nao entregavel nesta plataforma: %v", err)
	}
	// O sinal ja foi entregue ao canal armado; o lifecycle nasce depois.
	time.Sleep(100 * time.Millisecond)

	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })
	ctx, lc := lifecycle.New(context.Background(), lifecycle.Options{Stdin: pr})

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("o SIGTERM recebido antes de New se perdeu")
	}
	if got := lc.Reason(); got != "signal" {
		t.Errorf("Reason() = %q, quer \"signal\"", got)
	}
	lc.Wait()
}
