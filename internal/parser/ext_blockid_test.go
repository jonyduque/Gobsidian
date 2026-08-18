package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

func TestBlockIDExtraction(t *testing.T) {
	src := "Primeiro paragrafo. ^abc123\n\nSegundo paragrafo.\n\nTerceiro. ^def456\n"

	note := parser.Parse([]byte(src))
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
			note := parser.Parse([]byte(tt.in))
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

	note := parser.Parse([]byte(src))
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

	note := parser.Parse([]byte(src))
	if len(note.Blocks) != 1 {
		t.Fatalf("blocos = %d, quer 1: %+v", len(note.Blocks), note.Blocks)
	}
	got := src[note.Blocks[0].Start:note.Blocks[0].End]
	if got != "uma citacao ^cite1" {
		t.Errorf("bloco = %q, quer %q", got, "uma citacao ^cite1")
	}
}

// TestBlockIDMultiplePerParagraph confirma que um "^id" que NAO esta na
// ultima linha do paragrafo (linha soft-wrapped) e recusado — o Obsidian
// trata um "^id" no meio de um paragrafo como texto literal, nao como block
// id. Sem esta regra, "^um" (linha 1 de 2) produziria uma faixa que se
// sobrepoe com a de "^dois" (linha 2 de 2): mesmo Start, End diferente, e
// replace_block em "um" sobrescreveria "segunda linha", que pertence a
// "dois". So o marcador da ultima linha do bloco conta.
func TestBlockIDMultiplePerParagraph(t *testing.T) {
	src := "primeira linha ^um\nsegunda linha ^dois\n"

	note := parser.Parse([]byte(src))
	if len(note.Blocks) != 1 {
		t.Fatalf("blocos = %d, quer 1 (so o marcador da ultima linha do paragrafo conta): %+v", len(note.Blocks), note.Blocks)
	}
	if note.Blocks[0].ID != "dois" {
		t.Errorf("ID = %q, quer %q — \"^um\" no meio do paragrafo nao e block id", note.Blocks[0].ID, "dois")
	}
	got := src[note.Blocks[0].Start:note.Blocks[0].End]
	want := "primeira linha ^um\nsegunda linha ^dois"
	if got != want {
		t.Errorf("bloco = %q, quer %q", got, want)
	}
}

// TestBlockIDThreeLineParagraphOnlyLastMarkerCounts cobre o caso de tres
// linhas com dois marcadores intermediario+final: so o ultimo produz bloco,
// e a faixa cobre o paragrafo inteiro ate ele.
func TestBlockIDThreeLineParagraphOnlyLastMarkerCounts(t *testing.T) {
	src := "linha um\nlinha dois ^b\nlinha tres ^c\n"

	note := parser.Parse([]byte(src))
	if len(note.Blocks) != 1 {
		t.Fatalf("blocos = %d, quer 1: %+v", len(note.Blocks), note.Blocks)
	}
	if note.Blocks[0].ID != "c" {
		t.Errorf("ID = %q, quer %q — \"^b\" no meio nao e block id", note.Blocks[0].ID, "c")
	}
	got := src[note.Blocks[0].Start:note.Blocks[0].End]
	want := "linha um\nlinha dois ^b\nlinha tres ^c"
	if got != want {
		t.Errorf("bloco = %q, quer %q", got, want)
	}
}

// TestBlockIDMultilineBlockquotePrefixAsymmetry documenta o limite descrito
// em Block.Start: numa citacao de varias linhas, o "> " da PRIMEIRA linha
// fica fora da faixa (Start comeca depois dele), mas o "> " de qualquer
// linha de CONTINUACAO fica dentro — a faixa e um intervalo contiguo no
// buffer bruto, entao nao ha como excluir os dois ao mesmo tempo. Fixa o
// formato para que M4 descubra a assimetria em vez de ser surpreendido.
func TestBlockIDMultilineBlockquotePrefixAsymmetry(t *testing.T) {
	src := "> linha um\n> linha dois ^abc\n"

	note := parser.Parse([]byte(src))
	if len(note.Blocks) != 1 {
		t.Fatalf("blocos = %d, quer 1: %+v", len(note.Blocks), note.Blocks)
	}
	got := src[note.Blocks[0].Start:note.Blocks[0].End]
	want := "linha um\n> linha dois ^abc"
	if got != want {
		t.Errorf("bloco = %q, quer %q (\"> \" da 1a linha fora, da 2a dentro)", got, want)
	}
}

// TestBlockIDMultilineListItemPrefixAsymmetry e o mesmo limite que
// TestBlockIDMultilineBlockquotePrefixAsymmetry, para item de lista: a
// indentacao da linha de continuacao fica dentro da faixa.
func TestBlockIDMultilineListItemPrefixAsymmetry(t *testing.T) {
	src := "- linha um\n  linha dois ^abc\n"

	note := parser.Parse([]byte(src))
	if len(note.Blocks) != 1 {
		t.Fatalf("blocos = %d, quer 1: %+v", len(note.Blocks), note.Blocks)
	}
	got := src[note.Blocks[0].Start:note.Blocks[0].End]
	want := "linha um\n  linha dois ^abc"
	if got != want {
		t.Errorf("bloco = %q, quer %q (\"- \" da 1a linha fora, indentacao da 2a dentro)", got, want)
	}
}

// TestBlockIDDuplicateAcrossNote confirma que o mesmo id usado duas vezes na
// nota produz dois blocos distintos — a deduplicacao (se houver) e
// responsabilidade de outra camada, nao do parser.
func TestBlockIDDuplicateAcrossNote(t *testing.T) {
	src := "primeiro ^dup\n\nsegundo ^dup\n"

	note := parser.Parse([]byte(src))
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

	note := parser.Parse([]byte(src))
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

	note := parser.Parse([]byte(src))
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
			note := parser.Parse([]byte(tt.in))
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

	note := parser.Parse([]byte(src))
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

	note := parser.Parse([]byte(src))
	if len(note.Blocks) != 0 {
		t.Errorf("blocos = %+v, quer nenhum", note.Blocks)
	}
}
