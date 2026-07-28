package watcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func TestWatcher(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dir := t.TempDir()

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()

	w, err := New(v, idx, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errc := make(chan error, 1)
	go func() {
		errc <- w.Run(ctx)
	}()

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Create a note
	notePath := filepath.Join(dir, "teste.md")
	if err := os.WriteFile(notePath, []byte("conteudo"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Verify event via index
	found := false
	canon, _ := vault.Canonicalize(dir, notePath)
	for i := 0; i < 50; i++ {
		if _, ok := idx.Get(canon); ok {
			found = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !found {
		t.Fatal("timeout waiting for index to update with 'teste.md'")
	}

	// Shutdown test
	cancel()
	select {
	case err := <-errc:
		if err != context.Canceled {
			t.Errorf("Run error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for Run to exit")
	}

	if err := w.Close(); err != nil {
		t.Errorf("Close() = %v", err)
	}
}
