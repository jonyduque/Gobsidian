package index_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestResolvePathResolveAsTresFormas é o achado M6.
//
// O comentário de `ResolvePath` prometia caminho, nome de arquivo e alias. O
// corpo fazia DUAS: caminho exato e caminho insensível a maiúsculas. Nome e
// alias não resolviam, embora `resolveTarget` — que serve os wikilinks — saiba
// fazê-lo desde sempre. O efeito é que `[[STJ]]` dentro de uma nota resolvia e
// `note_read` com "STJ" não, apesar de `ResolvePath` ser a porta ÚNICA de todas
// as tools.
func TestResolvePathResolveAsTresFormas(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pasta/Acordao.md", "---\ntitle: Acordao\naliases: [STJ, Tribunal]\n---\n\ntexto\n")
	writeFile(t, root, "outra.md", "# Outra\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	casos := []struct {
		nome    string
		entrada string
		quer    vault.CanonicalPath
	}{
		{"caminho exato", "pasta/Acordao.md", "pasta/Acordao.md"},
		{"caminho com caixa diferente", "PASTA/acordao.MD", "pasta/Acordao.md"},
		{"nome de arquivo com extensao", "Acordao.md", "pasta/Acordao.md"},
		{"nome de arquivo sem extensao", "Acordao", "pasta/Acordao.md"},
		{"alias do frontmatter", "STJ", "pasta/Acordao.md"},
		{"segundo alias", "Tribunal", "pasta/Acordao.md"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			got, err := ix.ResolvePath(c.entrada)
			if err != nil {
				t.Fatalf("ResolvePath(%q): %v", c.entrada, err)
			}
			if got != c.quer {
				t.Errorf("ResolvePath(%q) = %q, queria %q", c.entrada, got, c.quer)
			}
		})
	}

	// O contrapeso: o que não existe continua não existindo.
	if _, err := ix.ResolvePath("NaoExiste"); !errors.Is(err, index.ErrPathNotFound) {
		t.Errorf("entrada inexistente: err = %v, queria ErrPathNotFound", err)
	}
}

// TestResolvePathAmbiguidadeENomeada fixa onde a ambiguidade é REAL.
//
// A varredura insensível a maiúsculas devolvia `ErrAmbiguousPath` num ramo em
// que ele era **inalcançável**: `lowerPath` tem chave única por construção. Ao
// trocar aquela varredura por um lookup direto, o erro ficaria órfão — e o
// tratamento dele no service viraria ramo morto.
//
// Ele vive agora onde a ambiguidade existe de verdade: dois arquivos com o mesmo
// NOME em pastas diferentes. Um wikilink desempata por proximidade, que é o
// comportamento documentado; uma chamada de tool não tem nota de origem, e
// escolher um dos dois devolveria um caminho arbitrário com cara de resposta.
func TestResolvePathAmbiguidadeENomeada(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a/Nota.md", "# Uma\n")
	writeFile(t, root, "b/Nota.md", "# Outra\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if _, err := ix.ResolvePath("Nota"); !errors.Is(err, index.ErrAmbiguousPath) {
		t.Errorf("dois homonimos: err = %v, queria ErrAmbiguousPath\n"+
			"escolher um deles sem nota de origem e devolver caminho arbitrario", err)
	}

	// E o caminho exato continua desambiguando: ele nunca pode perder para o nome.
	if got, err := ix.ResolvePath("a/Nota.md"); err != nil || got != "a/Nota.md" {
		t.Errorf("caminho exato = %q, err = %v; ele tem de vencer o nome ambiguo", got, err)
	}
}
