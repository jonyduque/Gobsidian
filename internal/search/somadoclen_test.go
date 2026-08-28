package search_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/search"
)

// TestSomaDocLenMemorizaEInvalida cobre o achado P1, e cobre a metade perigosa.
//
// `avgdl` — o divisor da normalização por comprimento do BM25 — era calculado
// por consulta percorrendo `idx.Paths()`, que aloca e ordena o cofre inteiro.
// Agora vem memorizado contra a geração do índice.
//
// **O risco da correção é maior que o do defeito.** Memorizar sem invalidar
// congela o `avgdl` no valor da partida: o índice cresce, a normalização
// continua dividindo pelo número velho, e o ranking degrada em silêncio —
// nenhum erro, nenhum log, só resultados progressivamente piores. Por isso o
// teste cobre as duas direções.
func TestSomaDocLenMemorizaEInvalida(t *testing.T) {
	ix := search.NewInverted()
	ix.Add("a.md", search.Analyze("um dois tres"))

	primeira := ix.SomaDocLen()
	if primeira <= 0 {
		t.Fatalf("SomaDocLen = %d: o cenario nao exercita nada", primeira)
	}
	if segunda := ix.SomaDocLen(); segunda != primeira {
		t.Errorf("duas chamadas seguidas sem mutacao deram %d e %d", primeira, segunda)
	}

	// Cresce o índice: a soma TEM de acompanhar.
	ix.Add("b.md", search.Analyze("quatro cinco seis sete"))
	depoisDoAdd := ix.SomaDocLen()
	if depoisDoAdd <= primeira {
		t.Errorf("SomaDocLen = %d depois de acrescentar uma nota, era %d antes\n"+
			"a memorizacao nao invalidou no Add: avgdl congela e o ranking degrada em silencio",
			depoisDoAdd, primeira)
	}

	// E encolhe: Remove também é mutação.
	ix.Remove("b.md")
	depoisDoRemove := ix.SomaDocLen()
	if depoisDoRemove != primeira {
		t.Errorf("SomaDocLen = %d depois de remover a nota acrescentada, era %d antes dela\n"+
			"a memorizacao nao invalidou no Remove", depoisDoRemove, primeira)
	}
}

// TestSomaDocLenIgualAoPercursoDireto fixa o VALOR, e não só a invalidação.
//
// Sem isto, uma memorização que devolvesse qualquer número estável passaria no
// teste acima nas duas primeiras asserções. A soma tem de bater com o percurso
// direto — que é o que o código antigo fazia, e é a definição do campo.
func TestSomaDocLenIgualAoPercursoDireto(t *testing.T) {
	ix := search.NewInverted()
	docs := map[string]string{
		"a.md":     "um dois tres",
		"b.md":     "quatro cinco",
		"c.md":     "seis sete oito nove dez",
		"vazia.md": "",
	}
	var esperado int64
	for path, corpo := range docs {
		ix.Add(path, search.Analyze(corpo))
	}
	for path := range docs {
		esperado += int64(ix.DocLength(path))
	}

	if got := ix.SomaDocLen(); got != esperado {
		t.Errorf("SomaDocLen = %d, soma direta de DocLength = %d", got, esperado)
	}
	if esperado == 0 {
		t.Fatal("a soma esperada deu zero: o cenario nao exercita nada")
	}
}
