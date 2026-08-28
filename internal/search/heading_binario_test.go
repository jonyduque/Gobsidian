package search

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

// TestDentroDeHeading cobre a troca de varredura linear por busca binária
// (achado P2).
//
// A varredura antiga testava TODOS os headings e devolvia verdadeiro no
// primeiro que casasse. A busca binária pega o último com `Start <= pos.Start`
// e confere só ele — o que é correto porque as linhas de heading não se
// sobrepõem, mas depende dos headings estarem em ordem de documento, que é como
// o parser os produz. Se essa premissa cair, é aqui que aparece.
//
// Os offsets são montados à mão para que o teste não dependa do parser: o que
// se testa é a decisão sobre offsets, não como eles foram obtidos.
func TestDentroDeHeading(t *testing.T) {
	// "# Um" na posição 0, "## Dois" na 100, "### Tres" na 200.
	// Fim da linha = Start + nivel + 1 (espaço) + len(texto).
	hs := []parser.Heading{
		{Level: 1, Text: "Um", Start: 0},     // linha 0..4
		{Level: 2, Text: "Dois", Start: 100}, // linha 100..107
		{Level: 3, Text: "Tres", Start: 200}, // linha 200..208
	}

	casos := []struct {
		nome  string
		start int64
		quer  bool
	}{
		{"antes de tudo", -1, false},
		{"dentro do primeiro", 2, true},
		{"logo apos o primeiro", 5, false},
		{"corpo entre o primeiro e o segundo", 50, false},
		{"dentro do do meio", 103, true},
		{"logo apos o do meio", 108, false},
		{"dentro do ultimo", 205, true},
		{"depois do ultimo", 300, false},
		// A busca binaria devolve o ULTIMO heading com Start <= pos. Se ela
		// devolvesse o primeiro, ou parasse no meio, este caso e o de 205
		// discordariam.
		{"exatamente no Start do ultimo", 200, true},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := dentroDeHeading(hs, TokenPosition{Start: c.start, End: c.start + 1}); got != c.quer {
				t.Errorf("dentroDeHeading(start=%d) = %v, queria %v", c.start, got, c.quer)
			}
		})
	}

	if dentroDeHeading(nil, TokenPosition{Start: 0, End: 1}) {
		t.Error("nota sem heading nenhum devolveu verdadeiro")
	}
}
