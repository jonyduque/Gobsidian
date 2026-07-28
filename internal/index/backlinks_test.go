package index_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Invariante central do indice: para toda nota N e todo link L em N que
// resolve para M, existe um backlink de N em M. E o inverso: todo backlink
// registrado corresponde a um link real.
func TestBacklinkInvariantUnderMutation(t *testing.T) {
	root := t.TempDir()
	for i := range 20 {
		writeFile(t, root, fmt.Sprintf("n%02d.md", i),
			fmt.Sprintf("# N%d\n\n[[n%02d]]\n[[n%02d]]\n", i, (i+1)%20, (i+7)%20))
	}

	v, _ := vault.New(root)
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	assertInvariant(t, idx)

	// Sequencia de mutacoes: modificar, remover, recriar.
	ctx := context.Background()

	writeFile(t, root, "n05.md", "# N5 sem links\n")
	if err := idx.Replace(ctx, v, "n05.md"); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	assertInvariant(t, idx)

	if err := os.Remove(filepath.Join(root, "n07.md")); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	idx.Remove("n07.md")
	assertInvariant(t, idx)

	writeFile(t, root, "n07.md", "# N7 de volta\n\n[[n00]]\n")
	if err := idx.Replace(ctx, v, "n07.md"); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	assertInvariant(t, idx)
}

func assertInvariant(t *testing.T, idx *index.Index) {
	t.Helper()

	// Direcao 1: todo link resolvido tem backlink correspondente.
	for _, path := range idx.Paths() {
		note, _ := idx.Get(path)
		for _, link := range note.Links {
			if link.Resolved == "" {
				continue
			}
			found := false
			for _, bl := range idx.Backlinks(link.Resolved) {
				if bl.From == path {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("link %s -> %s sem backlink correspondente", path, link.Resolved)
			}
		}
	}

	// Direcao 2: todo backlink corresponde a um link real.
	for _, target := range idx.Paths() {
		for _, bl := range idx.Backlinks(target) {
			origin, ok := idx.Get(bl.From)
			if !ok {
				t.Errorf("backlink de %s para %s, mas a origem nao esta no indice", bl.From, target)
				continue
			}
			found := false
			for _, link := range origin.Links {
				if link.Resolved == target {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("backlink fantasma: %s -> %s", bl.From, target)
			}
		}
	}
}
