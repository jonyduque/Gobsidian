package index_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/index"
	"github.com/jonyduque/Gobsidian/internal/vault"
)

// Link de ancora aponta para a PROPRIA nota, e nao para lugar nenhum.
//
// As tres formas — "[x](b.md#Sec)", "[x](#Topo)" e "[[#Topo]]" — caiam em
// LinkTargetMissing e entravam na contagem de broken_links, que o PRD chama de
// principal sinal de saude do cofre. Medido em 2026-09-06 em cofres reais: 267
// alvos comecando com "#" num deles e 372 em outro, todos falsos positivos.
func TestAncoraNaMesmaNota(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "b.md", "# Sec\n")
	writeFile(t, root, "a.md", strings.Join([]string{
		"# Topo",
		"[x](b.md#Sec)", // 0: outra nota, ancora existente
		"[x](#Topo)",    // 1: propria nota, ancora existente
		"[[#Topo]]",     // 2: idem, em wikilink
		"![[#Topo]]",    // 3: idem, em embed — a terceira forma que o
		//                     comentario de resolveTarget afirma
		"[x](#Nada)", // 4: propria nota, ancora inexistente
		"[[#Nada]]",  // 5: idem, em wikilink
		"[x]()",      // 6: contrapeso — sem alvo E sem ancora
	}, "\n\n")+"\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	note, ok := idx.Get("a.md")
	if !ok {
		t.Fatal("a.md ausente")
	}
	if len(note.Links) != 7 {
		t.Fatalf("links = %d, quer 7: %+v", len(note.Links), note.Links)
	}

	want := []struct {
		resolved vault.CanonicalPath
		state    index.LinkState
	}{
		{"b.md", index.LinkOK},
		{"a.md", index.LinkOK},
		{"a.md", index.LinkOK},
		{"a.md", index.LinkOK},
		{"a.md", index.LinkAnchorMissing},
		{"a.md", index.LinkAnchorMissing},
		{"", index.LinkTargetMissing},
	}

	for i, w := range want {
		got := note.Links[i]
		if got.Resolved != w.resolved || got.State != w.state {
			t.Errorf("link %d (%q#%q): Resolved=%q State=%v, quer %q/%v",
				i, got.Target, got.Anchor, got.Resolved, got.State, w.resolved, w.state)
		}
	}
}
