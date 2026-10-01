package console

import (
	"bytes"
	"strings"
	"testing"
	"unicode"
)

// colunasComLarguraDupla conta as colunas de uma linha SEM ANSI por uma conta
// propria do teste, como colunasNaTela: medir com larguraVisivel deixaria o
// mesmo erro dos dois lados. Aqui a tabela e a minima que o teste usa --
// ideogramas CJK unificados, kana e formas de largura total ocupam duas
// colunas.
func colunasComLarguraDupla(s string) int {
	n := 0
	for _, r := range s {
		switch {
		case unicode.In(r, unicode.Mn, unicode.Me):
		case (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3040 && r <= 0x30FF) || (r >= 0xFF01 && r <= 0xFF60):
			n += 2
		default:
			n++
		}
	}
	return n
}

// TestLarguraVisivelContaLarguraDupla: ideograma ocupa duas colunas no
// terminal, e contar uma desalinhava a borda da moldura a cada caractere.
func TestLarguraVisivelContaLarguraDupla(t *testing.T) {
	for _, c := range []struct {
		s    string
		quer int
	}{
		{"日本語", 6},
		{"ノート.md", 9},
		{"ＡＢ", 4},
		{"a日b", 4},
	} {
		if got := larguraVisivel(c.s); got != c.quer {
			t.Errorf("larguraVisivel(%q) = %d, quer %d", c.s, got, c.quer)
		}
	}
}

// TestMolduraFechaComNomeEmJapones e a forma que ESTADO.md registrava como
// divida: um cofre com nome em japones desalinha a borda.
func TestMolduraFechaComNomeEmJapones(t *testing.T) {
	var buf bytes.Buffer
	s := NewPlain(&buf)
	forcarGlifos = &glifosASCII
	defer func() { forcarGlifos = nil }()

	s.Bloco("1 de 2", []string{"  法律/民法.md  0.85", "  Civil/Dolo.md  0.40"}, "")

	linhas := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	quer := colunasComLarguraDupla(linhas[0])
	for i, l := range linhas {
		if got := colunasComLarguraDupla(l); got != quer {
			t.Errorf("linha %d com %d colunas na tela, a borda tem %d:\n%s", i, got, quer, buf.String())
		}
	}
}

// TestAjustarCortaSemPartirIdeograma: o corte para no ultimo caractere que
// cabe inteiro, e o que sobrar de uma coluna e preenchido -- senao a linha
// cortada sai uma coluna mais curta que a borda.
func TestAjustarCortaSemPartirIdeograma(t *testing.T) {
	forcarGlifos = &glifosASCII
	defer func() { forcarGlifos = nil }()

	for largura := 4; largura <= 9; largura++ {
		got := ajustar("日本語の文章です", largura, " ")
		if l := colunasComLarguraDupla(got); l != largura {
			t.Errorf("ajustar(..., %d) = %q com %d colunas", largura, got, l)
		}
	}
}

// TestCortarMantemAMarcaCombinante: cortar e larguraVisivel sao a mesma conta.
// cortar contava a marca combinante como coluna e parava antes dela, e "Ação"
// em NFD cortado em 2 colunas perdia a cedilha.
func TestCortarMantemAMarcaCombinante(t *testing.T) {
	nfd := "A" + "c" + string(rune(0x0327)) + "a" + string(rune(0x0303)) + "o"
	quer := "A" + "c" + string(rune(0x0327))
	if got := cortar(nfd, 2); got != quer {
		t.Errorf("cortar(Acao em NFD, 2) = %q, quer %q", got, quer)
	}
}
