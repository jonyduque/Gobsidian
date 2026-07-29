package watcher_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/watcher"
)

func TestCorrelateRenames(t *testing.T) {
	tmp := t.TempDir()
	v, _ := vault.New(tmp)
	idx := index.New()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Setup: initial files
	content := []byte("some content")
	emptyContent := []byte("")

	os.WriteFile(filepath.Join(tmp, "note1.md"), content, 0644)
	os.WriteFile(filepath.Join(tmp, "empty1.md"), emptyContent, 0644)

	idx.Build(context.Background(), v)

	// Action: simulate rename note1.md -> note2.md
	os.Rename(filepath.Join(tmp, "note1.md"), filepath.Join(tmp, "note2.md"))
	// Simulate rename empty1.md -> empty2.md
	os.Rename(filepath.Join(tmp, "empty1.md"), filepath.Join(tmp, "empty2.md"))
	// Simulate modified file
	os.WriteFile(filepath.Join(tmp, "note3.md"), []byte("initial"), 0644)
	idx.Build(context.Background(), v)
	os.Rename(filepath.Join(tmp, "note3.md"), filepath.Join(tmp, "note4.md"))
	os.WriteFile(filepath.Join(tmp, "note4.md"), []byte("modified"), 0644)

	batch := []vault.CanonicalPath{
		"note1.md", "note2.md",
		"empty1.md", "empty2.md",
		"note3.md", "note4.md",
	}

	renames, nonRenames := watcher.CorrelateRenames(batch, v, idx, log)

	if len(renames) != 1 {
		t.Fatalf("Expected 1 rename, got %d", len(renames))
	}
	if renames[0].From != "note1.md" || renames[0].To != "note2.md" {
		t.Errorf("Unexpected rename correlation: %v", renames[0])
	}

	// Ensure empty and modified files fell back to nonRenames
	nonRenamesMap := make(map[vault.CanonicalPath]bool)
	for _, p := range nonRenames {
		nonRenamesMap[p] = true
	}

	expectedNonRenames := []vault.CanonicalPath{"empty1.md", "empty2.md", "note3.md", "note4.md"}
	for _, p := range expectedNonRenames {
		if !nonRenamesMap[p] {
			t.Errorf("Expected %s in nonRenames, not found", p)
		}
	}
}
