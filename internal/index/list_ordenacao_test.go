package index_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestListOrdenaPorTamanho cobre o comparador do achado P4.
//
// O critério de tamanho era `int(a.Size - b.Size)`. Em build 32-bit a subtração
// transborda e devolve a ordem INVERTIDA, em silêncio — nenhum erro, só uma
// lista fora de ordem. `cmp.Compare` é o idioma e não transborda.
//
// O teste também é o que protege a outra metade do P4: o `strings.ToLower(q.Sort)`
// saiu de DENTRO do comparador, e uma hasteação errada mudaria o critério.
func TestListOrdenaPorTamanho(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pequena.md", "# P\n")
	writeFile(t, root, "media.md", "# M\n\n"+strings.Repeat("x", 500)+"\n")
	writeFile(t, root, "grande.md", "# G\n\n"+strings.Repeat("y", 5000)+"\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	casos := []struct {
		ordem string
		quer  []vault.CanonicalPath
	}{
		{"asc", []vault.CanonicalPath{"pequena.md", "media.md", "grande.md"}},
		{"desc", []vault.CanonicalPath{"grande.md", "media.md", "pequena.md"}},
	}
	for _, c := range casos {
		t.Run(c.ordem, func(t *testing.T) {
			notes, total := ix.List(index.Query{Sort: "size", Order: c.ordem})
			if total != 3 {
				t.Fatalf("total = %d, queria 3", total)
			}
			for i, n := range notes {
				if n.Path != c.quer[i] {
					got := make([]vault.CanonicalPath, len(notes))
					for j, x := range notes {
						got[j] = x.Path
					}
					t.Fatalf("ordem = %v, queria %v", got, c.quer)
				}
			}
		})
	}

	// O critério vem de q.Sort com a caixa que o cliente mandou: a hasteação do
	// ToLower não pode ter tornado o campo sensível a maiúsculas.
	notes, _ := ix.List(index.Query{Sort: "SIZE", Order: "ASC"})
	if len(notes) != 3 || notes[0].Path != "pequena.md" {
		t.Errorf("Sort=\"SIZE\" nao foi reconhecido: o ToLower icado mudou o criterio")
	}
}
