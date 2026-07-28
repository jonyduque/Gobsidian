package watcher

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/vault"
)

func TestDebounce_Coalescence(t *testing.T) {
	in := make(chan Event, 100)
	out := make(chan vault.CanonicalPath, 100)
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Debounce(ctx, in, out, 50*time.Millisecond, log)
	}()

	// 10 events on same path
	path1 := vault.CanonicalPath("file1.md")
	for i := 0; i < 10; i++ {
		in <- Event{Path: path1, Op: OpWrite}
	}

	// 2 distinct paths
	path2 := vault.CanonicalPath("file2.md")
	path3 := vault.CanonicalPath("file3.md")
	in <- Event{Path: path2, Op: OpWrite}
	in <- Event{Path: path3, Op: OpWrite}

	// Wait for tick
	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()
	close(out)

	var results []vault.CanonicalPath
	for p := range out {
		results = append(results, p)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 paths (1 coalesced + 2 distinct), got %d: %v", len(results), results)
	}

	counts := make(map[vault.CanonicalPath]int)
	for _, p := range results {
		counts[p]++
	}

	if counts[path1] != 1 {
		t.Errorf("expected path1 exactly once, got %d", counts[path1])
	}
	if counts[path2] != 1 {
		t.Errorf("expected path2 exactly once, got %d", counts[path2])
	}
	if counts[path3] != 1 {
		t.Errorf("expected path3 exactly once, got %d", counts[path3])
	}
}

func TestDebounce_NoStarvation(t *testing.T) {
	in := make(chan Event, 100)
	out := make(chan vault.CanonicalPath, 100)
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		Debounce(ctx, in, out, 10*time.Millisecond, log)
	}()

	// Write continuously for 50ms, longer than the 10ms debounce tick
	path := vault.CanonicalPath("file_starve.md")
	start := time.Now()
	emitted := 0

	// consumer
	var cwg sync.WaitGroup
	cwg.Add(1)
	go func() {
		defer cwg.Done()
		for {
			select {
			case <-out:
				emitted++
			case <-ctx.Done():
				return
			}
		}
	}()

	for time.Since(start) < 50*time.Millisecond {
		in <- Event{Path: path, Op: OpWrite}
		time.Sleep(1 * time.Millisecond) // Continuous fast writes
	}

	cancel()
	wg.Wait()
	cwg.Wait()

	if emitted == 0 {
		t.Fatalf("Starvation detected: expected multiple emissions during continuous write, got %d", emitted)
	}
}
