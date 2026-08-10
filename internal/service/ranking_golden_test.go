package service_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

// atualizaGolden regrava testdata/ranking/*.tsv com o que o codigo produz
// hoje. NUNCA usar para fazer um golden divergente passar: -update grava o
// que o codigo produz, nao o que esta certo. Uso legitimo e so quando uma
// mudanca de ranking foi revisada e aprovada por escrito.
var atualizaGolden = flag.Bool("update", false, "regrava os golden files")

// servicoDoCofre monta um *service.Service a partir de um diretorio ja
// populado no disco, incluindo o indice invertido — sem ele TestRankingGolden
// mediria busca so por metadados, e o BM25 nunca entraria em jogo.
func servicoDoCofre(t *testing.T, root string) *service.Service {
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
		if data, err := v.ReadAll(context.Background(), p); err == nil {
			body, _ := vault.StripBOM(data)
			inv.Add(string(p), search.Analyze(string(body)))
		}
	}

	return service.New(v, idx, inv, nil, service.Options{})
}

// corpusGolden monta um cofre determinístico e AFIRMA o próprio tamanho.
//
// A afirmação não é decoração: um corpus que gera menos notas do que o nome diz
// produz um golden menor, que passa, e some com a cobertura sem nada indicar.
//
// DESVIO DELIBERADO do corpo literal do brief da Task 78: o template original
// dava a TODAS as 300 notas o mesmo título, o mesmo heading e a mesma frase
// final, palavra por palavra. Isso deixa "algoritmo BM25 com pesos" e
// "intercorrente" idênticos em toda nota — cada consulta empata nas 300, o
// resultado é cortado no Limit padrão (confirmado: frase-exata.tsv e
// so-no-titulo.tsv saíram com exatamente 20 linhas, o teto do limit, não o
// tamanho real do match) e as duas perguntas que a própria tarefa manda
// conferir ("a frase exata casa uma só?", "a nota com o termo no título vem
// antes da que só tem no corpo?") ficam sem como ser respondidas: não há nota
// alguma cujo corpo sozinho contenha "intercorrente", nem nota alguma sem a
// frase completa. Um golden que empata em tudo passa e não cobre o peso de
// título nem o casamento de frase único — exatamente o "golden que passa com
// o corpus errado" que a linha 3 do brief nomeia como modo de falha. Corrigido
// aqui introduzindo duas notas de contraste; resto do template idêntico ao
// literal (mesmas 300 notas, mesmo layout de pasta, mesmo acento no título).
func corpusGolden(t *testing.T) (*service.Service, string) {
	t.Helper()
	const querNotas = 300

	// tituloComTermo é o pequeno grupo de notas cujo título leva
	// "intercorrente". Pequeno de propósito: se as 300 notas levassem o
	// termo no título, a única nota que só tem o termo no corpo nunca
	// apareceria dentro do Limit padrão (20), e o contraste título-vs-corpo
	// ficaria invisível no golden mesmo existindo no índice.
	tituloComTermo := map[int]bool{5: true, 15: true, 25: true, 35: true, 45: true}
	// notaTermoSoNoCorpo é a ÚNICA nota com "intercorrente" no corpo e não
	// no título — o contraste que a verificação desta tarefa pede.
	const notaTermoSoNoCorpo = 250
	// notaFraseUnica é a ÚNICA nota com a sequência exata "algoritmo BM25 com
	// pesos". Nas demais, as mesmas palavras aparecem fora de ordem, então o
	// termo isolado ainda é encontrável (não afeta termo-amplo/dois-termos)
	// mas a frase entre aspas não casa.
	const notaFraseUnica = 150

	root := t.TempDir()
	for i := 0; i < querNotas; i++ {
		// Acento no título de propósito: é o caminho de Normalize.
		titulo := fmt.Sprintf("Prescrição %04d", i)
		if tituloComTermo[i] {
			titulo = fmt.Sprintf("Prescrição intercorrente %04d", i)
		}

		corpoExtra := ""
		if i == notaTermoSoNoCorpo {
			corpoExtra = " O termo intercorrente aparece apenas aqui, no corpo."
		}

		fraseFinal := fmt.Sprintf(
			"O algoritmo de busca usa BM25 e pesos diferentes aqui quando %d.", i%13)
		if i == notaFraseUnica {
			fraseFinal = fmt.Sprintf("O algoritmo BM25 com pesos aparece aqui quando %d.", i%13)
		}

		corpo := fmt.Sprintf(
			"---\ntags: [t%d]\n---\n\n# %s\n\n"+
				"## Execução fiscal\n\nnota %04d sobre prescricao e execucao.%s %s\n",
			i%7, titulo, i, corpoExtra, fraseFinal)
		dir := filepath.Join(root, fmt.Sprintf("pasta%02d", i%10))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("n%04d.md", i)),
			[]byte(corpo), 0644); err != nil {
			t.Fatal(err)
		}
	}
	svc := servicoDoCofre(t, root)
	stats, err := svc.VaultStats(context.Background(), service.StatsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Notes != querNotas {
		t.Fatalf("corpus tem %d notas, quer %d — golden gerado sobre corpus "+
			"errado passa e nao cobre nada", stats.Notes, querNotas)
	}
	return svc, root
}

var consultasGolden = []struct {
	nome string
	opts service.SearchOptions
}{
	{"termo-amplo", service.SearchOptions{Query: "nota", Limit: 50}},
	{"dois-termos", service.SearchOptions{Query: "prescricao execucao"}},
	{"frase-exata", service.SearchOptions{Query: `"algoritmo BM25 com pesos"`}},
	{"com-acento", service.SearchOptions{Query: "Prescrição"}},
	{"so-no-titulo", service.SearchOptions{Query: "intercorrente"}},
	{"so-em-heading", service.SearchOptions{Query: "fiscal"}},
}

func TestRankingGolden(t *testing.T) {
	svc, _ := corpusGolden(t)
	for _, c := range consultasGolden {
		t.Run(c.nome, func(t *testing.T) {
			res, err := svc.Search(context.Background(), c.opts)
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Results) == 0 {
				t.Fatal("consulta nao casou nada: golden vazio passa sempre " +
					"e nao cobre ranking nenhum")
			}
			var b strings.Builder
			for _, r := range res.Results {
				fmt.Fprintf(&b, "%s\t%.6f\n", r.Path, r.Score)
			}
			golden := filepath.Join("testdata", "ranking", c.nome+".tsv")
			if *atualizaGolden {
				if err := os.MkdirAll(filepath.Dir(golden), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, []byte(b.String()), 0644); err != nil {
					t.Fatal(err)
				}
				t.Logf("golden gravado: %s — LEIA antes de commitar", golden)
				return
			}
			bruto, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("golden ausente (%v). Gere com -update e LEIA o "+
					"arquivo antes de commitar", err)
			}
			// Fim de linha nao e dado: as linhas sao caminho + TAB + score, e
			// o que este golden congela e a ORDEM e o VALOR, nada mais. O
			// arquivo e gravado com "\n" acima, mas quem o le pode te-lo
			// recebido em CRLF de um checkout com core.autocrlf=true — o
			// padrao dos runners Windows do CI e da maioria dos clones no
			// Windows. Comparar bytes crus fazia os seis subtestes reprovarem
			// com "ranking mudou" num clone novo, o que parece regressao de
			// ranking e nao e. .gitattributes agora fixa *.tsv em LF; esta
			// normalizacao existe para o teste nao depender DISSO tambem.
			quer := strings.ReplaceAll(string(bruto), "\r\n", "\n")
			if b.String() != quer {
				t.Errorf("ranking mudou.\n--- quer ---\n%s\n--- tem ---\n%s\n"+
					"Golden que muda exige explicacao escrita. NAO regenere "+
					"para fazer passar.", quer, b.String())
			}
		})
	}
}
