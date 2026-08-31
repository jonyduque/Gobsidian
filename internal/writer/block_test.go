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

	"github.com/jonyd/gobsidian/internal/writer"
)

func TestReplaceBlock_ParagraphListAndQuote(t *testing.T) {
	root := t.TempDir()
	raw := []byte("# Teste de Blocos\r\n\r\n" +
		"Este e um paragrafo original ^p1\r\n\r\n" +
		"- Item de lista original ^list1\r\n\r\n" +
		"> Citacao original ^quote1\r\n")

	alvo := filepath.Join(root, "blocos.md")
	if err := os.WriteFile(alvo, raw, 0644); err != nil {
		t.Fatal(err)
	}

	rawDisk, _ := os.ReadFile(alvo)
	note := parseNoteForTest(rawDisk)

	// 1. Paragrafo
	bP1, err := writer.FindBlock(note.Blocks, "p1")
	if err != nil {
		t.Fatalf("FindBlock p1: %v", err)
	}
	patchedP1 := writer.ReplaceBlockContent(rawDisk, *bP1, "Paragrafo substituido")
	_ = writer.WriteAtomic(context.Background(), alvo, patchedP1)

	sP1 := string(patchedP1)
	if !strings.Contains(sP1, "Paragrafo substituido ^p1") {
		t.Errorf("paragrafo nao substituido corretamente:\n%s", sP1)
	}

	// 2. Lista
	bList, err := writer.FindBlock(note.Blocks, "list1")
	if err != nil {
		t.Fatalf("FindBlock list1: %v", err)
	}
	patchedList := writer.ReplaceBlockContent(rawDisk, *bList, "Item de lista substituido")
	sList := string(patchedList)
	if !strings.Contains(sList, "- Item de lista substituido ^list1") {
		t.Errorf("item de lista nao conservou o marcador '- ':\n%s", sList)
	}

	// 3. Citacao
	bQuote, err := writer.FindBlock(note.Blocks, "quote1")
	if err != nil {
		t.Fatalf("FindBlock quote1: %v", err)
	}
	patchedQuote := writer.ReplaceBlockContent(rawDisk, *bQuote, "Citacao substituida")
	sQuote := string(patchedQuote)
	if !strings.Contains(sQuote, "> Citacao substituida ^quote1") {
		t.Errorf("citacao nao conservou o marcador '> ':\n%s", sQuote)
	}
}

func TestReplaceBlock_AmbiguousBlockID(t *testing.T) {
	raw := []byte("Paragrafo um ^dup\r\n\r\nParagrafo dois ^dup\r\n")
	note := parseNoteForTest(raw)

	_, err := writer.FindBlock(note.Blocks, "dup")
	if err == nil {
		t.Fatal("esperava erro de bloco ambiguo")
	}

	var ambErr *writer.AmbiguousBlockError
	if !errors.As(err, &ambErr) {
		t.Fatalf("esperava *AmbiguousBlockError, obteve %T: %v", err, err)
	}
	if ambErr.Occurrences != 2 {
		t.Errorf("Occurrences = %d, quer 2", ambErr.Occurrences)
	}
}

func TestReplaceBlock_BlockNotFoundNamesID(t *testing.T) {
	raw := []byte("Paragrafo normal\r\n")
	note := parseNoteForTest(raw)

	_, err := writer.FindBlock(note.Blocks, "inexistente")
	if err == nil {
		t.Fatal("esperava erro para bloco inexistente")
	}

	var nfErr *writer.BlockNotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatalf("esperava *BlockNotFoundError, obteve %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "^inexistente") {
		t.Errorf("mensagem de erro deve nomear o id (^inexistente): %v", err)
	}
}

func TestReplaceBlock_UnderBOMAndCRLF(t *testing.T) {
	root := t.TempDir()
	raw := []byte("\xEF\xBB\xBF---\r\ntitle: T\r\n---\r\n\r\n" +
		"# Topo\r\n\r\nParagrafo velho ^abc\r\n")

	alvo := filepath.Join(root, "nota_bom_block.md")
	_ = os.WriteFile(alvo, raw, 0644)

	rawDisk, _ := os.ReadFile(alvo)
	note := parseNoteForTest(rawDisk)

	b, err := writer.FindBlock(note.Blocks, "abc")
	if err != nil {
		t.Fatal(err)
	}

	patched := writer.ReplaceBlockContent(rawDisk, *b, "Paragrafo novo")
	_ = writer.WriteAtomic(context.Background(), alvo, patched)

	depois, _ := os.ReadFile(alvo)
	s := string(depois)

	if !bytes.HasPrefix(depois, []byte("\xEF\xBB\xBF")) {
		t.Error("BOM foi perdido na substituicao do bloco")
	}
	if regexp.MustCompile(`[^\r]\n`).MatchString(s) {
		t.Error("LF solto: EOL era CRLF e nao foi preservado")
	}
	if !strings.Contains(s, "Paragrafo novo ^abc\r\n") {
		t.Errorf("bloco nao foi substituido corretamente:\n%s", s)
	}
}
