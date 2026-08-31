package parser

import "testing"

// TestParagrafoEmNegritoBordas fixa a regra "sozinho na linha".
//
// Enfase no meio de um paragrafo NAO e titulo, e aceita-la encheria o retorno de
// note_outline de ruido que o cliente teria de filtrar — numa nota convertida de
// livro, negrito no meio de frase e comum.
//
// Os casos degenerados (`**`, `****`, `**  **`) estao aqui porque cada um deles
// e um indice fora de faixa em potencial na fatia do miolo.
func TestParagrafoEmNegritoBordas(t *testing.T) {
	casos := []struct {
		in   string
		quer string
		ok   bool
	}{
		{"**titulo**", "titulo", true},
		{"***titulo***", "titulo", true},
		{"__titulo__", "titulo", true},
		{"  **indentado**  ", "indentado", true},
		{"**a** e **b**", "", false},
		{"texto com **enfase** no meio", "", false},
		{"****", "", false},
		{"**", "", false},
		{"****", "", false},
		{"**  **", "", false},
		{"*italico*", "", false},
	}
	for _, c := range casos {
		got, ok := paragrafoEmNegrito(c.in)
		if ok != c.ok || got != c.quer {
			t.Errorf("paragrafoEmNegrito(%q) = (%q,%v), quer (%q,%v)", c.in, got, ok, c.quer, c.ok)
		}
	}
}
