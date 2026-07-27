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

// TestExtractHeadingsFenceClosingLengthMismatch pins a known gap rather than
// blessing it as correct: the fence tracker toggles on any line with a
// three-or-more backtick run, without checking that a closing fence is at
// least as long as the opener. CommonMark requires the closer to be >= the
// opener's length. A fence opened with four backticks should only close on
// four-or-more backticks; a bare "```" inside it is ordinary fenced content.
// Here it closes early on the inner "```" line, so everything after —
// including "## After" — is misread and the heading never appears. Recorded
// so an accidental change to this behavior gets caught, and flagged in the
// task report as a real gap for follow-up, not fixed in this task.
func TestExtractHeadingsFenceClosingLengthMismatch(t *testing.T) {
	body := "# Before\n````go\ncode\n```\nstill in fence?\n````\n## After\n"

	hs := parser.ExtractHeadings([]byte(body), 0)

	// Known-wrong under CommonMark: "## After" should be a heading here but
	// is swallowed because the tracker closes the fence on the inner ``` line.
	if len(hs) != 1 {
		t.Fatalf("headings = %d, quer 1 (comportamento atual, com lacuna conhecida): %+v", len(hs), hs)
	}
	if hs[0].Text != "Before" {
		t.Errorf("hs[0].Text = %q", hs[0].Text)
	}
}
