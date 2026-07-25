package lifecycle_test

import (
	"context"
	"io"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/lifecycle"
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
