package search_test

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/index"
	"github.com/jonyduque/Gobsidian/internal/search"
	"github.com/jonyduque/Gobsidian/internal/vault"
)

func createVaultWithNotes(t *testing.T, files map[string]string) (*vault.Vault, *index.Index, *search.Inverted) {
	t.Helper()
	root := t.TempDir()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	ix := search.NewInverted()

	for relPath, content := range files {
		full := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		cPath := vault.CanonicalPath(relPath)
		ix.Add(string(cPath), search.Analyze(content))
	}

	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	return v, idx, ix
}

func TestBM25FieldWeightsAreApplied(t *testing.T) {
	// Duas notas, mesmo termo, mesma frequencia, mesma extensao. A unica
	// diferenca e ONDE o termo aparece. Se os pesos nao forem aplicados, os
	// scores empatam — e um empate aqui e indistinguivel de "funciona",
	// porque a lista sai ordenada de qualquer jeito.
	_, idx, ix := createVaultWithNotes(t, map[string]string{
		"titulo.md": "---\ntitle: prescricao\n---\n# Titulo\n\ntexto texto\n",
		"corpo.md":  "---\ntitle: outra\n---\n# Outra\n\nprescricao texto\n",
	})

	res := search.CalculateBM25(search.Analyze("prescricao"), ix, idx)
	if len(res) < 2 {
		t.Fatalf("res = %+v, quer 2 resultados", res)
	}

	var noTitulo, noCorpo float64
	for _, r := range res {
		switch r.Path {
		case "titulo.md":
			noTitulo = r.Score
		case "corpo.md":
			noCorpo = r.Score
		}
	}

	if noTitulo <= noCorpo {
		t.Fatalf("titulo=%.4f corpo=%.4f — o peso 3x do titulo nao esta sendo aplicado",
			noTitulo, noCorpo)
	}
	if r := noTitulo / noCorpo; r < 1.3 {
		t.Errorf("razao titulo/corpo = %.2f; peso 3x deveria separar mais", r)
	}
}

func TestBM25WeightTitle(t *testing.T) {
	_, idx, ix := createVaultWithNotes(t, map[string]string{
		"t.md": "---\ntitle: civil\n---\n# Nota T\n\nconteudo\n",
		"c.md": "---\ntitle: outro\n---\n# Nota C\n\ncivil\n",
	})

	res := search.CalculateBM25(search.Analyze("civil"), ix, idx)
	if len(res) != 2 {
		t.Fatalf("len(res) = %d, quer 2", len(res))
	}

	if res[0].Path != "t.md" || res[0].Score <= res[1].Score*1.3 {
		t.Errorf("WeightTitle falhou: res = %+v", res)
	}
}

func TestBM25WeightHeadings(t *testing.T) {
	// As duas notas tem os MESMOS tokens, na mesma quantidade — so a POSICAO de
	// "civil" muda: heading numa, corpo na outra.
	//
	// O fixture anterior era "## civil / texto texto" contra "# Nota C / texto
	// civil", e as duas tinham comprimentos diferentes. O BM25 normaliza por
	// comprimento, entao a nota mais curta ja pontuava mais SEM peso nenhum de
	// heading: uma prova de mutacao em 2026-08-28 apagou a deteccao de heading e
	// este teste PASSOU. Teste que nao pode falhar e pior que teste ausente.
	_, idx, ix := createVaultWithNotes(t, map[string]string{
		"h.md": "---\ntitle: Nota\n---\n## civil\n\nalpha beta gama\n",
		"c.md": "---\ntitle: Nota\n---\n## alpha\n\ncivil beta gama\n",
	})

	res := search.CalculateBM25(search.Analyze("civil"), ix, idx)
	if len(res) != 2 {
		t.Fatalf("len(res) = %d, quer 2", len(res))
	}

	var scoreH, scoreC float64
	for _, r := range res {
		switch r.Path {
		case "h.md":
			scoreH = r.Score
		case "c.md":
			scoreC = r.Score
		}
	}

	if scoreH <= scoreC {
		t.Errorf("WeightHeadings falhou: scoreH=%.4f, scoreC=%.4f", scoreH, scoreC)
	}
}

// TestBM25PesoDeCorpoEOMenorDosTres substitui TestBM25WeightBody, que passava
// `nil` no lugar do índice.
//
// Com `idx == nil`, `pesoDeCampo` devolve `WeightBody` na PRIMEIRA linha, antes
// de olhar título ou heading: toda ocorrência do cofre vale peso de corpo por
// definição. O teste antigo afirmava só que "b1.md" casava e que o score era
// positivo — verdade para qualquer valor de WeightBody maior que zero, e
// verdade também se os três pesos fossem iguais. Ele nomeava um peso de campo e
// exercitava o ramo em que campo nenhum é consultado.
//
// Aqui o índice é real e as três notas têm o MESMO multiconjunto de tokens —
// {title, alpha, beta, gama, civil} —, logo o mesmo comprimento e o mesmo idf
// para "civil". A única coisa que difere é ONDE "civil" está: no título, no
// heading, ou no corpo. O que a ordem estrita afirma é a escala inteira
// (3 > 2 > 1), e o piso dela é WeightBody.
func TestBM25PesoDeCorpoEOMenorDosTres(t *testing.T) {
	_, idx, ix := createVaultWithNotes(t, map[string]string{
		"t.md": "---\ntitle: civil\n---\n## alpha\n\nbeta gama\n",
		"h.md": "---\ntitle: alpha\n---\n## civil\n\nbeta gama\n",
		"c.md": "---\ntitle: alpha\n---\n## beta\n\ncivil gama\n",
	})

	res := search.CalculateBM25(search.Analyze("civil"), ix, idx)
	if len(res) != 3 {
		t.Fatalf("len(res) = %d, quer 3 — as tres notas tem o termo: %+v", len(res), res)
	}

	score := map[string]float64{}
	for _, r := range res {
		score[r.Path] = r.Score
	}

	// Comprimentos iguais não são suposição: se um dia deixarem de ser, a
	// normalização por comprimento explica a ordem sozinha e o teste passa a
	// medir outra coisa.
	if a, b, c := ix.DocLength("t.md"), ix.DocLength("h.md"), ix.DocLength("c.md"); a != b || b != c {
		t.Fatalf("as tres notas precisam ter o mesmo comprimento; t=%d h=%d c=%d — "+
			"com comprimentos diferentes o BM25 ordena pela normalizacao, nao pelo peso de campo", a, b, c)
	}

	if !(score["t.md"] > score["h.md"]) {
		t.Errorf("titulo=%.6f nao ficou acima de heading=%.6f: WeightTitle nao esta separando de WeightHeadings",
			score["t.md"], score["h.md"])
	}
	if !(score["h.md"] > score["c.md"]) {
		t.Errorf("heading=%.6f nao ficou acima de corpo=%.6f: WeightHeadings nao esta separando de WeightBody",
			score["h.md"], score["c.md"])
	}
}

// TestBM25TermFrequencySaturation verifica a presença da curva sublinear de
// saturação por frequência de termo (mecanismo do parâmetro K1).
// Nota: Este teste valida a mecânica de saturação (razão > 1.0 e finita), e não
// engessa o valor numérico exato da constante ParamK1.
func TestBM25TermFrequencySaturation(t *testing.T) {
	// Duas notas de mesmo comprimento (5 tokens)
	ix := search.NewInverted()
	ix.Add("tf1.md", search.Analyze("prescricao a b c d"))
	ix.Add("tf5.md", search.Analyze("prescricao prescricao prescricao prescricao prescricao"))

	res := search.CalculateBM25(search.Analyze("prescricao"), ix, nil)
	if len(res) != 2 {
		t.Fatalf("len(res) = %d, quer 2", len(res))
	}

	var s1, s5 float64
	for _, r := range res {
		switch r.Path {
		case "tf1.md":
			s1 = r.Score
		case "tf5.md":
			s5 = r.Score
		}
	}

	ratio := s5 / s1
	if ratio < 1.2 || ratio > 2.2 {
		t.Errorf("saturação de frequência fora do esperado: ratio = %.4f", ratio)
	}
}

// TestBM25DocumentLengthNormalization verifica que a penalização por
// comprimento do documento está ativa (mecanismo do parâmetro B).
// Nota: Este teste valida que documentos mais curtos ordenam acima de mais
// longos para o mesmo termo, e não engessa o valor numérico exato de ParamB.
func TestBM25DocumentLengthNormalization(t *testing.T) {
	ix := search.NewInverted()
	ix.Add("curta.md", search.Analyze("prescricao civil"))
	ix.Add("longa.md", search.Analyze("prescricao a b c d e f g h i j k l m n o p q r s"))

	res := search.CalculateBM25(search.Analyze("prescricao"), ix, nil)
	if len(res) != 2 {
		t.Fatalf("len(res) = %d, quer 2", len(res))
	}

	if res[0].Path != "curta.md" {
		t.Errorf("penalização de comprimento falhou: res[0] = %s", res[0].Path)
	}
	ratio := res[0].Score / res[1].Score
	if ratio < 1.3 {
		t.Errorf("razão de penalidade = %.4f, quer > 1.3", ratio)
	}
}

func TestBM25RawVsReduced(t *testing.T) {
	// Termo cru ("prescricoes") pontua mais alto que termo reduzido ("prescricao") para busca por "prescricoes"
	ix := search.NewInverted()

	ix.Add("nota_raw.md", search.Analyze("prescrições"))
	ix.Add("nota_red.md", search.Analyze("prescrição"))

	res := search.CalculateBM25(search.Analyze("prescrições"), ix, nil)
	if len(res) != 2 {
		t.Fatalf("len(res) = %d, quer 2", len(res))
	}

	if res[0].Path != "nota_raw.md" {
		t.Errorf("Forma crua deveria pontuar acima da reduzida: res[0] = %s (score %.4f vs %.4f)",
			res[0].Path, res[0].Score, res[1].Score)
	}
}

// TestBM25TermoEmTodasAsNotasAindaPontua substitui TestBM25FrequentTermNoNaN,
// que so varria os resultados procurando NaN e por isso nao podia falhar:
// bm25.go monta a lista dentro de `if !math.IsNaN(score) && score > 0`, entao
// um NaN nunca chega ao laco. Medido — com o guarda trocado por
// `if score > 0 {`, o teste antigo continuava verde.
//
// A regra que este cenario CONSEGUE falsificar e a que importa para um termo
// de parada: o idf e `log(1 + (N-d+0.5)/(d+0.5))`, e o `1 +` existe para que
// d == N — termo presente em TODAS as notas — ainda produza idf positivo. Sem
// ele o idf fica negativo, o `idf <= 0` descarta o termo, e a busca por um
// termo comum devolve lista vazia em vez de um ranking.
//
// Este teste NAO afirma a ordem entre a.md e b.md, e a linha que afirmava foi
// removida de proposito. Ela passava por uma margem de ~3%, e a margem nao vem
// da regra que este teste nomeia: com ParamK1 = 1.2 e ParamB = 0.75
// (bm25.go:19-20), a fracao de tf/comprimento vale 1.507 para "de de de"
// (tf 3, dl 3) contra 1.457 para "de de" (tf 2, dl 2, avgdl 2.5) — conta
// derivada da formula, nao medida em execucao, e conferida contra a mesma
// conta feita na revisao. Tres ocorrencias ganham de duas porque a
// normalizacao de comprimento quase anula a frequencia maior; mexer no
// comprimento de qualquer uma das duas notas inverte o resultado sem que a
// regra do idf tenha mudado. Quem mata a mutacao do `1 +` e o Fatalf de
// `len(res) != 2`.
func TestBM25TermoEmTodasAsNotasAindaPontua(t *testing.T) {
	ix := search.NewInverted()
	ix.Add("a.md", search.Analyze("de de de"))
	ix.Add("b.md", search.Analyze("de de"))

	res := search.CalculateBM25(search.Analyze("de"), ix, nil)
	if len(res) != 2 {
		t.Fatalf("len(res) = %d, quer 2 — termo presente em todas as notas sumiu do resultado", len(res))
	}
	for _, r := range res {
		if math.IsNaN(r.Score) || r.Score <= 0 {
			t.Errorf("Score de %s = %v, quer finito e positivo", r.Path, r.Score)
		}
	}
}

func TestBM25DeterministicTieBreaking(t *testing.T) {
	ix := search.NewInverted()
	ix.Add("z.md", search.Analyze("usucapiao"))
	ix.Add("a.md", search.Analyze("usucapiao"))

	res := search.CalculateBM25(search.Analyze("usucapiao"), ix, nil)
	if len(res) != 2 {
		t.Fatalf("len(res) = %d, quer 2", len(res))
	}
	if res[0].Score != res[1].Score {
		t.Fatalf("esperava empate de score, obteve %f e %f", res[0].Score, res[1].Score)
	}
	if res[0].Path != "a.md" || res[1].Path != "z.md" {
		t.Errorf("desempate deterministico falhou: obteve %s, %s; quer a.md, z.md",
			res[0].Path, res[1].Path)
	}
}
