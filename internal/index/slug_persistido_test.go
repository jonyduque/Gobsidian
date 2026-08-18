package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/parser"
)

// construirPorCache monta o indice pelo TERCEIRO caminho de producao: gravar o
// cache de metadados e recarrega-lo. E o caminho do boot quente, e e o que ja
// escondeu uma divergencia — DocLength media 5 construido e 10 recarregado.
func construirPorCache(t *testing.T, root string) *index.Index {
	t.Helper()
	origem := construirPorBuild(t, root)

	cacheDir := t.TempDir()
	if err := index.SaveIndexCache(context.Background(), cacheDir, root, origem); err != nil {
		t.Fatalf("SaveIndexCache: %v", err)
	}
	recarregado, _, err := index.LoadIndexCache(context.Background(), cacheDir, root)
	if err != nil {
		t.Fatalf("LoadIndexCache: %v", err)
	}
	if recarregado == nil {
		t.Fatal("LoadIndexCache devolveu indice nulo")
	}
	return recarregado
}

func escreverFixtureDeHeadings(t *testing.T, root string) {
	t.Helper()
	// Acento, pontuacao, caixa, e o caso do '#' final — que ja custou caro e
	// tem tratamento proprio em parseATXHeading.
	arquivos := map[string]string{
		"a.md": "# Capítulo 118\n\ntexto\n\n## Artigo 5º — parágrafo único\n\nmais\n",
		"b.md": "## Notas sobre C#\n\ntexto\n\n### Seção: Ação & Órgão\n\nmais\n",
		"c.md": "# Titulo Simples\n\ntexto\n\n## outro em minuscula\n\nmais\n",
	}
	for nome, conteudo := range arquivos {
		if err := os.WriteFile(filepath.Join(root, nome), []byte(conteudo), 0644); err != nil {
			t.Fatalf("escrevendo %s: %v", nome, err)
		}
	}
}

// TestSlugPersistidoBateComORecomputado guarda o unico risco de trocar
// parser.Slug(h.Text) por h.Slug: um Heading que chegue ao indice com Slug vazio
// ou desatualizado.
//
// Ele varre as TRES fontes de Heading — Build, cache recarregado e Replace —
// porque o defeito so aparece na fonte que ninguem lembrou de conferir. E a
// mesma licao do DocLength construido contra recarregado.
func TestSlugPersistidoBateComORecomputado(t *testing.T) {
	root := t.TempDir()
	escreverFixtureDeHeadings(t, root)

	fontes := []struct {
		nome string
		idx  *index.Index
	}{
		{"Build", construirPorBuild(t, root)},
		{"cache recarregado", construirPorCache(t, root)},
		{"Replace", construirPorReplace(t, root)},
	}

	for _, f := range fontes {
		t.Run(f.nome, func(t *testing.T) {
			caminhos := f.idx.NotePaths()
			if len(caminhos) != 3 {
				t.Fatalf("NotePaths = %d, quer 3 — a fonte nao construiu o cofre", len(caminhos))
			}
			vistos := 0
			for _, p := range caminhos {
				n, ok := f.idx.Get(p)
				if !ok {
					t.Fatalf("NotePaths devolveu %q e Get nao resolve", p)
				}
				for _, h := range n.Headings {
					vistos++
					if quer := parser.Slug(h.Text); h.Slug != quer {
						t.Errorf("%s: heading %q tem Slug %q, recomputado da %q",
							p, h.Text, h.Slug, quer)
					}
					if h.Slug == "" {
						t.Errorf("%s: heading %q tem Slug vazio", p, h.Text)
					}
				}
			}
			// Controle: sem esta linha, uma fonte que devolvesse zero headings
			// passaria verde, que e a forma exata do teste que nao pode falhar.
			if vistos != 6 {
				t.Fatalf("%d headings conferidos, quer 6", vistos)
			}
		})
	}
}
