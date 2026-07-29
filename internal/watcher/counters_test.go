package watcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func setupTestWatcher(t *testing.T) (*Watcher, context.CancelFunc, string, *index.Index) {
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
	go w.Run(ctx)

	// Wait for watcher to start
	time.Sleep(50 * time.Millisecond)

	return w, cancel, dir, idx
}

func TestCounters_EventsReceived(t *testing.T) {
	w, cancel, dir, _ := setupTestWatcher(t)
	defer cancel()

	// Write a file
	notePath := filepath.Join(dir, "received.md")
	os.WriteFile(notePath, []byte("content"), 0644)

	// Wait for processing
	for i := 0; i < 50; i++ {
		if w.Stats().EventsReceived > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	stats := w.Stats()
	if stats.EventsReceived == 0 {
		t.Error("EventsReceived = 0, want > 0")
	}
}

func TestCounters_EventsDropped(t *testing.T) {
	w, cancel, dir, _ := setupTestWatcher(t)
	defer cancel()

	// Create an excluded file that triggers dropped
	notePath := filepath.Join(dir, "desktop.ini")
	os.WriteFile(notePath, []byte("content"), 0644)

	for i := 0; i < 50; i++ {
		if w.Stats().EventsDropped > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	stats := w.Stats()
	if stats.EventsDropped == 0 {
		t.Error("EventsDropped = 0, want > 0")
	}
}

func TestCounters_EventsProcessed(t *testing.T) {
	w, cancel, dir, _ := setupTestWatcher(t)
	defer cancel()

	// Write a relevant file
	notePath := filepath.Join(dir, "processed.md")
	os.WriteFile(notePath, []byte("content"), 0644)

	for i := 0; i < 50; i++ {
		if w.Stats().EventsProcessed > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	stats := w.Stats()
	if stats.EventsProcessed == 0 {
		t.Error("EventsProcessed = 0, want > 0")
	}
}

func TestCounters_EventsSkipped(t *testing.T) {
	w, cancel, dir, _ := setupTestWatcher(t)
	defer cancel()

	// Write a relevant file
	notePath := filepath.Join(dir, "skipped.md")
	os.WriteFile(notePath, []byte("content"), 0644)

	for i := 0; i < 50; i++ {
		if w.Stats().EventsProcessed > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Verify it was processed
	stats := w.Stats()
	if stats.EventsProcessed == 0 {
		t.Fatal("EventsProcessed = 0, want > 0")
	}

	// Now force the debouncer to apply it again without file changes
	// We do this by feeding the canonical path directly into debounced channel
	// so Apply will pick it up, check stat vs index, and skip.
	canon, _ := vault.Canonicalize(dir, notePath)
	w.debounced <- []vault.CanonicalPath{canon}

	for i := 0; i < 50; i++ {
		if w.Stats().EventsSkipped > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	stats = w.Stats()
	if stats.EventsSkipped == 0 {
		t.Error("EventsSkipped = 0, want > 0")
	}
}

// Reconciliations isn't easily triggered without fsnotify overflowing,
// but we can simulate the channel trigger.
func TestCounters_Reconciliations(t *testing.T) {
	w, cancel, _, _ := setupTestWatcher(t)
	defer cancel()

	// Feed error channel with fsnotify.ErrEventOverflow
	w.fsWatcher.Errors <- fsnotify.ErrEventOverflow

	time.Sleep(100 * time.Millisecond)

	stats := w.Stats()
	if stats.Reconciliations == 0 {
		t.Error("Reconciliations = 0, want > 0")
	}
}
