package search_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/jonyd/gobsidian/internal/search"
)

func TestAnalyzerDualIndexing(t *testing.T) {
	// "prescricoes" reduz para "prescricao"; "civil" nao reduz. As duas
	// formas do primeiro tem de sair, e a do segundo tem de sair com
	// Reduced vazio — nao com Reduced == Raw, que dobraria a posting list.
	toks := search.Analyze("Prescrições civil")

	if len(toks) != 2 {
		t.Fatalf("tokens = %d, quer 2: %+v", len(toks), toks)
	}

	if toks[0].Raw != "prescricoes" {
		t.Errorf("Raw = %q, quer %q — acento e caixa nao foram normalizados",
			toks[0].Raw, "prescricoes")
	}
	if toks[0].Reduced == "" || toks[0].Reduced == toks[0].Raw {
		t.Errorf("Reduced = %q; plural regular tem de reduzir e diferir da forma crua",
			toks[0].Reduced)
	}
	if toks[1].Reduced != "" {
		t.Errorf("Reduced = %q para termo que nao reduz; quer vazio, senao a "+
			"posting list recebe o mesmo termo duas vezes", toks[1].Reduced)
	}

	// Offsets em BYTES sobre o texto original, com acento. "Prescrições"
	// tem 11 runes e 13 bytes; um offset em runes recortaria no meio de "ç".
	if got := "Prescrições"[toks[0].Start:toks[0].End]; got != "Prescrições" {
		t.Errorf("fatia por offset = %q, quer o token inteiro — offset em rune, nao byte", got)
	}
}

func TestNormalizeAccentsAndCase(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"usucapiao", "usucapiao"},
		{"usucapião", "usucapiao"},
		{"USUCAPIÃO", "usucapiao"},
		{"Prescrição", "prescricao"},
		{"AÇÃO", "acao"},
	}

	for _, tc := range cases {
		norm := search.Normalize(tc.input)
		if norm != tc.want {
			t.Errorf("Normalize(%q) = %q, quer %q", tc.input, norm, tc.want)
		}
	}
}

func TestLegalTermsDistinctiveness(t *testing.T) {
	// "detenção" e "posse" ou termos jurídicos que diferem por sufixo
	// "posse" vs "possessória" -> "posse" (raw: posse, red: ""), "possessoria" (raw: possessoria, red: "")
	// Devem produzir formas cruas distintas para não haver colisão indevida.
	t1 := search.Analyze("posse")
	t2 := search.Analyze("possessória")

	if len(t1) != 1 || len(t2) != 1 {
		t.Fatalf("esperava 1 token cada, obteve %d e %d", len(t1), len(t2))
	}

	if t1[0].Raw == t2[0].Raw || (t2[0].Reduced != "" && t1[0].Raw == t2[0].Reduced) {
		t.Errorf("posse (%q) e possessória (%q / red %q) colidiram indevidamente",
			t1[0].Raw, t2[0].Raw, t2[0].Reduced)
	}
}

func TestPunctuationHyphenNumbers(t *testing.T) {
	// "guarda-chuva Art. 1.234/2026"
	toks := search.Analyze("guarda-chuva Art. 1.234/2026")

	expectedRaws := []string{"guarda", "chuva", "art", "1", "234", "2026"}
	if len(toks) != len(expectedRaws) {
		t.Fatalf("len(toks) = %d, quer %d: %+v", len(toks), len(expectedRaws), toks)
	}

	for i, want := range expectedRaws {
		if toks[i].Raw != want {
			t.Errorf("toks[%d].Raw = %q, quer %q", i, toks[i].Raw, want)
		}
	}
}

func TestBOMStrippedTextOffsets(t *testing.T) {
	// Texto simulando pós-vault.StripBOM
	bodyText := "# Título da Nota\n\nPrescrição civil\n"
	toks := search.Analyze(bodyText)

	found := false
	for _, tok := range toks {
		if tok.Raw == "prescricao" {
			found = true
			sliced := bodyText[tok.Start:tok.End]
			if sliced != "Prescrição" {
				t.Errorf("sliced = %q, quer %q", sliced, "Prescrição")
			}
		}
	}
	if !found {
		t.Error("token prescricao não foi encontrado")
	}
}

// TestNormalizeNaoVazaEstadoEntreUsos guarda o defeito que o pool INTRODUZ.
//
// transform.Transformer carrega estado interno. Devolvido ao pool sem Reset, o
// uso seguinte comeca no meio da conversao anterior — e o sintoma nao e lentidao,
// e uma string ERRADA, so sob concorrencia, so as vezes.
//
// Alterna entrada longa e acentuada com entrada curta, em varias goroutines, e
// exige o mesmo resultado que a versao sequencial daria.
func TestNormalizeNaoVazaEstadoEntreUsos(t *testing.T) {
	casos := []struct{ entrada, quer string }{
		{"Prescrição Intercorrente Execução Fiscal Ação Órgão", "prescricao intercorrente execucao fiscal acao orgao"},
		{"a", "a"},
		{"ÁÉÍÓÚÃÕÇ", "aeiouaoc"},
		{"sem acento nenhum aqui", "sem acento nenhum aqui"},
		{"É", "e"},
	}
	// Sequencial primeiro: se isto falhar, o defeito nao e de concorrencia.
	for _, c := range casos {
		if got := search.Normalize(c.entrada); got != c.quer {
			t.Fatalf("Normalize(%q) = %q, quer %q", c.entrada, got, c.quer)
		}
	}

	const goroutines = 32
	const voltas = 1000
	var wg sync.WaitGroup
	erros := make(chan string, goroutines*voltas)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for v := 0; v < voltas; v++ {
				c := casos[(g+v)%len(casos)]
				if got := search.Normalize(c.entrada); got != c.quer {
					erros <- fmt.Sprintf("Normalize(%q) = %q, quer %q", c.entrada, got, c.quer)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(erros)
	for e := range erros {
		t.Error(e)
	}
}
