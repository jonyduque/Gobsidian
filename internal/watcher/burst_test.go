package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func TestWatcher_Burst(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dir := t.TempDir()

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()

	w, err := New(v, idx, nil, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = w.Run(ctx)
	}()

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	count := 500
	for i := 0; i < count; i++ {
		path := filepath.Join(dir, fmt.Sprintf("note_%d.md", i))
		_ = os.WriteFile(path, []byte(fmt.Sprintf("note %d", i)), 0644)
	}

	// wait for processing
	for i := 0; i < 100; i++ {
		if idx.NoteCount() == count {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if idx.NoteCount() != count {
		t.Fatalf("esperava %d notas no indice, encontrou %d", count, idx.NoteCount())
	}

	var walkCount int
	_ = v.Walk(context.Background(), func(e vault.Entry) error {
		if e.IsNote {
			walkCount++
		}
		return nil
	})
	if idx.NoteCount() != walkCount {
		t.Fatalf("esperava que NoteCount (%d) correspondesse a v.Walk (%d)", idx.NoteCount(), walkCount)
	}
}
