package writer_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/writer"
)

func parseNoteForTest(raw []byte) *parser.ParsedNote {
	cleaned, hadBOM := vault.StripBOM(raw)
	note := parser.Parse(cleaned)
	if hadBOM {
		note.ShiftOffsets(int64(vault.BOMLen))
	}
	return note
}

func TestPatchSectionUnderBOMAndCRLFWritesTheRightBytes(t *testing.T) {
	root := t.TempDir()
	raw := []byte("\xEF\xBB\xBF---\r\ntitle: T\r\n---\r\n\r\n" +
		"# Topo\r\n\r\n## Alvo\r\n\r\nconteudo velho\r\n\r\n### Filha\r\n\r\npreservar\r\n\r\n## Depois\r\n\r\nintacto\r\n")

	alvo := filepath.Join(root, "nota.md")
	if err := os.WriteFile(alvo, raw, 0644); err != nil {
		t.Fatal(err)
	}

	rawDisk, err := os.ReadFile(alvo)
	if err != nil {
		t.Fatal(err)
	}

	note := parseNoteForTest(rawDisk)
	h, err := writer.FindHeading(note.Headings, "Alvo")
	if err != nil {
		t.Fatalf("FindHeading falhou: %v", err)
	}

	novoContent := "conteudo novo\r\n"
	patched := writer.PatchSectionContent(rawDisk, *h, novoContent)

	if err := writer.WriteAtomic(context.Background(), alvo, patched); err != nil {
		t.Fatalf("WriteAtomic falhou: %v", err)
	}

	depois, err := os.ReadFile(alvo)
	if err != nil {
		t.Fatal(err)
	}
	s := string(depois)

	if !bytes.HasPrefix(depois, []byte("\xEF\xBB\xBF")) {
		t.Error("BOM perdido na escrita")
	}

	if regexp.MustCompile(`[^\r]\n`).MatchString(s) {
		t.Error("LF solto: o EOL original era CRLF e nao foi preservado")
	}

	if !strings.Contains(s, "## Alvo\r\n") {
		t.Error("o heading do alvo foi apagado")
	}

	if strings.Contains(s, "conteudo velho") {
		t.Error("o conteudo antigo sobreviveu")
	}

	if !strings.Contains(s, "conteudo novo") {
		t.Error("o conteudo novo nao entrou")
	}

	if !strings.Contains(s, "## Depois\r\n\r\nintacto") {
		t.Errorf("conteudo fora do alvo (## Depois) foi destruido:\n%s", s)
	}
}

func TestPatchSection_WithoutBOM_LF(t *testing.T) {
	root := t.TempDir()
	raw := []byte("# Topo\n\n## Alvo\n\nconteudo velho\n\n## Depois\n\nintacto\n")

	alvo := filepath.Join(root, "nota_lf.md")
	if err := os.WriteFile(alvo, raw, 0644); err != nil {
		t.Fatal(err)
	}

	rawDisk, _ := os.ReadFile(alvo)
	note := parseNoteForTest(rawDisk)

	h, err := writer.FindHeading(note.Headings, "Alvo")
	if err != nil {
		t.Fatal(err)
	}

	patched := writer.PatchSectionContent(rawDisk, *h, "conteudo novo\n")
	_ = writer.WriteAtomic(context.Background(), alvo, patched)

	depois, _ := os.ReadFile(alvo)
	s := string(depois)

	if strings.Contains(s, "\r\n") {
		t.Error("CRLF inserido indevidamente em arquivo LF")
	}
	if !strings.Contains(s, "conteudo novo\n") {
		t.Error("conteudo novo nao foi inserido com LF")
	}
}

func TestFindHeading_AmbiguousHeading(t *testing.T) {
	raw := []byte("# Topo\n\n## Dup\n\ntexto1\n\n## Dup\n\ntexto2\n")
	note := parseNoteForTest(raw)

	_, err := writer.FindHeading(note.Headings, "Dup")
	if err == nil {
		t.Fatal("esperava erro de heading ambiguo")
	}

	var ambErr *writer.AmbiguousHeadingError
	if !errors.As(err, &ambErr) {
		t.Fatalf("esperava *AmbiguousHeadingError, obteve %T: %v", err, err)
	}
	if ambErr.Occurrences != 2 {
		t.Errorf("Occurrences = %d, quer 2", ambErr.Occurrences)
	}
}

func TestAppendSection_NoteEndAndHeadingEnd(t *testing.T) {
	root := t.TempDir()
	// Sem newline final
	raw := []byte("# Topo\r\n\r\n## Secao1\r\n\r\nlinha1")
	alvo := filepath.Join(root, "nota_append.md")
	_ = os.WriteFile(alvo, raw, 0644)

	rawDisk, _ := os.ReadFile(alvo)
	note := parseNoteForTest(rawDisk)

	// 1. Append no final da nota
	appendedEnd := writer.AppendSectionContent(rawDisk, nil, "linha2")
	_ = writer.WriteAtomic(context.Background(), alvo, appendedEnd)

	depois1, _ := os.ReadFile(alvo)
	s1 := string(depois1)
	if !strings.Contains(s1, "linha1\r\nlinha2") {
		t.Errorf("linha2 foi colada sem newline na linha1:\n%q", s1)
	}

	// 2. Append no final da secao Secao1
	h, _ := writer.FindHeading(note.Headings, "Secao1")
	appendedSec := writer.AppendSectionContent(rawDisk, h, "linha_anexada_na_secao")
	_ = writer.WriteAtomic(context.Background(), alvo, appendedSec)

	depois2, _ := os.ReadFile(alvo)
	s2 := string(depois2)
	if !strings.Contains(s2, "linha_anexada_na_secao") {
		t.Errorf("conteudo anexado na secao nao foi encontrado:\n%q", s2)
	}
}
