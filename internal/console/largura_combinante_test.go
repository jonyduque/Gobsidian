package console

import (
	"bytes"
	"strings"
	"testing"
	"unicode"
)

// TestLarguraVisivelNaoContaMarcaCombinante e H4.
func TestLarguraVisivelNaoContaMarcaCombinante(t *testing.T) {
	nfd := "A" + "c" + string(rune(0x0327)) + "a" + string(rune(0x0303)) + "o"
	if got := larguraVisivel(nfd); got != 4 {
		t.Fatalf("larguraVisivel(Acao em NFD) = %d, esperado 4", got)
	}
}

// colunasNaTela conta as colunas de uma linha SEM ANSI por uma conta propria do
// teste. Nao pode ser larguraVisivel: a primeira redacao do teste abaixo media
// as linhas com a funcao que monta a moldura, e o mesmo erro dos dois lados
// deixava as linhas "iguais" -- a prova de mutacao de 2026-09-14 mostrou o
// teste passando com a contagem de marca combinante quebrada.
func colunasNaTela(s string) int {
	n := 0
	for _, r := range s {
		if unicode.In(r, unicode.Mn, unicode.Me) {
			continue
		}
		n++
	}
	return n
}

// TestMolduraFechaComCaminhoEmNFD e a forma medida em 2026-09-14: na busca, a
// linha com um caminho em NFD saiu 4 colunas mais curta que a borda.
func TestMolduraFechaComCaminhoEmNFD(t *testing.T) {
	var buf bytes.Buffer
	s := NewPlain(&buf)
	forcarGlifos = &glifosASCII
	defer func() { forcarGlifos = nil }()

	nfd := "A" + "c" + string(rune(0x0327)) + "a" + string(rune(0x0303)) + "o Civil/Prescri" +
		"c" + string(rune(0x0327)) + "a" + string(rune(0x0303)) + "o.md"
	s.Bloco("1 de 1", []string{"  " + nfd + "  0.85", "    ... # Prescricao ..."}, "")

	linhas := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	quer := colunasNaTela(linhas[0])
	for i, l := range linhas {
		if got := colunasNaTela(l); got != quer {
			t.Errorf("linha %d com %d colunas na tela, a borda tem %d:\n%s", i, got, quer, buf.String())
		}
	}
}
