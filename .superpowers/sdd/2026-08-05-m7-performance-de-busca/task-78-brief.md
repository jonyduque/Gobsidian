# Task 78 — Golden de ranking de busca

**Tier: modelo forte.** O entregável é um teste que não pode ser enganado, e o modo de falha barato é um golden que passa com o corpus errado.

#### Onde encaixa
Primeiro. As seis otimizações seguintes são medidas contra ele.

#### O que vincula esta tarefa

Repetido aqui de propósito: o brief é a unidade que viaja, e decisão citada por
código fica no preâmbulo, que não viaja com ela.

- **Otimização que muda resultado é defeito, não trade-off.** O golden de
  ranking da Task 78 (`testdata/ranking/*.tsv`, teste `TestRankingGolden` em
  `internal/service/`) tem de ficar **idêntico**. Golden que muda exige
  explicação escrita e volta para revisão. **Nunca regenerar com `-update` para
  fazer passar** — `-update` grava o que o código produz, não o que está certo.
- **Ordem de acumulação de ponto flutuante não muda.** `CalculateBM25` soma
  `score += idf * tfScore` num laço. Reordenar a iteração muda o arredondamento
  e faz o golden falhar por motivo legítimo; a reação previsível é regenerar, o
  que apaga o gate. Se parecer necessário reordenar, **pare** e escreva por quê.
- **`benchstat` com `-count=6`, uma mudança por vez.** Baseline antes, mudança,
  baseline depois. `~` (sem diferença significativa) **reverte a mudança**:
  código mais feio sem ganho é dívida pura. Colar a saída, não o resumo dela.
- **Teto de latência não é afirmado sob `-race`** (custa 2× a 6×). Asserção de
  tempo fica atrás da constante `raceEnabled`, padrão já existente em
  `internal/service` e `internal/search`.
- **Nenhum teto de RNF é afrouxado nesta batelada.** RNF-04 está em 181 ms
  contra alvo de 100 ms. Alvo não atingido e registrado é informação; alvo
  afrouxado é ficção.

#### Armadilhas já pagas que se aplicam
- **Teste de fallback que deixa o caminho principal ligado mede o caminho
  principal.** Reincidiu duas vezes neste projeto.
- **Chave derivada calculada em dois lugares diverge**, e a divergência aparece
  no caminho menos usado — `[[STJ]]` continuou resolvendo, com `state=ok`, para
  uma nota já removida. Toda chave passa por **uma** função.
- **Campo com valor fixo mente sempre.** `alias_collisions` era `0` literal.
- **Prova de mutação escrita no condicional não é prova.** Tempo verbal no
  passado, com a saída colada.
- **Script Python que edita `.go` converte a sequencia de escape de quebra
  de linha numa quebra literal**, e corrompe a string Go.
  Use `Edit`, não script, para inserir código com escapes.

#### Regras de execução
Rodar `pwsh -File scripts/verify.ps1` antes de dizer que acabou. Registrar no
ledger (`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`) **antes** de
reportar conclusão. Escopo não encolhe em silêncio: se alguma parte não deu,
entregue o resto inteiro e diga o que ficou de fora e por quê — `BLOCKED` com
motivo é resposta melhor que entrega que parece completa.

#### Onde encaixa
Primeiro. As seis otimizações seguintes são medidas contra ele.


#### A evidência medida da lacuna
```
$ grep -rn 'func Test' internal/search/bm25_test.go
  TestBM25FieldWeightsAreApplied  TestBM25WeightTitle  TestBM25WeightHeadings
  TestBM25WeightBody  TestBM25TermFrequencySaturation
  TestBM25DocumentLengthNormalization  TestBM25RawVsReduced
  TestBM25FrequentTermNoNaN  TestBM25DeterministicTieBreaking
$ ls testdata/
parity  parser  vault_small
```
Nove testes de propriedade, zero golden de ranking.

#### A decisão que esta tarefa tem de acertar
O golden grava **caminho e score com 6 casas**, nesta ordem, uma linha por
resultado, para um conjunto fixo de consultas. Seis casas e não igualdade exata
de float: o objetivo é pegar mudança de ranking, não o último bit de
arredondamento (D-M7-2).

As consultas têm de cobrir os caminhos que as otimizações tocam:
termo amplo (milhares de postings), dois termos, frase exata, termo **com
acento** (é o que passa por `Normalize`), termo que só existe no **título**, e
termo que só existe num **heading**.

#### O corpo do teste
```go
// corpusGolden monta um cofre determinístico e AFIRMA o próprio tamanho.
//
// A afirmação não é decoração: um corpus que gera menos notas do que o nome diz
// produz um golden menor, que passa, e some com a cobertura sem nada indicar.
func corpusGolden(t *testing.T) (*service.Service, string) {
	t.Helper()
	const querNotas = 300
	root := t.TempDir()
	for i := 0; i < querNotas; i++ {
		// Acento no título de propósito: é o caminho de Normalize.
		corpo := fmt.Sprintf(
			"---\ntags: [t%d]\n---\n\n# Prescrição intercorrente %04d\n\n"+
				"## Execução fiscal\n\nnota %04d sobre prescricao e execucao. "+
				"O algoritmo BM25 com pesos aparece aqui quando %d.\n",
			i%7, i, i, i%13)
		dir := filepath.Join(root, fmt.Sprintf("pasta%02d", i%10))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("n%04d.md", i)),
			[]byte(corpo), 0644); err != nil {
			t.Fatal(err)
		}
	}
	svc := servicoDoCofre(t, root) // helper já existente nos testes de service
	stats, err := svc.VaultStats(context.Background())
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
	{"termo-amplo", service.SearchOptions{Query: "nota", Limit: intPtr(50)}},
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
			quer, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("golden ausente (%v). Gere com -update e LEIA o "+
					"arquivo antes de commitar", err)
			}
			if b.String() != string(quer) {
				t.Errorf("ranking mudou.\n--- quer ---\n%s\n--- tem ---\n%s\n"+
					"Golden que muda exige explicacao escrita. NAO regenere "+
					"para fazer passar.", quer, b.String())
			}
		})
	}
}
```

#### Verificações além dos passos
- Gerar com `-update`, **abrir cada `.tsv`** e conferir: a nota com o termo no
  título vem antes das que só o têm no corpo? A frase exata casa uma só?
- `len(res.Results) == 0` reprova. Consulta que não casa nada mede o caminho
  vazio — este projeto já registrou uma medição de RNF-04 de 0,58 ms por isso.

#### Regras de execução
Não alterar `bm25.go` nesta tarefa. Só teste e `testdata/ranking/*.tsv`.

#### Contrato de relatório
Colar os seis `.tsv` gerados, e uma frase por consulta dizendo **por que aquela
ordem está certa**. Esta tarefa **não tem prova de mutação**: ela não muda código de produção. A
frase precisa estar no relatório — o auditor não distingue "não se aplica" de
"esqueci".

**Files:** `internal/service/ranking_golden_test.go`, `testdata/ranking/*.tsv`
**Commit:** `test(search): freeze the ranking of a real corpus as a golden`

---

