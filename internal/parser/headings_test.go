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
	body := "# A\r\ntexto\r\n## B\r\n"

	hs := parser.ExtractHeadings([]byte(body), 0)

	if len(hs) != 2 {
		t.Fatalf("headings = %d, quer 2", len(hs))
	}
	if hs[0].Text != "A" || hs[1].Text != "B" {
		t.Errorf("textos = %q, %q — CR nao foi removido", hs[0].Text, hs[1].Text)
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
