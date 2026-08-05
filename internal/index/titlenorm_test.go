package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/text"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestTitleNormAcompanhaOTitulo guarda a divergencia que um campo derivado
// SEMPRE convida: Title muda num caminho e TitleNorm nao muda no outro.
//
// Cobre os tres caminhos que publicam Note: Build, Replace e MoveNote.
func TestTitleNormAcompanhaOTitulo(t *testing.T) {
	root := t.TempDir()
	escreve(t, root, "a.md", "# Prescrição Intercorrente\n\ncorpo\n")
	v, idx := cofreIndexado(t, root)

	confere := func(quando string, p vault.CanonicalPath) {
		t.Helper()
		n, ok := idx.Get(p)
		if !ok {
			t.Fatalf("%s: nota %q sumiu do indice", quando, p)
		}
		if quer := text.Normalize(n.Title); n.TitleNorm != quer {
			t.Errorf("%s: TitleNorm=%q, quer %q (Title=%q)",
				quando, n.TitleNorm, quer, n.Title)
		}
	}
	confere("apos Build", "a.md")

	escreve(t, root, "a.md", "# Execução Fiscal\n\ncorpo\n")
	if err := idx.Replace(context.Background(), v, "a.md"); err != nil {
		t.Fatal(err)
	}
	confere("apos Replace", "a.md")

	if err := os.Rename(filepath.Join(root, "a.md"), filepath.Join(root, "b.md")); err != nil {
		t.Fatal(err)
	}
	idx.MoveNote(v, "a.md", "b.md")
	confere("apos MoveNote", "b.md")
}

// cofreIndexado e um helper que cria um vault e o indexa.
func cofreIndexado(t *testing.T, root string) (*vault.Vault, *Index) {
	t.Helper()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}
	return v, idx
}

// escreve e um helper que escreve conteudo num arquivo.
func escreve(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
