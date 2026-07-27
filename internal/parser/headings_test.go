package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

func TestExtractHeadingsSectionBoundaries(t *testing.T) {
	body := "# Titulo\ntexto a\n\n## Cap 1\ntexto b\n\n### Sub\ntexto c\n\n## Cap 2\ntexto d\n"

	hs := parser.ExtractHeadings([]byte(body), 0)

	if len(hs) != 4 {
		t.Fatalf("headings = %d, quer 4: %+v", len(hs), hs)
	}

	// "## Cap 1" termina onde "## Cap 2" comeca — inclui "### Sub", que e
	// subsecao dele, e NAO vai ate o fim do arquivo.
	cap1 := hs[1]
	if cap1.Text != "Cap 1" || cap1.Level != 2 {
		t.Fatalf("hs[1] = %+v", cap1)
	}
	cap2Start := hs[3].Start
	if cap1.End != cap2Start {
		t.Errorf("Cap 1 End = %d, quer %d (inicio de Cap 2)", cap1.End, cap2Start)
	}

	// "### Sub" termina no proximo heading de nivel <= 3, que e "## Cap 2".
	if hs[2].End != cap2Start {
		t.Errorf("Sub End = %d, quer %d", hs[2].End, cap2Start)
	}

	// O ultimo heading vai ate o fim.
	if hs[3].End != int64(len(body)) {
		t.Errorf("Cap 2 End = %d, quer %d", hs[3].End, len(body))
	}

	// BodyStart pula a linha do titulo.
	if got := body[cap1.BodyStart:cap1.End]; got != "texto b\n\n### Sub\ntexto c\n\n" {
		t.Errorf("corpo de Cap 1 = %q", got)
	}
}

func TestExtractHeadingsRespectsBodyOffset(t *testing.T) {
	body := "# Titulo\n"
	hs := parser.ExtractHeadings([]byte(body), 100)

	if len(hs) != 1 {
		t.Fatalf("headings = %d, quer 1", len(hs))
	}
	if hs[0].Start != 100 {
		t.Errorf("Start = %d, quer 100 — offsets sao relativos ao arquivo", hs[0].Start)
	}
}

func TestExtractHeadingsIgnoresCodeBlocks(t *testing.T) {
	body := "# Real\n\n```\n# Nao e heading\n```\n\n## Tambem real\n"

	hs := parser.ExtractHeadings([]byte(body), 0)

	if len(hs) != 2 {
		t.Fatalf("headings = %d, quer 2: %+v", len(hs), hs)
	}
	if hs[1].Text != "Tambem real" {
		t.Errorf("hs[1].Text = %q", hs[1].Text)
	}
}

func TestExtractHeadingsCRLF(t *testing.T) {
	// B is level 1, same as A: closeSections only ends A's section at a
	// heading of level <= A's own (verified in
	// TestExtractHeadingsSectionBoundaries). A level-2 "## B" would be a
	// subsection of A, so A.End would legitimately run to the buffer's end
	// instead of B's Start — same-level siblings are what makes A.End land
	// exactly at B.Start observable here, mirroring the choice made in
	// TestExtractHeadingsFenceRequiresMatchingCloseLength.
	body := "# A\r\ntexto\r\n# B\r\n"

	hs := parser.ExtractHeadings([]byte(body), 0)

	if len(hs) != 2 {
		t.Fatalf("headings = %d, quer 2", len(hs))
	}
	if hs[0].Text != "A" || hs[1].Text != "B" {
		t.Errorf("textos = %q, %q", hs[0].Text, hs[1].Text)
	}

	// Os offsets sao o que importa aqui, e o que uma assercao so de texto nao
	// pega: TrimSpace dentro de parseATXHeading ja come o CR, entao remover o
	// TrimRight nao muda o texto e muda todas as posicoes. Foi exatamente esta
	// classe de omissao que a revisao da Task 12 encontrou.
	if hs[0].BodyStart != 5 {
		t.Errorf("A.BodyStart = %d, quer 5", hs[0].BodyStart)
	}
	if hs[1].Start != 12 {
		t.Errorf("B.Start = %d, quer 12", hs[1].Start)
	}
	if hs[0].End != hs[1].Start {
		t.Errorf("A.End = %d, quer %d (inicio de B)", hs[0].End, hs[1].Start)
	}
	if got := body[hs[0].BodyStart:hs[0].End]; got != "texto\r\n" {
		t.Errorf("corpo de A = %q", got)
	}
}

// As regras abaixo sobreviviam a mutacao sem nenhum teste reprovar: o
// comprimento minimo da cerca de fechamento, a info string, os limites de
// indentacao e cada regra de parseATXHeading. Uma regra que nenhuma mutacao
// reprova nao esta verificada, esta apenas escrita.
func TestExtractHeadingsRules(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"fechamento mais longo que a abertura fecha", "````\ncodigo\n`````\n# Depois\n", []string{"Depois"}},
		{"fechamento mais curto nao fecha", "````\n```\n# Dentro\n````\n# Depois\n", []string{"Depois"}},
		{"til nao fecha crase", "```\n~~~\n# Dentro\n```\n# Depois\n", []string{"Depois"}},
		{"crase nao fecha til", "~~~\n```\n# Dentro\n~~~\n# Depois\n", []string{"Depois"}},
		{"info string no fechamento nao fecha", "```\n``` go\n# Dentro\n```\n# Depois\n", []string{"Depois"}},
		{"abertura com linguagem fecha com cerca simples", "```go\ncodigo\n```\n# Depois\n", []string{"Depois"}},
		{"crase na info string nao abre cerca", "```x``` y\n# Depois\n", []string{"Depois"}},
		{"cerca indentada com quatro espacos nao abre", "    ```\n# Depois\n", []string{"Depois"}},
		{"cerca indentada com tres espacos abre", "   ```\n# Dentro\n   ```\n# Depois\n", []string{"Depois"}},
		{"heading indentado com tres espacos vale", "   # A\n", []string{"A"}},
		{"heading indentado com quatro espacos nao vale", "    # A\n", nil},
		{"sete cerquilhas nao e heading", "####### A\n", nil},
		{"sem espaco depois da cerquilha nao e heading", "#A\n", nil},
		{"cerquilhas sozinhas viram heading de texto vazio", "##\n", []string{""}},
		{"fechamento com cerquilhas e removido", "## Titulo ##\n", []string{"Titulo"}},
		{"cerquilha colada ao texto NAO e fechamento", "# Notas sobre C#\n", []string{"Notas sobre C#"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hs := parser.ExtractHeadings([]byte(tt.body), 0)

			if len(hs) != len(tt.want) {
				t.Fatalf("headings = %d (%+v), quer %d %v", len(hs), hs, len(tt.want), tt.want)
			}
			for i, w := range tt.want {
				if hs[i].Text != w {
					t.Errorf("headings[%d].Text = %q, quer %q", i, hs[i].Text, w)
				}
			}
		})
	}
}

// TestExtractHeadingsFenceRequiresMatchingCloseLength asserts the
// CommonMark-correct guarantee: a closing fence must have at least as many
// characters as the opener. A fence opened with four backticks is NOT closed
// by a bare three-backtick line inside it — that line is ordinary fenced
// content, since a note documenting Markdown routinely nests a shorter fence
// example inside a longer one. "# After" therefore stays inside the block
// and must not be extracted as a heading.
//
// closesFence compares n >= open.count rather than n == open.count on
// purpose: CommonMark only requires the closer to be at least as long as the
// opener, not exactly as long. A five-backtick line must still close a
// four-backtick fence.
//
// "After" is level 1, same as "Before": that makes closeSections end
// "Before"'s section exactly at "After"'s Start, so a truncated End (the
// bug this guards) is directly observable instead of hidden behind a
// level-nesting rule.
func TestExtractHeadingsFenceRequiresMatchingCloseLength(t *testing.T) {
	body := "# Before\n````go\ncode\n```\nstill in fence?\n````\n# After\n"

	hs := parser.ExtractHeadings([]byte(body), 0)

	// The inner ``` does not close the ```` fence, so "# After" is still
	// inside the block and must not be read as structure.
	if len(hs) != 2 {
		t.Fatalf("headings = %d, quer 2: %+v", len(hs), hs)
	}
	if hs[0].Text != "Before" {
		t.Errorf("hs[0].Text = %q", hs[0].Text)
	}
	if hs[1].Text != "After" {
		t.Errorf("hs[1].Text = %q", hs[1].Text)
	}

	// The section-offset consequence matters as much as the count: "Before"
	// must span the whole fenced block, including the inner ``` line, up to
	// where "## After" actually starts. A correct heading count with a
	// truncated End is still a broken note_read.
	afterStart := hs[1].Start
	if hs[0].End != afterStart {
		t.Errorf("Before.End = %d, quer %d (inicio de After — cerca inteira incluida)", hs[0].End, afterStart)
	}
}

// TestExtractHeadingsUnterminatedFenceSwallowsRestOfBuffer covers a fence
// that opens and never closes before the buffer ends: everything after the
// opener, including any headings, stays inside the (still open) block and
// must not be extracted.
func TestExtractHeadingsUnterminatedFenceSwallowsRestOfBuffer(t *testing.T) {
	body := "# Before\n```\ncode\n## Not a heading\nmore code\n"

	hs := parser.ExtractHeadings([]byte(body), 0)

	if len(hs) != 1 {
		t.Fatalf("headings = %d, quer 1: %+v", len(hs), hs)
	}
	if hs[0].Text != "Before" {
		t.Errorf("hs[0].Text = %q", hs[0].Text)
	}
	if hs[0].End != int64(len(body)) {
		t.Errorf("Before.End = %d, quer %d (fim do buffer, cerca nunca fechou)", hs[0].End, len(body))
	}
}

// TestExtractHeadingsFenceClosesOnBarePlainFence covers a fence opened with
// a language tag: the info string is only checked at open time, so a plain
// closing line with no tag still closes it.
func TestExtractHeadingsFenceClosesOnBarePlainFence(t *testing.T) {
	body := "# Before\n```go\ncode\n```\n## After\n"

	hs := parser.ExtractHeadings([]byte(body), 0)

	if len(hs) != 2 {
		t.Fatalf("headings = %d, quer 2: %+v", len(hs), hs)
	}
	if hs[1].Text != "After" {
		t.Errorf("hs[1].Text = %q", hs[1].Text)
	}
}

// TestExtractHeadingsFenceCharactersDoNotCrossClose covers the two fence
// characters not closing each other: a "~~~" line inside a "```" block is
// ordinary content, and vice versa.
func TestExtractHeadingsFenceCharactersDoNotCrossClose(t *testing.T) {
	t.Run("tilde dentro de crase", func(t *testing.T) {
		body := "# Before\n```\n~~~\nstill fenced\n```\n## After\n"

		hs := parser.ExtractHeadings([]byte(body), 0)

		if len(hs) != 2 {
			t.Fatalf("headings = %d, quer 2: %+v", len(hs), hs)
		}
		if hs[1].Text != "After" {
			t.Errorf("hs[1].Text = %q", hs[1].Text)
		}
	})

	t.Run("crase dentro de til", func(t *testing.T) {
		body := "# Before\n~~~\n```\nstill fenced\n~~~\n## After\n"

		hs := parser.ExtractHeadings([]byte(body), 0)

		if len(hs) != 2 {
			t.Fatalf("headings = %d, quer 2: %+v", len(hs), hs)
		}
		if hs[1].Text != "After" {
			t.Errorf("hs[1].Text = %q", hs[1].Text)
		}
	})
}
