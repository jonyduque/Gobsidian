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

// TestInlineFieldTwoPerLine documenta o comportamento real de "a:: 1 b:: 2".
// A regra 3 do Dataview continua valendo para o VALOR: a forma sem
// colchetes nao tem terminador proprio, entao o valor registrado para "a" e
// a linha inteira depois de "a::", incluindo o "b:: 2" — exatamente como o
// Dataview real, que exige colchetes para mais de um campo por linha.
//
// Como o parser nao consome mais esse valor (regra 3b — precisa permanecer
// visivel para outros parsers inline, ver TestInlineFieldValueWithWikilink),
// o segundo "::" dentro dele tambem dispara o parser inline de novo. Sem a
// regra 2 (chave so no INICIO DA LINHA), o passo pra tras a partir desse
// segundo "::" pararia no primeiro ':' do par "a::" e inventaria a chave
// espuria "1 b" com bytes que ja pertenciam ao valor de "a". Com a regra 2,
// esse passo pra tras encontra o ':' antes de alcancar o inicio da linha e
// rejeita — "1 b" nao comeca a linha, entao nao vira campo. Ver
// TestInlineFieldBracketedTwoPerLine para a forma que SUPORTA mais de um
// campo por linha de proposito.
func TestInlineFieldTwoPerLine(t *testing.T) {
	note, err := parser.Parse([]byte("a:: 1 b:: 2\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["a"]; len(got) != 1 || got[0] != "1 b:: 2" {
		t.Errorf("a = %v, quer [\"1 b:: 2\"]", got)
	}
	if _, ok := note.Inline["1 b"]; ok {
		t.Errorf("inline = %v, \"1 b\" nao deveria existir (regra 2: campo simples e de linha)", note.Inline)
	}
	if len(note.Inline) != 1 {
		t.Fatalf("inline = %v, quer so a chave \"a\"", note.Inline)
	}
}

// TestInlineFieldAfterTagNotField confirma o segundo artefato que a regra 2
// fecha: "#tag::valor" nao pode produzir um campo cuja chave e montada com
// bytes que o parser de tag (prioridade 120, dispara antes) ja reivindicou.
// O parser de tag avanca sobre "#tag", mas o buffer bruto por tras continua
// com o '#' ali; o passo pra tras a partir do "::" encontraria "tag" antes
// de alcancar o inicio da linha, e regra 2 rejeita porque "tag" nao comeca a
// linha — o '#' esta antes dele. A tag em si continua sendo extraida
// normalmente pelo TagExtension; este teste olha so para note.Inline.
func TestInlineFieldAfterTagNotField(t *testing.T) {
	note, err := parser.Parse([]byte("#tag::valor\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(note.Inline) != 0 {
		t.Errorf("inline = %v, quer vazio (\"tag\" nao pode virar chave de campo)", note.Inline)
	}
	if len(note.Tags) != 1 || note.Tags[0] != "tag" {
		t.Errorf("tags = %v, quer [tag] (a tag em si continua valendo)", note.Tags)
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

// TestInlineFieldValueWithWikilink confirma que um wikilink dentro do valor
// de um campo AINDA e coletado como link: o parser nao consome o valor, so
// avanca sobre "chave::" e deixa o goldmark oferecer o resto da linha a
// outros parsers inline. O valor continua registrado em Inline por inteiro.
// Sem isto, "fonte:: [[STJ]]" apagaria um link que o commit anterior ao
// Task 17 ja coletava — Dataview field guardando link e idioma comum de
// cofre.
func TestInlineFieldValueWithWikilink(t *testing.T) {
	note, err := parser.Parse([]byte("fonte:: [[STJ]]\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["fonte"]; len(got) != 1 || got[0] != "[[STJ]]" {
		t.Errorf("fonte = %v, quer [\"[[STJ]]\"]", got)
	}
	if len(note.Links) != 1 || note.Links[0].Target != "STJ" {
		t.Errorf("links = %+v, quer um link para STJ", note.Links)
	}
}

// TestInlineFieldValueWithEmbed e o mesmo que
// TestInlineFieldValueWithWikilink, para a forma de embed ("![[...]]").
func TestInlineFieldValueWithEmbed(t *testing.T) {
	note, err := parser.Parse([]byte("capa:: ![[img.png]]\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["capa"]; len(got) != 1 || got[0] != "![[img.png]]" {
		t.Errorf("capa = %v, quer [\"![[img.png]]\"]", got)
	}
	if len(note.Links) != 1 || note.Links[0].Target != "img.png" || note.Links[0].Kind != parser.LinkEmbed {
		t.Errorf("links = %+v, quer um embed para img.png", note.Links)
	}
}

// TestInlineFieldValueWithMarkdownLink e o mesmo, para a grafia Markdown de
// link ("[texto](destino)").
func TestInlineFieldValueWithMarkdownLink(t *testing.T) {
	note, err := parser.Parse([]byte("fonte:: [STJ](stj.md)\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["fonte"]; len(got) != 1 || got[0] != "[STJ](stj.md)" {
		t.Errorf("fonte = %v, quer [\"[STJ](stj.md)\"]", got)
	}
	found := false
	for _, l := range note.Links {
		if l.Target == "stj.md" && l.Kind == parser.LinkMarkdown {
			found = true
		}
	}
	if !found {
		t.Errorf("links = %+v, quer um link markdown para stj.md", note.Links)
	}
}

// TestInlineFieldBracketedNestedWikilinkValue confirma que o valor de um
// campo entre colchetes cujo valor e ele mesmo um wikilink nao trunca no
// primeiro ']' (que pertence ao wikilink): o rastreamento de profundidade
// encontra o ']' que fecha o CAMPO, nao o do wikilink aninhado.
func TestInlineFieldBracketedNestedWikilinkValue(t *testing.T) {
	note, err := parser.Parse([]byte("[fonte:: [[STJ]]]\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := note.Inline["fonte"]; len(got) != 1 || got[0] != "[[STJ]]" {
		t.Errorf("fonte = %v, quer [\"[[STJ]]\"]", got)
	}
}

// TestInlineFieldBracketedDeclinesBeforeMarkdownLink confirma o Critical 2:
// quando o ']' da forma entre colchetes e seguido imediatamente de '(', o
// texto inteiro e o de um link Markdown, nao um campo — mesma prioridade
// (130) vem antes do parser de link do CommonMark (~200), entao sem esta
// recusa "destino.md" nunca seria interpretado como link.
func TestInlineFieldBracketedDeclinesBeforeMarkdownLink(t *testing.T) {
	note, err := parser.Parse([]byte("[Nota:: veja](destino.md)\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	found := false
	for _, l := range note.Links {
		if l.Target == "destino.md" && l.Kind == parser.LinkMarkdown {
			found = true
		}
	}
	if !found {
		t.Errorf("links = %+v, quer um link markdown para destino.md", note.Links)
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
