package parser

import "testing"

// Link Markdown com ancora: o parser separava so o wikilink, e o indice
// procurava a nota "b.md#Sec" — que nao existe — e contava alvo ausente.
// Medido em 2026-09-06 em cofres reais: 267 alvos comecando com "#" em um
// cofre, 372 em outro, todos contados como broken_links.
func TestMarkdownLinkSeparaAncora(t *testing.T) {
	casos := []struct {
		nome, src, target, anchor string
	}{
		{"nota e heading", "[x](b.md#Sec)", "b.md", "Sec"},
		{"so ancora", "[x](#Topo)", "", "Topo"},
		{"ancora percent-encoded", "[x](b.md#Se%C3%A7%C3%A3o)", "b.md", "Seção"},
		{"embed markdown", "![x](b.md#Sec)", "b.md", "Sec"},
		{"sem ancora fica igual", "[x](b.md)", "b.md", ""},
		{"URL com fragmento: Target sem fragmento, esquema intacto", "[x](https://ex.com/p#f)", "https://ex.com/p", "f"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			doc := Parse([]byte(c.src))
			if len(doc.Links) != 1 {
				t.Fatalf("links = %d, queria 1", len(doc.Links))
			}
			l := doc.Links[0]
			if l.Target != c.target || l.Anchor != c.anchor {
				t.Errorf("Target=%q Anchor=%q, queria %q/%q", l.Target, l.Anchor, c.target, c.anchor)
			}
		})
	}
}

// O '#' escapado como %23 faz parte do NOME do arquivo, e nao separa ancora.
// E por isso que a separacao acontece antes de PercentDecode: decodificar
// primeiro transformaria "C%23.md" em "C#.md" e o '#' viraria separador de uma
// ancora que ninguem escreveu.
func TestMarkdownLinkNaoSeparaNoPercent23(t *testing.T) {
	doc := Parse([]byte("[x](C%23.md)"))
	if len(doc.Links) != 1 {
		t.Fatalf("links = %d, queria 1", len(doc.Links))
	}
	if got := doc.Links[0]; got.Target != "C#.md" || got.Anchor != "" {
		t.Errorf("Target=%q Anchor=%q, queria %q/%q", got.Target, got.Anchor, "C#.md", "")
	}
}
