package search_test

import (
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/search"
)

// BenchmarkAnalyzeCorpoTipico mede o achado P12 na forma em que ele custa: a
// tokenização roda sobre o cofre INTEIRO na indexação, e é proporcional aos
// bytes, não à contagem de notas.
func BenchmarkAnalyzeCorpoTipico(b *testing.B) {
	// ~35 KB, o tamanho de corpo que gen_vault.ps1 usa no cofre de referência.
	// Mistura acentos, porque um corpo 100% ASCII mediria só o melhor caso.
	trecho := "O acordao do RESP 1234 tratou da prescricao intercorrente e da execucao fiscal. " +
		"Ver tambem a Sumula 106 do STJ e o artigo 40 da Lei 6.830/80. "
	corpo := strings.Repeat(trecho, 35000/len(trecho)+1)
	b.SetBytes(int64(len(corpo)))
	b.ResetTimer()
	for b.Loop() {
		if len(search.Analyze(corpo)) == 0 {
			b.Fatal("nenhum token")
		}
	}
}
