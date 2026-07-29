package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

func TestMoveNote_EquivalentToRemoveReplace(t *testing.T) {
	// Setup 2 identical vaults
	root1 := t.TempDir()
	root2 := t.TempDir()

	contentA := "---\naliases:\n  - STJ\ntags:\n  - tag1\n---\n# Nota A\n\nLink para [[c]]\n"
	contentC := "# Nota C\n\nLink para [[STJ]]\n"

	writeFileHelper(t, root1, "a.md", contentA)
	writeFileHelper(t, root1, "c.md", contentC)

	writeFileHelper(t, root2, "a.md", contentA)
	writeFileHelper(t, root2, "c.md", contentC)

	v1, err := vault.New(root1)
	if err != nil {
		t.Fatalf("vault.New v1: %v", err)
	}
	v2, err := vault.New(root2)
	if err != nil {
		t.Fatalf("vault.New v2: %v", err)
	}

	idx1 := New()
	idx2 := New()
	ctx := context.Background()

	if err := idx1.Build(ctx, v1); err != nil {
		t.Fatalf("idx1.Build: %v", err)
	}
	if err := idx2.Build(ctx, v2); err != nil {
		t.Fatalf("idx2.Build: %v", err)
	}

	// Rename on disk in both
	if err := os.Rename(filepath.Join(root1, "a.md"), filepath.Join(root1, "b.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root2, "a.md"), filepath.Join(root2, "b.md")); err != nil {
		t.Fatal(err)
	}

	// Path 1: MoveNote(v1, "a.md", "b.md")
	idx1.MoveNote(v1, "a.md", "b.md")

	// Path 2: Remove("a.md") + Replace(ctx, v2, "b.md")
	idx2.Remove("a.md")
	if err := idx2.Replace(ctx, v2, "b.md"); err != nil {
		t.Fatalf("idx2.Replace: %v", err)
	}

	// Compare structure by structure
	// 1. notes
	note1, ok1 := idx1.notes["b.md"]
	note2, ok2 := idx2.notes["b.md"]
	if !ok1 || !ok2 {
		t.Fatalf("notes structure mismatch: ok1=%v, ok2=%v", ok1, ok2)
	}
	if note1.Path != note2.Path || note1.Title != note2.Title || note1.Size != note2.Size {
		t.Errorf("notes mismatch: note1=%+v, note2=%+v", note1, note2)
	}
	if _, ok := idx1.notes["a.md"]; ok {
		t.Errorf("a.md still present in idx1.notes")
	}

	// 2. lowerPath
	if p, ok := idx1.lowerPath["b.md"]; !ok || p != "b.md" {
		t.Errorf("lowerPath mismatch idx1: %v", p)
	}

	// 3. byName
	if paths := idx1.byName["b.md"]; len(paths) != 1 || paths[0] != "b.md" {
		t.Errorf("byName mismatch idx1: %v", paths)
	}

	// 4. tags
	if paths := idx1.tags["tag1"]; len(paths) != 1 || paths[0] != "b.md" {
		t.Errorf("tags mismatch idx1: %v", paths)
	}

	// 5. byAlias
	if paths := idx1.byAlias["stj"]; len(paths) != 1 || paths[0] != "b.md" {
		t.Errorf("byAlias mismatch idx1: %v", paths)
	}

	// 6. incoming backlinks
	if bls := idx1.backlinks["b.md"]; len(bls) != 1 || bls[0].From != "c.md" {
		t.Errorf("incoming backlinks mismatch idx1: %v", bls)
	}

	// 7. outgoing backlinks / c.md links resolution
	cNote1, _ := idx1.notes["c.md"]
	cNote2, _ := idx2.notes["c.md"]
	if len(cNote1.Links) != len(cNote2.Links) {
		t.Fatalf("cNote links count mismatch: %d vs %d", len(cNote1.Links), len(cNote2.Links))
	}
	for i := range cNote1.Links {
		if cNote1.Links[i].Resolved != cNote2.Links[i].Resolved {
			t.Errorf("link %d resolution mismatch: idx1=%s, idx2=%s", i, cNote1.Links[i].Resolved, cNote2.Links[i].Resolved)
		}
	}

	// 8. generation divergence (MoveNote = +1 generation, Remove+Replace = +2)
	if idx1.generation != 3 { // Build (+2) + MoveNote (+1) = 3
		t.Errorf("idx1 generation = %d, expected 3", idx1.generation)
	}
	if idx2.generation != 4 { // Build (+2) + Remove (+1) + Replace (+1) = 4
		t.Errorf("idx2 generation = %d, expected 4", idx2.generation)
	}
}

func TestMoveNote_UpdatesNotes(t *testing.T) {
	root := t.TempDir()
	writeFileHelper(t, root, "a.md", "# A\n")
	writeFileHelper(t, root, "b.md", "# B\n")
	v, _ := vault.New(root)
	idx := New()
	idx.Build(context.Background(), v)

	os.Rename(filepath.Join(root, "a.md"), filepath.Join(root, "b.md"))
	idx.MoveNote(v, "a.md", "b.md")

	if _, ok := idx.notes["a.md"]; ok {
		t.Errorf("notes structure survived mutation: a.md still present")
	}
	if _, ok := idx.notes["b.md"]; !ok {
		t.Errorf("notes structure: b.md not present")
	}
}

func TestMoveNote_UpdatesLowerPath(t *testing.T) {
	root := t.TempDir()
	writeFileHelper(t, root, "a.md", "# A\n")
	v, _ := vault.New(root)
	idx := New()
	idx.Build(context.Background(), v)

	os.Rename(filepath.Join(root, "a.md"), filepath.Join(root, "b.md"))
	idx.MoveNote(v, "a.md", "b.md")

	if p, ok := idx.lowerPath["a.md"]; ok {
		t.Errorf("lowerPath structure survived mutation: a.md -> %s", p)
	}
	if p, ok := idx.lowerPath["b.md"]; !ok || p != "b.md" {
		t.Errorf("lowerPath structure: b.md -> %s (ok=%v)", p, ok)
	}
}

func TestMoveNote_UpdatesByName(t *testing.T) {
	root := t.TempDir()
	writeFileHelper(t, root, "a.md", "# A\n")
	v, _ := vault.New(root)
	idx := New()
	idx.Build(context.Background(), v)

	os.Rename(filepath.Join(root, "a.md"), filepath.Join(root, "b.md"))
	idx.MoveNote(v, "a.md", "b.md")

	if paths := idx.byName["a.md"]; len(paths) != 0 {
		t.Errorf("byName structure survived mutation: a.md still in byName (%v)", paths)
	}
	if paths := idx.byName["b.md"]; len(paths) != 1 || paths[0] != "b.md" {
		t.Errorf("byName structure: b.md -> %v", paths)
	}
}

func TestMoveNote_UpdatesTags(t *testing.T) {
	root := t.TempDir()
	writeFileHelper(t, root, "a.md", "---\ntags:\n  - mylabel\n---\n# A\n")
	v, _ := vault.New(root)
	idx := New()
	idx.Build(context.Background(), v)

	os.Rename(filepath.Join(root, "a.md"), filepath.Join(root, "b.md"))
	idx.MoveNote(v, "a.md", "b.md")

	paths := idx.tags["mylabel"]
	if len(paths) != 1 || paths[0] != "b.md" {
		t.Errorf("tags structure survived mutation: tag mylabel -> %v", paths)
	}
}

func TestMoveNote_UpdatesByAlias(t *testing.T) {
	root := t.TempDir()
	writeFileHelper(t, root, "a.md", "---\naliases:\n  - STJ\n---\n# A\n")
	v, _ := vault.New(root)
	idx := New()
	idx.Build(context.Background(), v)

	os.Rename(filepath.Join(root, "a.md"), filepath.Join(root, "b.md"))
	idx.MoveNote(v, "a.md", "b.md")

	paths := idx.byAlias["stj"]
	if len(paths) != 1 || paths[0] != "b.md" {
		t.Errorf("byAlias structure survived mutation: alias STJ -> %v", paths)
	}
}

func TestMoveNote_UpdatesIncomingBacklinks(t *testing.T) {
	root := t.TempDir()
	writeFileHelper(t, root, "a.md", "---\naliases:\n  - STJ\n---\n# A\n")
	writeFileHelper(t, root, "c.md", "# C\n\nLink para [[STJ]]\n")
	v, _ := vault.New(root)
	idx := New()
	idx.Build(context.Background(), v)

	os.Rename(filepath.Join(root, "a.md"), filepath.Join(root, "b.md"))
	idx.MoveNote(v, "a.md", "b.md")

	if bls := idx.backlinks["a.md"]; len(bls) != 0 {
		t.Errorf("incoming backlinks survived mutation: a.md has backlinks %v", bls)
	}
	if bls := idx.backlinks["b.md"]; len(bls) != 1 || bls[0].From != "c.md" {
		t.Errorf("incoming backlinks: b.md backlinks = %v", bls)
	}
}

func TestMoveNote_UpdatesOutgoingBacklinks(t *testing.T) {
	root := t.TempDir()
	writeFileHelper(t, root, "a.md", "# A\n\nLink para [[c]]\n")
	writeFileHelper(t, root, "c.md", "# C\n")
	v, _ := vault.New(root)
	idx := New()
	idx.Build(context.Background(), v)

	os.Rename(filepath.Join(root, "a.md"), filepath.Join(root, "b.md"))
	idx.MoveNote(v, "a.md", "b.md")

	bls := idx.backlinks["c.md"]
	if len(bls) != 1 || bls[0].From != "b.md" {
		t.Errorf("outgoing backlinks survived mutation: c.md backlinks = %v", bls)
	}
}

func TestMoveNote_ReprocessesBrokenLinks(t *testing.T) {
	root := t.TempDir()
	writeFileHelper(t, root, "a.md", "# A\n")
	writeFileHelper(t, root, "d.md", "# D\n\nLink para [[a]]\n")
	v, _ := vault.New(root)
	idx := New()
	idx.Build(context.Background(), v)

	// Move a.md to b.md
	os.Rename(filepath.Join(root, "a.md"), filepath.Join(root, "b.md"))
	idx.MoveNote(v, "a.md", "b.md")

	// d.md's link to [[a]] must now be LinkTargetMissing because a.md was moved and no a.md exists anymore
	dNote, _ := idx.Get("d.md")
	if len(dNote.Links) != 1 || dNote.Links[0].State != LinkTargetMissing {
		t.Errorf("d.md link state = %v, expected LinkTargetMissing", dNote.Links[0].State)
	}
}

func TestMoveNote_StatFailureZerosModTime(t *testing.T) {
	root := t.TempDir()
	writeFileHelper(t, root, "a.md", "# A\n")
	v, _ := vault.New(root)
	idx := New()
	idx.Build(context.Background(), v)

	// Move without creating file on disk so os.Stat fails
	idx.MoveNote(v, "a.md", "nonexistent.md")

	n, ok := idx.notes["nonexistent.md"]
	if !ok {
		t.Fatal("nonexistent.md not found in notes")
	}
	if !n.ModTime.IsZero() {
		t.Errorf("expected zero ModTime on stat failure, got %v", n.ModTime)
	}
}

func writeFileHelper(t *testing.T, root, path, content string) {
	t.Helper()
	abs := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
