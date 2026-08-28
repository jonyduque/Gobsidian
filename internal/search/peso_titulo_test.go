package search_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/search"
)

// TestPesoDeTituloExigeTermoInteiro é o achado P2 no caminho real de pontuação —
// a metade dele que é RESULTADO ERRADO, não lentidão.
//
// `getFieldWeight` testava `strings.Contains(n.TitleNorm, term)` — substring
// crua. Uma busca por "ar" recebia `WeightTitle`, o maior peso do sistema, de
// uma nota chamada "Barragem", só porque "ar" aparece dentro de "barragem". O
// efeito é uma nota irrelevante subir ao topo, e aparece justamente nas
// consultas curtas, que são as mais comuns.
//
// # Por que este teste cobre UMA metade só
//
// O frontmatter é tokenizado junto com o corpo: `ix.Add(path, Analyze(content))`
// recebe o arquivo inteiro. Uma nota titulada "Ar puro" ganha uma ocorrência de
// "ar" nos tokens do documento **independentemente do peso de campo**, então
// compará-la com uma nota de título neutro não isola nada — ela pontuaria mais
// mesmo com a regra de título apagada. Uma versão anterior deste teste tinha
// exatamente essa asserção, e uma prova de mutação a reprovou.
//
// O que dá para isolar aqui é o caso do PEDAÇO de palavra: "Barragem" e "Zebra"
// não contribuem token "ar" nenhum, então as duas notas têm tokens idênticos
// para esta consulta e a única diferença possível é o peso de campo. A regra
// completa é coberta por `TestTituloContemTermo`, que a testa isolada.
func TestPesoDeTituloExigeTermoInteiro(t *testing.T) {
	const corpo = "\n\nmenciona ar uma vez no corpo aqui.\n"
	_, idx, ix := createVaultWithNotes(t, map[string]string{
		"substring.md": "---\ntitle: Barragem\n---" + corpo,
		"neutra.md":    "---\ntitle: Zebra\n---" + corpo,
	})

	res := search.CalculateBM25(search.Analyze("ar"), ix, idx)
	if len(res) < 2 {
		t.Fatalf("resultados = %d, queria 2; o cenario nao exercita nada: %+v", len(res), res)
	}
	pontos := map[string]float64{}
	for _, r := range res {
		pontos[r.Path] = r.Score
	}

	const epsilon = 1e-9
	if pontos["substring.md"]-pontos["neutra.md"] > epsilon {
		t.Errorf("nota titulada \"Barragem\" pontuou %.4f contra %.4f da neutra na busca \"ar\"\n"+
			"o termo NAO esta no titulo como termo, so como pedaco de \"barragem\",\n"+
			"e mesmo assim recebeu peso de TITULO, o maior do sistema",
			pontos["substring.md"], pontos["neutra.md"])
	}
}
