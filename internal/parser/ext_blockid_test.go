package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

func TestBlockIDExtraction(t *testing.T) {
	src := "Primeiro paragrafo. ^abc123\n\nSegundo paragrafo.\n\nTerceiro. ^def456\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Blocks) != 2 {
		t.Fatalf("blocos = %d, quer 2: %+v", len(note.Blocks), note.Blocks)
	}
	if note.Blocks[0].ID != "abc123" {
		t.Errorf("ID = %q, quer %q — o circunflexo nao faz parte do id", note.Blocks[0].ID, "abc123")
	}

	// Start e End delimitam o BLOCO, nao o marcador: e o que note_read com
	// block_id devolve, e o que note_patch com replace_block substitui.
	got := src[note.Blocks[0].Start:note.Blocks[0].End]
	if got != "Primeiro paragrafo. ^abc123" {
		t.Errorf("bloco = %q", got)
	}
}

func TestBlockIDRejectsNonTerminal(t *testing.T) {
	tests := []struct{ name, in string }{
		{"no meio da linha", "texto ^abc123 mais texto\n"},
		{"dentro de codigo", "```\ntexto ^abc123\n```\n"},
		{"circunflexo sozinho", "texto ^\n"},
		{"caracteres invalidos", "texto ^abc def\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parser.Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(note.Blocks) != 0 {
				t.Errorf("blocos = %+v, quer nenhum", note.Blocks)
			}
		})
	}
}

// TestBlockIDListItem confirma que Start aponta para o inicio do texto do
// item de lista (apos o marcador "- "), nao para o inicio do documento nem
// para o "^". Em lista aninhada o mesmo vale para o item filho.
func TestBlockIDListItem(t *testing.T) {
	src := "- primeiro item ^item1\n" +
		"  - filho aninhado ^filho1\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Blocks) != 2 {
		t.Fatalf("blocos = %d, quer 2: %+v", len(note.Blocks), note.Blocks)
	}

	if note.Blocks[0].ID != "item1" {
		t.Errorf("ID[0] = %q, quer %q", note.Blocks[0].ID, "item1")
	}
	got0 := src[note.Blocks[0].Start:note.Blocks[0].End]
	if got0 != "primeiro item ^item1" {
		t.Errorf("bloco[0] = %q, quer %q", got0, "primeiro item ^item1")
	}

	if note.Blocks[1].ID != "filho1" {
		t.Errorf("ID[1] = %q, quer %q", note.Blocks[1].ID, "filho1")
	}
	got1 := src[note.Blocks[1].Start:note.Blocks[1].End]
	if got1 != "filho aninhado ^filho1" {
		t.Errorf("bloco[1] = %q, quer %q", got1, "filho aninhado ^filho1")
	}
}

// TestBlockIDBlockquote confirma que o Start nao inclui o prefixo "> " do
// blockquote — a citacao e sintaxe, nao conteudo do bloco.
func TestBlockIDBlockquote(t *testing.T) {
	src := "> uma citacao ^cite1\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Blocks) != 1 {
		t.Fatalf("blocos = %d, quer 1: %+v", len(note.Blocks), note.Blocks)
	}
	got := src[note.Blocks[0].Start:note.Blocks[0].End]
	if got != "uma citacao ^cite1" {
		t.Errorf("bloco = %q, quer %q", got, "uma citacao ^cite1")
	}
}

// TestBlockIDMultiplePerParagraph confirma que dois marcadores na mesma nota
// produzem dois blocos, e que dois marcadores dentro do MESMO paragrafo
// (linhas soft-wrapped) tambem sao capturados independentemente, cada um com
// Start no inicio do paragrafo (o mesmo paragrafo para os dois).
func TestBlockIDMultiplePerParagraph(t *testing.T) {
	src := "primeira linha ^um\nsegunda linha ^dois\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Blocks) != 2 {
		t.Fatalf("blocos = %d, quer 2: %+v", len(note.Blocks), note.Blocks)
	}
	if note.Blocks[0].ID != "um" || note.Blocks[1].ID != "dois" {
		t.Errorf("IDs = %q, %q", note.Blocks[0].ID, note.Blocks[1].ID)
	}
	// Os dois marcadores pertencem ao mesmo paragrafo: mesmo Start.
	if note.Blocks[0].Start != note.Blocks[1].Start {
		t.Errorf("Start[0] = %d, Start[1] = %d, queria iguais (mesmo paragrafo)",
			note.Blocks[0].Start, note.Blocks[1].Start)
	}
}

// TestBlockIDDuplicateAcrossNote confirma que o mesmo id usado duas vezes na
// nota produz dois blocos distintos — a deduplicacao (se houver) e
// responsabilidade de outra camada, nao do parser.
func TestBlockIDDuplicateAcrossNote(t *testing.T) {
	src := "primeiro ^dup\n\nsegundo ^dup\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Blocks) != 2 {
		t.Fatalf("blocos = %d, quer 2: %+v", len(note.Blocks), note.Blocks)
	}
	if note.Blocks[0].ID != "dup" || note.Blocks[1].ID != "dup" {
		t.Errorf("IDs = %q, %q, queria ambos %q", note.Blocks[0].ID, note.Blocks[1].ID, "dup")
	}
	if note.Blocks[0].Start == note.Blocks[1].Start {
		t.Errorf("Start identico para blocos distintos: %d", note.Blocks[0].Start)
	}
}

// TestBlockIDMultilineParagraph confirma que quando o paragrafo tem varias
// linhas antes do marcador, Start ainda aponta para a PRIMEIRA linha do
// paragrafo, nao para a linha do "^".
func TestBlockIDMultilineParagraph(t *testing.T) {
	src := "linha um\nlinha dois\nlinha tres ^abc\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Blocks) != 1 {
		t.Fatalf("blocos = %d, quer 1: %+v", len(note.Blocks), note.Blocks)
	}
	got := src[note.Blocks[0].Start:note.Blocks[0].End]
	want := "linha um\nlinha dois\nlinha tres ^abc"
	if got != want {
		t.Errorf("bloco = %q, quer %q", got, want)
	}
}

// TestBlockIDTrailingSpaces confirma que espacos em branco depois do id
// ainda contam como fim de linha valido, e que End nao inclui esses espacos.
func TestBlockIDTrailingSpaces(t *testing.T) {
	src := "paragrafo com espacos ^abc123   \n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Blocks) != 1 {
		t.Fatalf("blocos = %d, quer 1: %+v", len(note.Blocks), note.Blocks)
	}
	if note.Blocks[0].ID != "abc123" {
		t.Errorf("ID = %q, quer %q", note.Blocks[0].ID, "abc123")
	}
	got := src[note.Blocks[0].Start:note.Blocks[0].End]
	want := "paragrafo com espacos ^abc123"
	if got != want {
		t.Errorf("bloco = %q, quer %q (sem os espacos finais)", got, want)
	}
}

// TestBlockIDCharset confirma o alfabeto aceito: maiusculas, hifen, e ids
// so-digitos sao todos validos.
func TestBlockIDCharset(t *testing.T) {
	tests := []struct {
		name, in, wantID string
	}{
		{"maiusculas", "texto ^ABC123\n", "ABC123"},
		{"hifen", "texto ^abc-123\n", "abc-123"},
		{"so digitos", "texto ^123456\n", "123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parser.Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(note.Blocks) != 1 {
				t.Fatalf("blocos = %d, quer 1: %+v", len(note.Blocks), note.Blocks)
			}
			if note.Blocks[0].ID != tt.wantID {
				t.Errorf("ID = %q, quer %q", note.Blocks[0].ID, tt.wantID)
			}
		})
	}
}

// TestBlockIDCaretAtLineStart confirma que "^" logo no inicio da linha, sem
// nada antes, ainda e um marcador valido — nao ha regra que exija texto
// precedendo o circunflexo.
func TestBlockIDCaretAtLineStart(t *testing.T) {
	src := "^soleiro\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Blocks) != 1 {
		t.Fatalf("blocos = %d, quer 1: %+v", len(note.Blocks), note.Blocks)
	}
	if note.Blocks[0].ID != "soleiro" {
		t.Errorf("ID = %q, quer %q", note.Blocks[0].ID, "soleiro")
	}
	got := src[note.Blocks[0].Start:note.Blocks[0].End]
	if got != "^soleiro" {
		t.Errorf("bloco = %q, quer %q", got, "^soleiro")
	}
}

// TestBlockIDInlineCodeSpan confirma que um "^abc123" dentro de um span de
// codigo inline (crases simples, nao bloco cercado) nao produz bloco — o
// goldmark ja suprime parsers inline dentro de CodeSpan.
func TestBlockIDInlineCodeSpan(t *testing.T) {
	src := "texto `codigo ^abc123`\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Blocks) != 0 {
		t.Errorf("blocos = %+v, quer nenhum", note.Blocks)
	}
}
