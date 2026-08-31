package search

import (
	"testing"
	"unicode"
	"unicode/utf8"
)

// TestAnalyzeCaminhoRapidoConcordaComOLento é a guarda do achado P12.
//
// O caminho rápido ASCII existe porque a esmagadora maioria dos bytes de uma
// nota é ASCII e `utf8.DecodeRuneInString` mais `unicode.IsLetter` são chamadas
// com tabela por trás. Mas ele só é uma otimização enquanto CONCORDA com o
// caminho lento: se discordar num único byte, a tokenização passa a depender de
// o byte ser ASCII — duas contas para a mesma pergunta, que é o padrão que esta
// base já pagou caro.
//
// Confere byte a byte, todos os 128.
func TestAnalyzeCaminhoRapidoConcordaComOLento(t *testing.T) {
	for c := range byte(utf8.RuneSelf) {
		rapido := ehAlfanumericoASCII(c)
		r, _ := utf8.DecodeRuneInString(string(c))
		lento := unicode.IsLetter(r) || unicode.IsDigit(r)
		if rapido != lento {
			t.Errorf("byte %d (%q): rapido = %v, lento = %v", c, string(rune(c)), rapido, lento)
		}
	}
}

// TestAnalyzeOffsetsNaoMudamComAcento fixa o que o caminho rápido pode quebrar
// de pior: os OFFSETS.
//
// Eles são o que o recorte de trecho e a reescrita de link usam para fatiar o
// arquivo. Um caminho rápido que avance o cursor errado num byte multibyte
// desloca tudo o que vem depois — e o sintoma seria trecho cortado no meio de
// uma palavra, não um erro.
func TestAnalyzeOffsetsNaoMudamComAcento(t *testing.T) {
	texto := "ação de execução no processo 13.1.10 — fim"
	for _, tok := range Analyze(texto) {
		if tok.Start < 0 || tok.End > int64(len(texto)) || tok.End <= tok.Start {
			t.Fatalf("token %q com offsets %d..%d fora do texto", tok.Raw, tok.Start, tok.End)
		}
		fatia := texto[tok.Start:tok.End]
		if Normalize(fatia) != tok.Raw {
			t.Errorf("token Raw = %q mas o texto em %d..%d e %q (normalizado: %q)",
				tok.Raw, tok.Start, tok.End, fatia, Normalize(fatia))
		}
	}
}
