package writer_test

import (
	"errors"
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/writer"
)

func TestRewriteLinks_PreservesAliasAndAnchor(t *testing.T) {
	input := "Veja [[Civil/PONTO 03|Ponto 3 — Obrigações]] e [[a#Seção]] e [[a#^bloco]]."
	src := []byte(input)

	note := parser.Parse(src)

	if len(note.Links) != 3 {
		t.Fatalf("esperado 3 links, obtido %d", len(note.Links))
	}

	replacements := []writer.LinkReplacement{
		{Link: note.Links[0], NewTarget: "Direito Civil/PONTO 03"},
		{Link: note.Links[1], NewTarget: "b"},
		{Link: note.Links[2], NewTarget: "b"},
	}

	got, err := writer.RewriteLinks(src, replacements)
	if err != nil {
		t.Fatalf("RewriteLinks: %v", err)
	}

	want := "Veja [[Direito Civil/PONTO 03|Ponto 3 — Obrigações]] e [[b#Seção]] e [[b#^bloco]]."
	if string(got) != want {
		t.Errorf("obtido %q, quer %q", string(got), want)
	}
}

func TestRewriteLinks_PreservesSyntaxAndEmbed(t *testing.T) {
	input := "Markdown: [texto](antigo.md)\nEmbed: ![[imagem_antiga.png]]"
	src := []byte(input)

	note := parser.Parse(src)

	if len(note.Links) != 2 {
		t.Fatalf("esperado 2 links, obtido %d", len(note.Links))
	}

	replacements := []writer.LinkReplacement{
		{Link: note.Links[0], NewTarget: "novo.md"},
		{Link: note.Links[1], NewTarget: "imagem_nova.png"},
	}

	got, err := writer.RewriteLinks(src, replacements)
	if err != nil {
		t.Fatalf("RewriteLinks: %v", err)
	}

	want := "Markdown: [texto](novo.md)\nEmbed: ![[imagem_nova.png]]"
	if string(got) != want {
		t.Errorf("obtido %q, quer %q", string(got), want)
	}
}

func TestRewriteLinks_MultipleOccurrencesInSameNote(t *testing.T) {
	input := "Primeiro [[nota_antiga]], segundo [[nota_antiga]] e terceiro [texto](nota_antiga.md)."
	src := []byte(input)

	note := parser.Parse(src)

	if len(note.Links) != 3 {
		t.Fatalf("esperado 3 links, obtido %d", len(note.Links))
	}

	replacements := []writer.LinkReplacement{
		{Link: note.Links[0], NewTarget: "pasta/nota_nova"},
		{Link: note.Links[1], NewTarget: "pasta/nota_nova"},
		{Link: note.Links[2], NewTarget: "pasta/nota_nova.md"},
	}

	got, err := writer.RewriteLinks(src, replacements)
	if err != nil {
		t.Fatalf("RewriteLinks: %v", err)
	}

	want := "Primeiro [[pasta/nota_nova]], segundo [[pasta/nota_nova]] e terceiro [texto](pasta/nota_nova.md)."
	if string(got) != want {
		t.Errorf("obtido %q, quer %q", string(got), want)
	}
}

func TestRewriteLinks_RejectsInvalidOffsets(t *testing.T) {
	src := []byte("Texto com [[link]].")

	badLink := parser.Link{
		Raw:    "[[link]]",
		Target: "link",
		Kind:   parser.LinkWiki,
		Start:  -1,
		End:    -1,
	}

	_, err := writer.RewriteLinks(src, []writer.LinkReplacement{{Link: badLink, NewTarget: "novo"}})
	if err == nil {
		t.Fatal("esperado erro ao passar Start=-1, obtido nil")
	}

	if !errors.Is(err, writer.ErrInvalidLinkOffset) {
		t.Errorf("esperado ErrInvalidLinkOffset, obtido: %v", err)
	}
}

func TestRewriteLinks_PreservesBOMAndEOL(t *testing.T) {
	input := "\xef\xbb\xbfLinha 1\r\nVeja [[antigo]]\r\nLinha 3\r\n"
	src := []byte(input)

	note := parser.Parse(src)

	if len(note.Links) != 1 {
		t.Fatalf("esperado 1 link, obtido %d", len(note.Links))
	}

	replacements := []writer.LinkReplacement{
		{Link: note.Links[0], NewTarget: "novo_caminho"},
	}

	got, err := writer.RewriteLinks(src, replacements)
	if err != nil {
		t.Fatalf("RewriteLinks: %v", err)
	}

	want := "\xef\xbb\xbfLinha 1\r\nVeja [[novo_caminho]]\r\nLinha 3\r\n"
	if string(got) != want {
		t.Errorf("obtido %q, quer %q", string(got), want)
	}
}
