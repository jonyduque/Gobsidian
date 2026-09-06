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

// A ancora do link Markdown sobrevive a reescrita.
//
// Desde 2026-09-06 o parser separa "b.md#Sec" em Target e Anchor, e os ramos
// Markdown de BuildLinkText formatavam SO o alvo novo: um note_move apagaria o
// "#Sec" de todo link reescrito, silenciosamente, e o destino da secao viraria
// o topo da nota.
//
// A ultima linha cobre o encoding: o parser devolve a ancora DECODIFICADA
// ("Com Espaco"), e reemiti-la crua produziria "[x](c.md#Com Espaco)" — um
// destino com espaco, que o CommonMark nao aceita sem colchete angular, ou
// seja, um link que deixa de ser link.
func TestRewriteLinks_PreservaAncoraEmLinkMarkdown(t *testing.T) {
	input := "Link: [x](b.md#Sec)\nEmbed: ![x](b.md#Sec)\nEspaco: [x](b.md#Com%20Espaco)"
	src := []byte(input)

	note := parser.Parse(src)

	if len(note.Links) != 3 {
		t.Fatalf("esperado 3 links, obtido %d", len(note.Links))
	}

	replacements := []writer.LinkReplacement{
		{Link: note.Links[0], NewTarget: "c.md"},
		{Link: note.Links[1], NewTarget: "c.md"},
		{Link: note.Links[2], NewTarget: "c.md"},
	}

	got, err := writer.RewriteLinks(src, replacements)
	if err != nil {
		t.Fatalf("RewriteLinks: %v", err)
	}

	want := "Link: [x](c.md#Sec)\nEmbed: ![x](c.md#Sec)\nEspaco: [x](c.md#Com%20Espaco)"
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

// TestRewriteLinks_LinkNoInicioENoFim fixa as duas fronteiras da passada
// unica: o primeiro link comeca no byte 0, entao a primeira fatia escrita e
// src[0:0]; o ultimo termina exatamente no EOF, entao a fatia final e
// src[len(src):]. As duas sao vazias, e uma delas escrita errado por um byte
// nao aparece em nenhum dos outros casos — todos eles tem texto antes do
// primeiro link e depois do ultimo.
func TestRewriteLinks_LinkNoInicioENoFim(t *testing.T) {
	input := "[[a]] meio [[b]]"
	src := []byte(input)

	note := parser.Parse(src)

	if len(note.Links) != 2 {
		t.Fatalf("esperado 2 links, obtido %d", len(note.Links))
	}
	// Guarda das fronteiras que o teste existe para cobrir. Sem ela, um parser
	// que passasse a devolver offsets encolhidos deixaria o teste verde
	// medindo outra coisa.
	if note.Links[0].Start != 0 {
		t.Fatalf("Links[0].Start = %d, quer 0", note.Links[0].Start)
	}
	if note.Links[1].End != int64(len(src)) {
		t.Fatalf("Links[1].End = %d, quer %d (EOF)", note.Links[1].End, len(src))
	}

	replacements := []writer.LinkReplacement{
		{Link: note.Links[0], NewTarget: "pasta/um"},
		{Link: note.Links[1], NewTarget: "pasta/dois"},
	}

	got, err := writer.RewriteLinks(src, replacements)
	if err != nil {
		t.Fatalf("RewriteLinks: %v", err)
	}

	want := "[[pasta/um]] meio [[pasta/dois]]"
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
