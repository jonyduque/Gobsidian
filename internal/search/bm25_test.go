package search_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
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

// escreve é um helper que escreve conteúdo num arquivo.
func escreve(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// servicoCompleto monta um *search.Inverted a partir de um diretório já
// populado e um vault+index, como exigido pelo teste de cache.
func servicoCompleto(t *testing.T, root string) (*search.Inverted, *vault.Vault, *index.Index) {
	t.Helper()

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		if err := inv.Update(context.Background(), v, p); err != nil {
			t.Fatalf("inv.Update: %v", err)
		}
	}

	return inv, v, idx
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
	_, idx, ix := createVaultWithNotes(t, map[string]string{
		"h.md": "---\ntitle: Nota H\n---\n## civil\n\ntexto texto\n",
		"c.md": "---\ntitle: Nota C\n---\n# Nota C\n\ntexto civil\n",
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

func TestBM25WeightBody(t *testing.T) {
	ix := search.NewInverted()
	ix.Add("b1.md", search.Analyze("usucapiao"))
	ix.Add("b2.md", search.Analyze("outra palavra"))

	res := search.CalculateBM25(search.Analyze("usucapiao"), ix, nil)
	if len(res) != 1 || res[0].Path != "b1.md" {
		t.Fatalf("WeightBody falhou: %+v", res)
	}
	if res[0].Score <= 0 {
		t.Errorf("WeightBody produziu score <= 0: %f", res[0].Score)
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

func TestBM25FrequentTermNoNaN(t *testing.T) {
	ix := search.NewInverted()
	ix.Add("a.md", search.Analyze("de de de"))
	ix.Add("b.md", search.Analyze("de de"))

	res := search.CalculateBM25(search.Analyze("de"), ix, nil)
	for _, r := range res {
		if r.Score != r.Score { // NaN check
			t.Errorf("Score = NaN para termo frequente na nota %s", r.Path)
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

// TestAvgdlInvalidaComAGeracao guarda o defeito que o cache INTRODUZ.
//
// avgdl obsoleto nao levanta erro: muda o score em silencio. O teste adiciona
// uma nota MUITO longa depois da primeira busca — se o cache nao invalidar, o
// avgdl continua o da corpus pequeno e a normalizacao por tamanho fica errada.
func TestAvgdlInvalidaComAGeracao(t *testing.T) {
	root := t.TempDir()
	escreve(t, root, "a.md", "# A\n\nprescricao\n")
	escreve(t, root, "b.md", "# B\n\nprescricao prescricao\n")
	inv, v, idx := servicoCompleto(t, root)

	// A asserção que importa é sobre o avgdl, não sobre o score.
	//
	// A primeira versão deste teste comparava só `antes[0].Score` com
	// `depois[0].Score` e exigia que fossem diferentes. Isso NÃO cobre a
	// invalidação: o corpus passa de duas para três notas, então o IDF muda
	// sozinho e os scores diferem mesmo servindo um avgdl obsoleto. Medido —
	// removendo `gen == ix.avgdlGen` da condição de cache, aquele teste
	// continuava passando.
	//
	// GetAvgdl é lido direto porque é o único valor que a regra promete
	// atualizar. Uma chamada antes popula o cache; se a geração não
	// invalidar, a chamada depois devolve o mesmo número.
	avgdlAntes := inv.GetAvgdl(idx)
	antes := search.CalculateBM25(search.Analyze("prescricao"), inv, idx)

	// Nota longa: muda avgdl de forma detectavel.
	longa := "# C\n\n" + strings.Repeat("palavra ", 5000) + "prescricao\n"
	escreve(t, root, "c.md", longa)
	if err := idx.Replace(context.Background(), v, "c.md"); err != nil {
		t.Fatal(err)
	}
	if err := inv.Update(context.Background(), v, "c.md"); err != nil {
		t.Fatal(err)
	}

	avgdlDepois := inv.GetAvgdl(idx)
	if avgdlDepois == avgdlAntes {
		t.Errorf("GetAvgdl = %.6f antes e depois de entrar uma nota de 5000 "+
			"palavras: o cache nao foi invalidado pela geracao", avgdlAntes)
	}
	if avgdlDepois <= avgdlAntes {
		t.Errorf("avgdl caiu de %.6f para %.6f depois de entrar a nota mais "+
			"longa do corpus; o valor esta errado, nao so obsoleto",
			avgdlAntes, avgdlDepois)
	}

	depois := search.CalculateBM25(search.Analyze("prescricao"), inv, idx)
	if len(antes) == 0 || len(depois) == 0 {
		t.Fatal("busca vazia nos dois lados nao prova nada")
	}
	if antes[0].Score == depois[0].Score {
		t.Errorf("score de %q identico (%.6f) depois de o corpus mudar de "+
			"tamanho: avgdl nao foi invalidado",
			antes[0].Path, antes[0].Score)
	}
}
