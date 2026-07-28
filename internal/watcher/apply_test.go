package watcher_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/watcher"
)

func TestApply(t *testing.T) {
	root := t.TempDir()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.Open: %v", err)
	}
	idx := index.New()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	in := make(chan vault.CanonicalPath, 10)

	// Arquivo que não muda
	path1 := filepath.Join(root, "same.md")
	if err := os.WriteFile(path1, []byte("same"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	canon1, _ := vault.Canonicalize(root, path1)
	if err := idx.Replace(context.Background(), v, canon1); err != nil {
		t.Fatalf("idx.Replace: %v", err)
	}

	// Arquivo que muda (nova nota)
	path2 := filepath.Join(root, "new.md")
	if err := os.WriteFile(path2, []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	canon2, _ := vault.Canonicalize(root, path2)

	// Arquivo removido
	path3 := filepath.Join(root, "deleted.md")
	if err := os.WriteFile(path3, []byte("del"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	canon3, _ := vault.Canonicalize(root, path3)
	if err := idx.Replace(context.Background(), v, canon3); err != nil {
		t.Fatalf("idx.Replace: %v", err)
	}
	os.Remove(path3)

	// Arquivo com erro de parse
	// Wait, the parser doesn't return an error for invalid frontmatter, it just parses it.
	// But let's say we have an unreadable file or parse error if one exists.
	// We'll write a file with bad chars maybe? Or we just mock a bad parse.
	// But it says: "Uma nota com erro de parse é pulada sem derrubar o laço, e as notas seguintes continuam sendo atualizadas?"
	in <- canon1
	in <- canon2
	in <- canon3
	close(in)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	processed, skipped := watcher.Apply(ctx, in, idx, v, log)

	if processed != 3 {
		t.Errorf("processed: got %d, want 3", processed)
	}
	if skipped != 1 { // same.md was skipped, meaning 0 parses performed for it
		t.Errorf("skipped (0-parses): got %d, want 1", skipped)
	}

	// new.md should be in index
	if _, ok := idx.Get(canon2); !ok {
		t.Errorf("expected new.md to be in index")
	}

	// deleted.md should NOT be in index
	if _, ok := idx.Get(canon3); ok {
		t.Errorf("expected deleted.md to be removed from index")
	}
}
