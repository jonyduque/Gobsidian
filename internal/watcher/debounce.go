package watcher

import (
	"context"
	"log/slog"
	"time"

	"github.com/jonyd/gobsidian/internal/vault"
)

// Debounce reads events from 'in' and emits coalesced paths to 'out'.
// It uses a single ticker and a dirty set to coalesce multiple events on the same path.
func Debounce(ctx context.Context, in <-chan Event, out chan<- vault.CanonicalPath, window time.Duration, log *slog.Logger) {
	dirty := make(map[vault.CanonicalPath]struct{})

	if window <= 0 {
		window = 250 * time.Millisecond // default
	}

	ticker := time.NewTicker(window)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if len(dirty) > 0 {
				log.Debug("Debouncer context cancelled with pending paths", "pending_count", len(dirty))
			}
			return
		case evt, ok := <-in:
			if !ok {
				// input channel closed
				flush(ctx, dirty, out)
				return
			}
			dirty[evt.Path] = struct{}{}
		case <-ticker.C:
			flush(ctx, dirty, out)
		}
	}
}

// flush empties the dirty set into the output channel.
// It creates a new map instead of modifying the existing one, but doing it in-place is fine.
func flush(ctx context.Context, dirty map[vault.CanonicalPath]struct{}, out chan<- vault.CanonicalPath) {
	for path := range dirty {
		select {
		case out <- path:
			delete(dirty, path)
		case <-ctx.Done():
			return
		}
	}
}
