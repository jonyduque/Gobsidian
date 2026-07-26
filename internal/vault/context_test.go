package vault_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// TestWalkStopsOnPreCancelledContext confirms that Walk never invokes the
// callback and returns ctx.Err() when the context is already cancelled
// before the walk starts.
func TestWalkStopsOnPreCancelledContext(t *testing.T) {
	root := t.TempDir()
	const total = 300
	for i := 0; i < total; i++ {
		writeFile(t, root, fmt.Sprintf("dir%d/note%d.md", i, i), "# x\n")
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	visited := 0
	walkErr := v.Walk(ctx, func(vault.Entry) error {
		visited++
		return nil
	})

	if !errors.Is(walkErr, context.Canceled) {
		t.Fatalf("Walk err = %v, quer context.Canceled", walkErr)
	}
	if visited != 0 {
		t.Fatalf("visited = %d, quer 0 (contexto ja cancelado antes de comecar)", visited)
	}
}

// TestWalkStopsMidwayOnCancel confirms that cancelling the context from
// inside the callback actually aborts the walk before it reaches every
// entry, rather than merely being observed and ignored.
func TestWalkStopsMidwayOnCancel(t *testing.T) {
	root := t.TempDir()
	const total = 300
	for i := 0; i < total; i++ {
		writeFile(t, root, fmt.Sprintf("dir%d/note%d.md", i, i), "# x\n")
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	const stopAfter = 10
	visited := 0
	walkErr := v.Walk(ctx, func(vault.Entry) error {
		visited++
		if visited == stopAfter {
			cancel()
		}
		return nil
	})

	if !errors.Is(walkErr, context.Canceled) {
		t.Fatalf("Walk err = %v, quer context.Canceled", walkErr)
	}
	if visited >= total {
		t.Fatalf("visited = %d, quer bem menos que %d (walk deveria ter parado cedo)", visited, total)
	}
	if visited < stopAfter {
		t.Fatalf("visited = %d, quer >= %d", visited, stopAfter)
	}
}
