package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

func TestInlineFields(t *testing.T) {
	src := "autor:: Fulano\nano:: 2026\n\n[status:: revisado]\n\nnao e campo: valor\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got := note.Inline["autor"]; len(got) != 1 || got[0] != "Fulano" {
		t.Errorf("autor = %v, quer [Fulano]", got)
	}
	if got := note.Inline["ano"]; len(got) != 1 || got[0] != "2026" {
		t.Errorf("ano = %v, quer [2026]", got)
	}
	if got := note.Inline["status"]; len(got) != 1 || got[0] != "revisado" {
		t.Errorf("status = %v, quer [revisado] — a forma entre colchetes conta", got)
	}
	if _, ok := note.Inline["nao e campo"]; ok {
		t.Error("dois-pontos simples nao e campo inline")
	}
}

func TestInlineFieldRepeatedKey(t *testing.T) {
	src := "tema:: prescricao\ntema:: decadencia\n"

	note, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["tema"]; len(got) != 2 {
		t.Fatalf("tema = %v, quer duas ocorrencias", got)
	}
}

func TestInlineFieldNotInCode(t *testing.T) {
	note, err := parser.Parse([]byte("```\nautor:: Fulano\n```\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Inline) != 0 {
		t.Errorf("inline = %v, quer vazio", note.Inline)
	}
}

// TestInlineFieldSingleColonRejected confirma o caso mais comum de falso
// positivo: um unico ':' e pontuacao normal, nao um campo. Cobre hora,
// razao "a:b", URL com esquema e caminho Windows.
func TestInlineFieldSingleColonRejected(t *testing.T) {
	tests := []struct{ name, in string }{
		{"hora", "reuniao as 14:30\n"},
		{"razao curta", "a:b\n"},
		{"url", "veja http://exemplo.com\n"},
		{"caminho windows", "arquivo em C:\\caminho\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := parser.Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(note.Inline) != 0 {
				t.Errorf("inline = %v, quer vazio", note.Inline)
			}
		})
	}
}

// TestInlineFieldTripleColonRejected confirma que tres ':' seguidos nao
// produzem campo — a regra pede EXATAMENTE dois.
func TestInlineFieldTripleColonRejected(t *testing.T) {
	note, err := parser.Parse([]byte("chave:::valor\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Inline) != 0 {
		t.Errorf("inline = %v, quer vazio", note.Inline)
	}
}

// TestInlineFieldNoSpaceAfterColons confirma que "chave::valor", sem espaco
// depois dos dois pontos, ainda produz o campo.
func TestInlineFieldNoSpaceAfterColons(t *testing.T) {
	note, err := parser.Parse([]byte("chave::valor\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["chave"]; len(got) != 1 || got[0] != "valor" {
		t.Errorf("chave = %v, quer [valor]", got)
	}
}

// TestInlineFieldEmptyValue confirma que "chave::" sem nada depois ainda
// registra o campo, com valor vazio.
func TestInlineFieldEmptyValue(t *testing.T) {
	note, err := parser.Parse([]byte("chave::\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got, ok := note.Inline["chave"]
	if !ok || len(got) != 1 || got[0] != "" {
		t.Errorf("chave = %v (ok=%v), quer [\"\"]", got, ok)
	}
}

// TestInlineFieldNoKeyRejected confirma que "::" sem nada a esquerda (nem
// letra, nem espaco, nem inicio de bloco entre colchetes) nao produz campo.
func TestInlineFieldNoKeyRejected(t *testing.T) {
	note, err := parser.Parse([]byte(":: valor\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Inline) != 0 {
		t.Errorf("inline = %v, quer vazio", note.Inline)
	}
}

// TestInlineFieldKeyWithSpace confirma que uma chave com espaco interno
// ("data de leitura") e aceita por inteiro.
func TestInlineFieldKeyWithSpace(t *testing.T) {
	note, err := parser.Parse([]byte("data de leitura:: 2026-01-01\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["data de leitura"]; len(got) != 1 || got[0] != "2026-01-01" {
		t.Errorf("data de leitura = %v, quer [2026-01-01]", got)
	}
}

// TestInlineFieldBracketedTrailingText confirma que "[chave:: valor] texto
// depois" fecha exatamente no ']' e deixa o texto seguinte intacto — so
// podemos confirmar isso pelo campo em si, ja que o parser nao expoe texto
// solto.
func TestInlineFieldBracketedTrailingText(t *testing.T) {
	note, err := parser.Parse([]byte("[chave:: valor] texto depois\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["chave"]; len(got) != 1 || got[0] != "valor" {
		t.Errorf("chave = %v, quer [valor]", got)
	}
}

// TestInlineFieldTwoPerLine documenta o comportamento real de "a:: 1 b:: 2":
// a forma sem colchetes consome o RESTO DA LINHA como valor (regra 3), entao
// so um campo nasce daqui, com "b:: 2" dentro do valor de "a" — o mesmo
// comportamento do Dataview real, que exige colchetes para varios campos por
// linha.
func TestInlineFieldTwoPerLine(t *testing.T) {
	note, err := parser.Parse([]byte("a:: 1 b:: 2\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Inline) != 1 {
		t.Fatalf("inline = %v, quer 1 chave", note.Inline)
	}
	if got := note.Inline["a"]; len(got) != 1 || got[0] != "1 b:: 2" {
		t.Errorf("a = %v, quer [\"1 b:: 2\"]", got)
	}
	if _, ok := note.Inline["b"]; ok {
		t.Error("b nao deveria existir — foi absorvido pelo valor de a")
	}
}

// TestInlineFieldBracketedTwoPerLine confirma que a forma entre colchetes
// SIM suporta varios campos na mesma linha, porque cada um tem seu proprio
// terminador ']'.
func TestInlineFieldBracketedTwoPerLine(t *testing.T) {
	note, err := parser.Parse([]byte("[a:: 1] [b:: 2]\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["a"]; len(got) != 1 || got[0] != "1" {
		t.Errorf("a = %v, quer [1]", got)
	}
	if got := note.Inline["b"]; len(got) != 1 || got[0] != "2" {
		t.Errorf("b = %v, quer [2]", got)
	}
}

// TestInlineFieldInListItem confirma que a chave nao vaza para dentro do
// marcador de lista "- ": sem o limite de linha vindo de parent.Lines(), o
// '-' do marcador (que pertence ao alfabeto de chave) contaminaria "autor"
// virando "- autor".
func TestInlineFieldInListItem(t *testing.T) {
	note, err := parser.Parse([]byte("- autor:: Fulano\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["autor"]; len(got) != 1 || got[0] != "Fulano" {
		t.Errorf("autor = %v, quer [Fulano]", got)
	}
}

// TestInlineFieldInBlockquote confirma o mesmo para blockquote: o prefixo
// "> " nao entra na chave.
func TestInlineFieldInBlockquote(t *testing.T) {
	note, err := parser.Parse([]byte("> autor:: Fulano\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["autor"]; len(got) != 1 || got[0] != "Fulano" {
		t.Errorf("autor = %v, quer [Fulano]", got)
	}
}

// TestInlineFieldValueWithWikilink documenta que um wikilink dentro do valor
// de um campo nao e coletado separadamente como link: a forma sem colchetes
// consome o resto da linha, opaco a outros parsers inline — o mesmo
// comportamento que WikilinkNode ja tem para o texto do alias.
func TestInlineFieldValueWithWikilink(t *testing.T) {
	note, err := parser.Parse([]byte("fonte:: [[STJ]]\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["fonte"]; len(got) != 1 || got[0] != "[[STJ]]" {
		t.Errorf("fonte = %v, quer [\"[[STJ]]\"]", got)
	}
	if len(note.Links) != 0 {
		t.Errorf("links = %+v, quer nenhum — o valor do campo e opaco", note.Links)
	}
}

// TestInlineFieldInlineCodeSpan confirma que "::" dentro de um span de
// codigo inline nao produz campo — o goldmark ja suprime parsers inline ali.
func TestInlineFieldInlineCodeSpan(t *testing.T) {
	note, err := parser.Parse([]byte("texto `autor:: Fulano` fim\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Inline) != 0 {
		t.Errorf("inline = %v, quer vazio", note.Inline)
	}
}
