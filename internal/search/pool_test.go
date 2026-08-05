package search_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/search"
)

func TestPoolReuse(t *testing.T) {
	// Teste simples: chamar Normalize múltiplas vezes
	// Se o pool estiver funcionando, alocs deve ser menor que criar novo cada vez
	s1 := search.Normalize("PRESCRIÇÃO")
	s2 := search.Normalize("É")
	s3 := search.Normalize("ÁÉÍÓÚ")

	if s1 != "prescricao" || s2 != "e" || s3 != "aeiou" {
		t.Fatalf("Resultados incorretos: %q, %q, %q", s1, s2, s3)
	}
}
