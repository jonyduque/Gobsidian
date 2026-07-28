package watcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/vault"
)

func TestWatcher(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dir := t.TempDir()

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	w, err := New(v, 10*time.Millisecond, log)
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

	// Verify event
	select {
	case evt := <-w.Events():
		if evt != "teste.md" {
			t.Errorf("got path %q, want 'teste.md'", evt)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
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

	// Verify events channel is closed
	closed := false
	for i := 0; i < 10; i++ {
		select {
		case _, ok := <-w.Events():
			if !ok {
				closed = true
				break
			}
		default:
		}
		if closed {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !closed {
		t.Error("Events channel not closed")
	}
}
