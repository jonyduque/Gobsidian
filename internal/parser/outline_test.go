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

// TestPisoDeTituloDescartaRotuloDeFlashcard fixa o piso de 5 caracteres.
//
// O ruido medido nos cofres reais em 2026-09-01 tinha um nome so: `**V:**`, o
// marcador de VERSO de flashcard, sozinho na linha. Ele respondia por 14.940 dos
// 14.940 candidatos de um cofre e por 14.961 dos 22.180 de outro.
//
// A tabela abaixo traz o ruido REAL medido, e nao exemplos inventados.
func TestPisoDeTituloDescartaRotuloDeFlashcard(t *testing.T) {
	casos := []struct {
		nome  string
		linha string
		quer  bool
	}{
		{"marcador de verso", "**V:**", false},
		{"resposta curta", "**Nao.**", false},
		{"resposta em caixa alta", "**SIM.**", false},
		{"numero solto", "**22**", false},
		{"quatro caracteres, no limite de baixo", "**abcd**", false},
		{"cinco caracteres, o piso", "**abcde**", true},
		{"secao numerada de verdade", "**13.1.10 Substituicao**", true},
		{"rotulo legitimo que termina em dois-pontos", "**Jurisprudencia:**", true},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			got := len(DetectCandidates([]byte(c.linha+"\n\ncorpo\n"), 0)) == 1
			if got != c.quer {
				t.Errorf("%q virou candidato = %v, quer %v", c.linha, got, c.quer)
			}
		})
	}
}

// TestPisoDeTituloValeParaSetext e o contrapeso da conta unica: um setext de
// duas letras nao e mais titulo que um negrito de duas letras, e o piso tem de
// valer para as duas formas.
func TestPisoDeTituloValeParaSetext(t *testing.T) {
	curto := DetectCandidates([]byte("V:\n===\n\ncorpo\n"), 0)
	if len(curto) != 0 {
		t.Errorf("setext de 2 caracteres virou candidato: %+v", curto)
	}
	longo := DetectCandidates([]byte("Capitulo Um\n===\n\ncorpo\n"), 0)
	if len(longo) != 1 {
		t.Errorf("setext legitimo foi descartado junto: %+v", longo)
	}
}
