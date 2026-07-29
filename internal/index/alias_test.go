package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func TestAliasSurvivesReplaceAndRemove(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "---\naliases: [STJ]\n---\n# Nota A\n")
	writeFile(t, root, "b.md", "# Nota B\n\n[[STJ]]\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()
	ctx := context.Background()
	if err := idx.Build(ctx, v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// 1. After Build, b.md link [[STJ]] resolves to a.md with state LinkOK
	bNote, ok := idx.Get("b.md")
	if !ok || len(bNote.Links) != 1 {
		t.Fatalf("b.md links count = %d, want 1 (ok=%v)", len(bNote.Links), ok)
	}
	if bNote.Links[0].Resolved != "a.md" || bNote.Links[0].State != index.LinkOK {
		t.Fatalf("after Build: resolved=%q, state=%v; want 'a.md', LinkOK", bNote.Links[0].Resolved, bNote.Links[0].State)
	}

	// 2. After Replace(a.md), b.md link [[STJ]] should still resolve to a.md
	if err := idx.Replace(ctx, v, "a.md"); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	bNote, ok = idx.Get("b.md")
	if !ok || len(bNote.Links) != 1 {
		t.Fatalf("b.md links count = %d, want 1 (ok=%v)", len(bNote.Links), ok)
	}
	if bNote.Links[0].Resolved != "a.md" || bNote.Links[0].State != index.LinkOK {
		t.Fatalf("after Replace: resolved=%q, state=%v; want 'a.md', LinkOK", bNote.Links[0].Resolved, bNote.Links[0].State)
	}

	// 3. After Remove(a.md), b.md link [[STJ]] must resolve to "" with state LinkTargetMissing
	if err := os.Remove(filepath.Join(root, "a.md")); err != nil {
		t.Fatalf("Remove file: %v", err)
	}
	idx.Remove("a.md")

	bNote, ok = idx.Get("b.md")
	if !ok || len(bNote.Links) != 1 {
		t.Fatalf("b.md links count = %d, want 1 (ok=%v)", len(bNote.Links), ok)
	}
	if bNote.Links[0].Resolved != "" || bNote.Links[0].State != index.LinkTargetMissing {
		t.Fatalf("after Remove: resolved=%q, state=%v; want '', LinkTargetMissing", bNote.Links[0].Resolved, bNote.Links[0].State)
	}
}
