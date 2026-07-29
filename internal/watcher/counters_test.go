package watcher

import (
	"context"
	"io"
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
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
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

	notePath := filepath.Join(dir, "received.md")
	os.WriteFile(notePath, []byte("content"), 0644)

	for range 50 {
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

	notePath := filepath.Join(dir, "desktop.ini")
	os.WriteFile(notePath, []byte("content"), 0644)

	for range 50 {
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

	notePath := filepath.Join(dir, "processed.md")
	os.WriteFile(notePath, []byte("content"), 0644)

	for range 50 {
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

	notePath := filepath.Join(dir, "skipped.md")
	os.WriteFile(notePath, []byte("content"), 0644)

	for range 50 {
		if w.Stats().EventsProcessed > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	stats := w.Stats()
	if stats.EventsProcessed == 0 {
		t.Fatal("EventsProcessed = 0, want > 0")
	}

	canon, _ := vault.Canonicalize(dir, notePath)
	w.debounced <- []vault.CanonicalPath{canon}

	for range 50 {
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

func TestCounters_Reconciliations(t *testing.T) {
	w, cancel, _, _ := setupTestWatcher(t)
	defer cancel()

	w.fsWatcher.Errors <- fsnotify.ErrEventOverflow

	time.Sleep(100 * time.Millisecond)

	stats := w.Stats()
	if stats.Reconciliations == 0 {
		t.Error("Reconciliations = 0, want > 0")
	}
}

func TestCounters_DropReasons(t *testing.T) {
	w, cancel, dir, _ := setupTestWatcher(t)
	defer cancel()

	// 1. Chmod
	w.fsWatcher.Events <- fsnotify.Event{Name: filepath.Join(dir, "nota.md"), Op: fsnotify.Chmod}
	// 2. Outside vault
	w.fsWatcher.Events <- fsnotify.Event{Name: "D:\\fora\\nota.md", Op: fsnotify.Write}
	// 3. Excluded (.git)
	w.fsWatcher.Events <- fsnotify.Event{Name: filepath.Join(dir, ".git", "config"), Op: fsnotify.Write}
	// 4. Unknown op
	w.fsWatcher.Events <- fsnotify.Event{Name: filepath.Join(dir, "nota.md"), Op: 0}

	for range 50 {
		st := w.Stats()
		if st.DroppedByReason["chmod"] == 1 &&
			st.DroppedByReason["outside_vault"] == 1 &&
			st.DroppedByReason["excluded"] == 1 &&
			st.DroppedByReason["unknown_op"] == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	st := w.Stats()
	if st.DroppedByReason["chmod"] != 1 {
		t.Errorf("chmod drop count = %d, want 1", st.DroppedByReason["chmod"])
	}
	if st.DroppedByReason["outside_vault"] != 1 {
		t.Errorf("outside_vault drop count = %d, want 1", st.DroppedByReason["outside_vault"])
	}
	if st.DroppedByReason["excluded"] != 1 {
		t.Errorf("excluded drop count = %d, want 1", st.DroppedByReason["excluded"])
	}
	if st.DroppedByReason["unknown_op"] != 1 {
		t.Errorf("unknown_op drop count = %d, want 1", st.DroppedByReason["unknown_op"])
	}
	if st.EventsDropped != 4 {
		t.Errorf("EventsDropped sum = %d, want 4", st.EventsDropped)
	}
}

func TestCounters_ActiveState(t *testing.T) {
	w, cancel, _, _ := setupTestWatcher(t)

	if !w.Stats().Active {
		t.Error("Active = false, want true while running")
	}

	cancel()
	time.Sleep(100 * time.Millisecond)

	if w.Stats().Active {
		t.Error("Active = true, want false after cancel")
	}
}

func TestCounters_Coalesced(t *testing.T) {
	w, cancel, dir, _ := setupTestWatcher(t)
	defer cancel()

	canon, _ := vault.Canonicalize(dir, filepath.Join(dir, "nota.md"))
	in := make(chan Event, 10)
	out := make(chan []vault.CanonicalPath, 10)

	ctx, cancelDebounce := context.WithCancel(context.Background())
	defer cancelDebounce()

	go Debounce(ctx, in, out, 50*time.Millisecond, slog.Default(), &w.coalesced)

	// Send duplicate events to dirty set
	in <- Event{Path: canon, Op: OpCreate}
	in <- Event{Path: canon, Op: OpWrite}

	for range 50 {
		if w.Stats().EventsCoalesced > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if w.Stats().EventsCoalesced != 1 {
		t.Errorf("EventsCoalesced = %d, want 1", w.Stats().EventsCoalesced)
	}
}

func TestCounters_ReconciledUpdatedAndRemoved(t *testing.T) {
	w, cancel, dir, idx := setupTestWatcher(t)
	defer cancel()

	v, _ := vault.New(dir)
	os.WriteFile(filepath.Join(dir, "n1.md"), []byte("content"), 0644)

	// Trigger reconcile via channel
	w.reconcile <- struct{}{}

	for range 50 {
		st := w.Stats()
		if st.ReconciledUpdated > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	st := w.Stats()
	if st.ReconciledUpdated != 1 {
		t.Errorf("ReconciledUpdated = %d, want 1", st.ReconciledUpdated)
	}

	// Delete file from disk without telling index
	os.Remove(filepath.Join(dir, "n1.md"))

	w.reconcile <- struct{}{}

	for range 50 {
		st := w.Stats()
		if st.ReconciledRemoved > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	st = w.Stats()
	if st.ReconciledRemoved != 1 {
		t.Errorf("ReconciledRemoved = %d, want 1", st.ReconciledRemoved)
	}
	_ = idx
	_ = v
}
