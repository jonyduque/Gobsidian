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
	for _, path := range idx.NotePaths() {
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
	for _, target := range idx.NotePaths() {
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

// A invariante de coerencia acima nao consegue pegar estado OBSOLETO: um link
// que ainda afirma resolver para uma nota apagada, com o backlink intacto, e
// perfeitamente coerente consigo mesmo. Ela detecta inconsistencia, nunca
// desatualizacao.
//
// Foi assim que reprocessLinksLocked ficou sustentando a corretude de Remove
// sem nenhum teste: apagar a chamada deixava a suite inteira verde, enquanto
// remover uma nota passava a deixar todo link para ela dizendo "ok", apontando
// para arquivo que nao existe mais. vault_stats reportaria zero links quebrados
// sobre um cofre que perdeu uma nota.
//
// Este teste afirma a TRANSICAO de estado, que e o que falta.
func TestLinkStateFollowsRemoveAndRecreate(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "origem.md", "# Origem\n\n[[alvo]]\n")
	writeFile(t, root, "alvo.md", "# Alvo\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	estado := func() (index.LinkState, int) {
		t.Helper()
		n, ok := idx.Get("origem.md")
		if !ok {
			t.Fatal("origem.md ausente do indice")
		}
		if len(n.Links) != 1 {
			t.Fatalf("links = %d, quer 1", len(n.Links))
		}
		return n.Links[0].State, len(idx.Backlinks("alvo.md"))
	}

	if st, bl := estado(); st != index.LinkOK || bl != 1 {
		t.Fatalf("inicial: state=%v backlinks=%d, quer ok e 1", st, bl)
	}

	if err := os.Remove(filepath.Join(root, "alvo.md")); err != nil {
		t.Fatalf("os.Remove: %v", err)
	}
	idx.Remove("alvo.md")

	if st, bl := estado(); st != index.LinkTargetMissing || bl != 0 {
		t.Errorf("depois de remover: state=%v backlinks=%d, quer target_missing e 0 — "+
			"o link continua afirmando resolver para uma nota que nao existe mais", st, bl)
	}

	writeFile(t, root, "alvo.md", "# Alvo de volta\n")
	if err := idx.Replace(context.Background(), v, "alvo.md"); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	if st, bl := estado(); st != index.LinkOK || bl != 1 {
		t.Errorf("depois de recriar: state=%v backlinks=%d, quer ok e 1 — "+
			"o link nao voltou a resolver quando o alvo reapareceu", st, bl)
	}
}
