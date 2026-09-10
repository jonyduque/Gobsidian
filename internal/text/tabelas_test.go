package text_test

import (
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/jonyduque/Gobsidian/internal/text"
)

// TestVersaoDasTabelasEhOMaiorDeUnicodeVersion confere a conta contra a
// constante da stdlib, derivada de outro jeito -- Split em vez de Cut. Repetir
// a MESMA expressao no teste provaria so que ela e igual a si mesma.
func TestVersaoDasTabelasEhOMaiorDeUnicodeVersion(t *testing.T) {
	partes := strings.Split(unicode.Version, ".")
	if len(partes) == 0 {
		t.Fatalf("unicode.Version = %q, sem ponto nenhum", unicode.Version)
	}
	quer, err := strconv.Atoi(partes[0])
	if err != nil {
		t.Fatalf("unicode.Version = %q, primeiro campo nao e numero: %v", unicode.Version, err)
	}
	if got := text.VersaoDasTabelas(); got != quer {
		t.Errorf("VersaoDasTabelas() = %d, unicode.Version = %q pede %d", got, unicode.Version, quer)
	}
}

// TestVersaoDasTabelasNaoEhZero existe porque zero e o valor de fallback, e um
// fallback que passasse despercebido faria toda geracao Unicode parecer a
// mesma -- que e exatamente o defeito que VersaoDasTabelas existe para impedir.
func TestVersaoDasTabelasNaoEhZero(t *testing.T) {
	if text.VersaoDasTabelas() == 0 {
		t.Fatalf("VersaoDasTabelas() = 0 com unicode.Version = %q: o fallback disparou", unicode.Version)
	}
}
