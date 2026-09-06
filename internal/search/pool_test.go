package search_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/search"
)

// TestPoolReuse afirma o RESULTADO de Normalize em tres entradas acentuadas,
// e so isso. O nome e o comentario antigo prometiam medir alocacoes ("se o
// pool estiver funcionando, allocs deve ser menor") — nenhuma linha aqui mede
// alocacao, e acreditar no comentario e reportar cobertura que nao existe.
//
// Quem exercita o pool sob concorrencia, que e o risco real de um
// sync.Pool de transformers, e TestNormalizeNaoVazaEstadoEntreUsos em
// analyzer_test.go: 32 goroutines x 1000 voltas alternando entrada longa
// acentuada com entrada curta.
func TestPoolReuse(t *testing.T) {
	s1 := search.Normalize("PRESCRIÇÃO")
	s2 := search.Normalize("É")
	s3 := search.Normalize("ÁÉÍÓÚ")

	if s1 != "prescricao" || s2 != "e" || s3 != "aeiou" {
		t.Fatalf("Resultados incorretos: %q, %q, %q", s1, s2, s3)
	}
}
