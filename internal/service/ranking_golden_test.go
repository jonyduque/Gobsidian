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

// enchimento devolve n tokens de preenchimento, separados por espaço.
//
// O vocabulário é pequeno de propósito — doze palavras que se repetem. O que
// tem de variar entre as notas é o COMPRIMENTO; um vocabulário distinto por
// nota daria idf máximo a cada palavra e faria o score depender de quais
// palavras a nota tem, escondendo a normalização por comprimento que este
// corpus existe para exercitar.
func enchimento(n int) string {
	var b strings.Builder
	for j := 0; j < n; j++ {
		if j > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "verbete%02d", j%12)
	}
	return b.String()
}

// corpusGolden monta um cofre determinístico e AFIRMA o próprio tamanho.
//
// A afirmação não é decoração: um corpus que gera menos notas do que o nome diz
// produz um golden menor, que passa, e some com a cobertura sem nada indicar.
//
// # Por que as 300 notas são todas diferentes
//
// A versão anterior deste corpus vinha do corpo literal do brief da Task 78:
// TODAS as 300 notas com o mesmo título, o mesmo heading e a mesma frase final,
// palavra por palavra, mais duas notas de contraste acrescentadas depois. O
// sintoma que se via era "frase-exata.tsv e so-no-titulo.tsv saíram com
// exatamente 20 linhas, o teto do Limit padrão", e ele foi diagnosticado como
// corte pelo limite — o sintoma certo, a causa errada.
//
// A causa era o corpus. Com 299 notas idênticas, toda consulta empatava nas
// 300; o que sobrava para ordenar era o desempate determinístico por caminho, e
// `n0150` — a única cuja frase final tinha três tokens a menos — vencia por ser
// a mais curta, inclusive em `so-em-heading`, cujo heading `## Execução fiscal`
// era igual nas 300. Os goldens congelavam o desempate, não o ranking — mas
// congelar o desempate não é o mesmo que ser inerte ao peso de campo, e a
// primeira versão desta docstring dizia que era: "apagar o peso de heading não
// mudava um byte de nenhum `.tsv`". É falso, e não tinha sido medido.
//
// O que se mede, reproduzindo o corpus antigo com `go test -overlay` contra um
// `bm25.go` com `return WeightHeadings` trocado por `return WeightBody`: apagar
// o peso de heading não movia UMA LINHA de lugar em `.tsv` nenhum, e mudava a
// coluna de score de `so-em-heading` e de `dois-termos` — 40 linhas ao todo. O
// segundo entrava porque o heading antigo era `## Execução fiscal` e "execucao"
// é um dos dois termos daquela consulta. A sequência das 20 linhas saía
// idêntica nas duas execuções, e é só essa parte que o golden congelava.
//
// Agora cada nota difere das outras em comprimento e em quais termos carrega,
// por fórmulas determinísticas — nada de `rand`, que tornaria o golden função
// da semente:
//
//   - enchimento de 20 + ((i*7+29)%60) + i/60 tokens: o comprimento varia, e o
//     BM25 normaliza por comprimento. O `+ i/60` não é enfeite: `(i*7+29)%60`
//     tem período 60 e o corpus tem 300 notas, então cada comprimento sairia
//     repetido cinco vezes — e como 3, 4 e 5 dividem 60, essas cinco notas
//     também concordariam em tf de "nota", em "prescricao" e em "execucao".
//     Elas ficavam byte a byte equivalentes para a consulta `termo-amplo`, os
//     dois primeiros resultados empatavam, e a verificação abaixo reprovava.
//     Medido: com `+ i/60`, os cinco deixam de empatar. O `+29` desloca a fase
//     por outro motivo: sem ele o mínimo caía em `i == 0`, e `n0000` era ao
//     mesmo tempo a nota unicamente mais curta e membro de TODAS as classes de
//     congruência (0 % 3 == 0 % 5 == 0 % 7 == 0 % 11 == 0), de modo que vencia
//     três dos seis goldens por acumular tudo — a forma "a nota mais curta
//     ganha" que esta tarefa saiu justamente para eliminar;
//   - "nota" (consulta `termo-amplo`) aparece 1 + i%4 vezes: a frequência varia;
//   - "prescricao" (consulta `dois-termos`) só em i%3 == 0 e "execucao" só em
//     i%5 == 0: quem tem os dois pontua acima de quem tem um;
//   - o título acentuado "Prescrição" (consulta `com-acento`) só em i%7 == 0:
//     a forma crua no título contra a forma reduzida no corpo;
//   - o heading "Rito fiscal" (consulta `so-em-heading`) só em i%11 == 0, e
//     "fiscal" não aparece no corpo de nota alguma. É o único golden que depende
//     do peso de heading, e só depende dele porque o termo não tem outro lugar
//     de onde vir. O que a ORDEM deste golden mede, porém, é o comprimento:
//     "fiscal" ocorre exatamente uma vez, sempre num heading, em todas as notas
//     que casam, então `WeightHeadings` é fator comum a elas e aparece na coluna
//     de score, não na sequência. Mutar aquele peso muda a segunda coluna deste
//     `.tsv` inteiro e não move nenhuma linha — é assim que ele reprova;
//
// As duas notas de contraste continuam, e respondem as perguntas que a Task 78
// mandava conferir: `tituloComTermo` contra `notaTermoSoNoCorpo` responde "a
// nota com o termo no título vem antes da que só tem no corpo?", e
// `notaFraseUnica` responde "a frase exata casa uma só?".
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
		titulo := fmt.Sprintf("Registro %04d", i)
		if i%7 == 0 {
			// Acento no título de propósito: é o caminho de Normalize, e é a
			// forma CRUA que `com-acento` procura. As notas com i%3 == 0 trazem
			// a forma reduzida no corpo; é o contraste que `com-acento` mede.
			titulo = fmt.Sprintf("Prescrição %04d", i)
		}
		if tituloComTermo[i] {
			titulo += " intercorrente"
		}

		// "fiscal" só existe aqui, e em nota alguma no corpo: é o que faz
		// `so-em-heading` depender do peso de heading e de mais nada.
		heading := "Andamento"
		if i%11 == 0 {
			heading = "Rito fiscal"
		}

		var corpo strings.Builder
		fmt.Fprintf(&corpo, "---\ntags: [t%d]\n---\n\n# %s\n\n## %s\n\n",
			i%7, titulo, heading)
		for n := 0; n < 1+i%4; n++ {
			corpo.WriteString("nota ")
		}
		fmt.Fprintf(&corpo, "%04d ", i)
		if i%3 == 0 {
			corpo.WriteString("prescricao ")
		}
		if i%5 == 0 {
			corpo.WriteString("execucao ")
		}
		if i == notaTermoSoNoCorpo {
			corpo.WriteString("intercorrente ")
		}
		corpo.WriteString(enchimento(20 + (i*7+29)%60 + i/60))

		fraseFinal := fmt.Sprintf(
			" O algoritmo de busca usa BM25 e pesos diferentes aqui quando %d.\n", i%13)
		if i == notaFraseUnica {
			fraseFinal = fmt.Sprintf(" O algoritmo BM25 com pesos aparece aqui quando %d.\n", i%13)
		}
		corpo.WriteString(fraseFinal)

		dir := filepath.Join(root, fmt.Sprintf("pasta%02d", i%10))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("n%04d.md", i)),
			[]byte(corpo.String()), 0644); err != nil {
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

// conferirDiscriminacao reprova um corpus que empata antes de o empate virar
// golden.
//
// Enquanto as 300 notas foram idênticas, todo `.tsv` era uma lista de scores
// iguais em que só o desempate determinístico por caminho decidia a ordem:
// apagar o peso de heading não movia UMA LINHA de lugar em arquivo nenhum — só
// reescrevia a coluna de score de `so-em-heading` e de `dois-termos`. Um golden
// que congela um desempate por caminho é caro de manter e não cobre ranking. Estas duas afirmações são o que impede a regressão de voltar em
// silêncio quando alguém mexer no gerador.
func conferirDiscriminacao(t *testing.T, nome string, got []service.SearchHit) {
	t.Helper()

	// `frase-exata` é a exceção declarada: ela casa UMA nota entre 300, e casar
	// uma só é exatamente a propriedade que aquele golden congela.
	if nome == "frase-exata" {
		if len(got) != 1 {
			t.Fatalf("frase-exata casou %d notas, quer 1: a frase deixou de ser "+
				"unica no corpus e o golden nao prova mais casamento de frase", len(got))
		}
		return
	}

	if len(got) < 2 {
		t.Fatalf("%s casou %d resultado(s): um golden de um item nao congela "+
			"ordem nenhuma", nome, len(got))
	}
	if got[0].Score <= got[1].Score {
		t.Fatalf("%s: os dois primeiros empatam (%.6f); o corpus nao discrimina",
			nome, got[0].Score)
	}
	distintos := make(map[float64]bool, len(got))
	for _, r := range got {
		distintos[r.Score] = true
	}
	// Metade, e não todos: notas que caem na mesma fórmula de comprimento e de
	// frequência empatam legitimamente entre si. O que não pode acontecer é a
	// lista inteira ser um bloco de empates.
	if len(distintos)*2 < len(got) {
		t.Fatalf("%s: %d resultados com apenas %d score(s) distinto(s); o golden "+
			"congela o desempate, nao o ranking", nome, len(got), len(distintos))
	}
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
			conferirDiscriminacao(t, c.nome, res.Results)
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
