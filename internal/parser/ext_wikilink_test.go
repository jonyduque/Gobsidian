package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

func TestWikilinkForms(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		target string
		alias  string
		anchor string
		kind   parser.LinkKind
	}{
		{"simples", "[[nota]]", "nota", "", "", parser.LinkWiki},
		{"com alias", "[[nota|apelido]]", "nota", "apelido", "", parser.LinkWiki},
		{"com heading", "[[nota#Cap 1]]", "nota", "", "Cap 1", parser.LinkWiki},
		{"com bloco", "[[nota#^abc123]]", "nota", "", "^abc123", parser.LinkWiki},
		{"heading e alias", "[[nota#Cap 1|Cap um]]", "nota", "Cap um", "Cap 1", parser.LinkWiki},
		{"caminho", "[[Civil/PONTO 03]]", "Civil/PONTO 03", "", "", parser.LinkWiki},
		{"embed", "![[nota]]", "nota", "", "", parser.LinkEmbed},
		{"embed de imagem", "![[diagrama.png]]", "diagrama.png", "", "", parser.LinkEmbed},
		{"embed de secao", "![[nota#Cap 1]]", "nota", "", "Cap 1", parser.LinkEmbed},
		// Espacos interiores: prova que Raw e byte-exato. Sem esta linha, uma
		// implementacao que reconstruisse Raw a partir das partes passaria — e
		// note_move reescreveria o link normalizando o texto do usuario.
		{"espacos interiores", "[[ nota ]]", "nota", "", "", parser.LinkWiki},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parser.Parse([]byte("texto " + tt.in + " texto\n"))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(note.Links) != 1 {
				t.Fatalf("links = %d, quer 1: %+v", len(note.Links), note.Links)
			}

			got := note.Links[0]
			if got.Target != tt.target {
				t.Errorf("Target = %q, quer %q", got.Target, tt.target)
			}
			if got.Alias != tt.alias {
				t.Errorf("Alias = %q, quer %q", got.Alias, tt.alias)
			}
			if got.Anchor != tt.anchor {
				t.Errorf("Anchor = %q, quer %q", got.Anchor, tt.anchor)
			}
			if got.Kind != tt.kind {
				t.Errorf("Kind = %v, quer %v", got.Kind, tt.kind)
			}
			if got.Raw != tt.in {
				t.Errorf("Raw = %q, quer %q — a forma original tem que ser preservada", got.Raw, tt.in)
			}
		})
	}
}

// RF-17: a diferenca entre um grafo de links correto e um grafo plausivel.
func TestWikilinkSuppressedInCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"bloco cercado", "```\n[[nao e link]]\n```\n"},
		{"bloco cercado com linguagem", "```go\n// [[nao e link]]\n```\n"},
		{"codigo inline", "use `[[nao e link]]` aqui\n"},
		{"bloco indentado", "    [[nao e link]]\n"},
		{"escapado", "\\[\\[nao e link\\]\\]\n"},
		{"colchete literal simples", "isto [nao] e link\n"},
		{"til cercado", "~~~\n[[nao e link]]\n~~~\n"},
		// Formas degeneradas: sem elas os guardas correspondentes do parser
		// sobrevivem a mutacao sem nenhum teste reprovar.
		{"alvo vazio", "[[]]\n"},
		{"alvo so espacos", "[[   ]]\n"},
		{"so alias, sem alvo", "[[|apenas]]\n"},
		// "[[" aninhado cujo "]]" mais proximo fecha um alvo vazio: sem o
		// guarda de aninhamento, o "[[" externo captura "x[[" como alvo
		// (nao-vazio, entao o guarda de alvo vazio nao pega) e produz um
		// link espurio. As duas outras formas de "[[" aninhado nao servem
		// pra este teste porque o goldmark reoferece o gatilho um byte
		// adiante e acha um "[[...]]" valido de qualquer forma — o guarda
		// so e observavel quando a estrutura obriga ambas as tentativas a
		// zero links.
		{"colchete duplo aninhado antes do fechamento", "[[x[[]]\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parser.Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(note.Links) != 0 {
				t.Errorf("links = %+v, quer nenhum", note.Links)
			}
		})
	}
}

func TestWikilinkOffsetsPointAtSource(t *testing.T) {
	src := "abc [[nota]] def\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Links) != 1 {
		t.Fatalf("links = %d, quer 1", len(note.Links))
	}

	l := note.Links[0]
	if got := src[l.Start:l.End]; got != "[[nota]]" {
		t.Errorf("src[%d:%d] = %q, quer %q", l.Start, l.End, got, "[[nota]]")
	}
}

// TestWikilinkOffsetsWithFrontmatter confirma que bodyOffset e somado: sem a
// soma, l.Start/l.End continuariam relativos ao corpo, e a fatia apontaria
// para dentro do proprio frontmatter em vez do link.
func TestWikilinkOffsetsWithFrontmatter(t *testing.T) {
	src := "---\ntitle: x\n---\nabc [[nota]] def\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Links) != 1 {
		t.Fatalf("links = %d, quer 1", len(note.Links))
	}

	l := note.Links[0]
	if got := src[l.Start:l.End]; got != "[[nota]]" {
		t.Errorf("src[%d:%d] = %q, quer %q", l.Start, l.End, got, "[[nota]]")
	}
}
