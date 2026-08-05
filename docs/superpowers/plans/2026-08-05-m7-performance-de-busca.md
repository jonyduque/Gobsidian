# M7 — Performance de busca e leitura em lote (Tasks 78–87)

Base: `236b135`. Alvo: os quatro RNF não atingidos que dependem de código nosso,
mais uma tool nova.

---

## Fase 1 — o que escapou no lote anterior, e por que o instrumento não viu

Escrito antes das tarefas, de propósito. O lote de 2026-08-03/04 (formato do
cache) achou **três defeitos de produção**, e nenhum foi pego por gate:

| Defeito | Quem pegou | Classe |
|---|---|---|
| `DocLength` 5 construído vs 10 recarregado | pessoa, com sonda | **nenhum gate comparava índice construído contra índice carregado** |
| Reconciliação reparava só os metadados | pessoa, com sonda | **teste afirmava sobre UMA estrutura quando a resposta depende de duas** |
| Diretório novo nunca varrido | pessoa, com sonda | **nenhum teste para estado que existia antes do watch existir** |
| `TestCounters_Reconciled…` medindo a máquina | gate, intermitente | teste de fallback com o caminho principal ligado — **armadilha já registrada, reincidente** |
| `create_dirs` e "nenhuma tool cria diretório" | pessoa, conferindo | **afirmação sobre o código sem abrir o código** |

Duas classes foram fechadas no próprio lote (o diferencial construído-vs-cache e
o teste de reconciliação que afirma sobre as duas estruturas). **Duas seguem
abertas**, e as duas mordem exatamente esta batelada:

1. **Não há golden que fixe o ranking.** `bm25_test.go` tem nove testes de
   propriedade — peso de título, saturação, desempate — e nenhum que congele a
   ordem de um corpus real. Quatro das seis otimizações mudam como o score é
   calculado ou cacheado. Um score sutilmente diferente passa por todos os nove.
2. **Não há checagem de "doc cita artefato que não existe".**
   `check_tool_params.ps1` pega o inverso (campo no schema que o código ignora).
   `index_cache.gob` esteve dois anos no PRD como decisão fechada sem nunca ter
   sido implementado, e esta batelada acrescenta documentação de tool nova.

**Por isso as Tasks 78 e 79 vêm antes de qualquer otimização.** Escrever a
otimização primeiro garante que a próxima escape do mesmo jeito.

---

## Decisões fechadas para toda a batelada

Repetidas dentro de cada seção que as vincula. **Não re-litigar.**

**D-M7-1. Otimização que muda resultado é defeito, não trade-off.** As seis
mudanças de performance devem deixar o golden da Task 78 **idêntico**. Um golden
que muda exige explicação escrita e revisão; **nunca regenerar com `-update`
para fazer passar**. `-update` grava o que o código produz, não o que está
certo — já custou caro aqui.

**D-M7-2. Ordem de acumulação de ponto flutuante não muda.** `CalculateBM25`
soma `score += idf * tfScore` num laço. Trocar a ordem de iteração muda o
arredondamento e faz o golden falhar por motivo legítimo — e a reação previsível
é regenerar o golden, que apaga o gate. Onde uma otimização precisar mudar a
ordem, ela **para** e vira decisão de quem revisa. O golden grava score com 6
casas decimais justamente para não ser refém do último bit.

**D-M7-3. Cada tarefa mede sozinha, com `benchstat`, `-count=6`.** Baseline
antes, uma mudança, baseline depois. `~` (sem diferença significativa) **reverte
a mudança**: código mais feio sem ganho é dívida pura. Colar a saída do
`benchstat` no relatório, não o resumo dela.

**D-M7-4. Teto de latência não é afirmado sob `-race`.** O detector custa 2× a
6×. Toda asserção de tempo fica atrás da constante `raceEnabled` (padrão já
existente em `internal/service` e `internal/search`), e a medição vai ao
relatório nos dois modos.

**D-M7-5. RNF-04 continua não atingido até prova em contrário.** O alvo é 100 ms
p95 para `vault_search`; a medição atual é 181 ms com `limit: 200`. Nenhuma
tarefa desta batelada tem licença para afrouxar o teto. Alvo não atingido e
registrado é informação; alvo afrouxado é ficção.

---

## Armadilhas já pagas que valem para a batelada inteira

- **Teste de fallback que deixa o caminho principal ligado mede o caminho
  principal.** Reincidiu duas vezes neste projeto.
- **`-update` de golden grava o que o código produz.** Ler cada arquivo gerado
  contra o que se esperava **antes** de rodar.
- **Chave derivada calculada em dois lugares diverge**, e a divergência aparece
  no caminho menos usado. Toda chave passa por **uma** função.
- **Campo com valor fixo mente sempre.** `alias_collisions` era `0` literal.
- **Prova de mutação escrita no condicional não é prova.** Tempo verbal no
  passado, com a saída colada.
- **Script Python que edita `.go` transforma `\n` de string em quebra real.**
  Use `Edit`, não script, para inserir código com escapes.

---

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

# Task 79 — Checador de artefato citado na doc que não existe no código

**Tier: modelo barato.** Transcrição: a regra está decidida abaixo.

#### Onde encaixa
Segundo. Vale para a doc que a Task 84 vai escrever.

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


#### A evidência medida da lacuna
```
$ grep -rn 'index_cache' --include=*.go .
(vazio)
$ ls "$LOCALAPPDATA/gobsidian/560a6a08c9fa8602/"
inverted_cache.gob
```
`docs/PRD.md` Q3 decidiu persistir `index_cache.gob` **e** `inverted_cache.gob`.
Só o segundo existe. A decisão está escrita como fechada desde 2026-07-29.

#### A decisão que esta tarefa tem de acertar
O checador procura, em `docs/*.md` e `README.md`, tokens em crase que **parecem
identificador de código** e não aparecem em nenhum `.go`:

- nome de arquivo `*.gob`, `*.go`
- token `snake_case` dentro de bloco de parâmetro/JSON
- identificador `CamelCase` com parêntese, ex.: `` `MoveNote()` ``

**Saída é lista, não veredito.** Cada achado é uma frase que precisa de uma
pessoa confirmando — igual a `audit_reports.ps1`. Sai `1` com achados.

**Ruído mata checador.** Rodar contra o repositório inteiro e olhar o volume
antes de aceitar a regra: a primeira versão do checador de briefs sinalizou uma
linha que **negava** ter placeholder. Se passar de ~20 achados legítimos-mas-
irrelevantes, restringir o padrão, não aceitar o barulho.

#### Verificações além dos passos
Prova de que o instrumento pega: acrescente `` `create_dirs` `` a
`docs/TOOLS.md`, rode, confirme que aparece, **remova**. Colar as duas saídas.

#### Contrato de relatório
Volume total de achados no repositório hoje, e a lista. Prova de disparo acima.
Esta tarefa **não tem prova de mutação** — o entregável é um script PowerShell,
e `mutate.ps1` roda teste Go. A prova equivalente é o disparo controlado acima.

**Files:** `scripts/check_doc_refs.ps1`
**Commit:** `test(docs): flag doc references to code artifacts that do not exist`

---

# Task 80 — `Normalize` sem reconstruir o pipeline a cada chamada

**Tier: modelo barato.** O corpo do teste difícil está escrito abaixo.

#### Onde encaixa
Primeira otimização. Beneficia busca **e** indexação.

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


#### A evidência medida do defeito
```
$ go tool pprof -sample_index=alloc_space -top internal/service s_mem.out
      flat  flat%   sum%        cum   cum%
 2545.70MB 80.45% 80.45%  2545.70MB 80.45%  golang.org/x/text/transform.Chain (inline)
```
80,45% de toda a alocação. `analyzer.go:26` monta a chain dentro da função.

#### A decisão que esta tarefa tem de acertar
**Duas mudanças, medidas separadamente:**

1. `sync.Pool` de transformers, com `Reset()` antes de cada uso. O comentário
   atual — *"Cria um novo transformer a cada chamada para garantir
   thread-safety total"* — está **certo sobre o perigo**: `transform.Transformer`
   tem estado e não é seguro concorrentemente. O pool preserva a garantia
   (uma instância por goroutine em voo) e elimina a construção.
2. Atalho ASCII: string sem byte `>= 0x80` não tem o que decompor. `NFD`/`NFC`
   viram no-op e só `ToLower` importa.

Fazer 1 e depois 2, com `benchstat` entre elas. Se a 2 der `~`, **reverter a 2**
(D-M7-3): ela adiciona um ramo ao caminho mais quente do sistema.

#### Armadilhas específicas desta tarefa
Estado vazando entre usos do pool é a falha silenciosa aqui: produz string
**errada**, não lenta, e só sob concorrência.

#### O corpo do teste que não é óbvio
```go
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

	const goroutines = 16
	const voltas = 500
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
```

#### Verificações além dos passos
- Rodar o teste acima **com `-race`**: é o único modo em que o vazamento aparece
  de forma confiável.
- Golden da Task 78 idêntico (D-M7-1).

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path internal/search/analyzer.go `
  -Anchor 't.Reset()' -Replacement '_ = t' `
  -Test TestNormalizeNaoVazaEstadoEntreUsos -Package ./internal/search/
```
Exit `0` esperado. Se der `1`, o teste não consegue reprovar sem o `Reset` e
**não** cobre a regra.

#### Contrato de relatório
`benchstat` de `BenchmarkSearchLimit200` e `BenchmarkIndexBuild`, antes e depois,
`-count=6`, para cada uma das duas mudanças. Saída do `mutate.ps1` colada.

**Files:** `internal/search/analyzer.go`, `internal/search/analyzer_test.go`
**Commit:** `perf(search): reuse the normalization pipeline instead of rebuilding it`

---

# Task 81 — Título normalizado calculado na indexação, não por posição

**Tier: modelo barato.**

#### Onde encaixa
Depois da Task 80. Mede-se contra a baseline que ela deixou.

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

#### A evidência medida do defeito
`bm25.go:170`, dentro do laço que roda **para cada posição de cada posting**:
```go
normTitle := Normalize(n.Title)
```
`CalculateBM25` cum = 1094,74 MB no perfil, ~137 MB por busca. O título não muda
entre posições da mesma nota.

#### A decisão que esta tarefa tem de acertar
`Note` ganha `TitleNorm string`, preenchido onde `Title` é preenchido.
`getFieldWeight` lê o campo.

**Uma função só produz a chave** (armadilha do `aliasKey`): `TitleNorm` é
escrito num lugar — o construtor de `Note` — e **nunca** recalculado no caminho
de leitura. `MoveNote` copia a struct, então o campo viaja junto; conferir isso
explicitamente, porque `MoveNote` já entrou fora do contrato uma vez.

#### O corpo do teste que não é óbvio
```go
// TestTitleNormAcompanhaOTitulo guarda a divergencia que um campo derivado
// SEMPRE convida: Title muda num caminho e TitleNorm nao muda no outro.
//
// Cobre os tres caminhos que publicam Note: Build, Replace e MoveNote.
func TestTitleNormAcompanhaOTitulo(t *testing.T) {
	root := t.TempDir()
	escreve(t, root, "a.md", "# Prescrição Intercorrente\n\ncorpo\n")
	v, idx := cofreIndexado(t, root)

	confere := func(quando string, p vault.CanonicalPath) {
		t.Helper()
		n, ok := idx.Get(p)
		if !ok {
			t.Fatalf("%s: nota %q sumiu do indice", quando, p)
		}
		if quer := search.Normalize(n.Title); n.TitleNorm != quer {
			t.Errorf("%s: TitleNorm=%q, quer %q (Title=%q)",
				quando, n.TitleNorm, quer, n.Title)
		}
	}
	confere("apos Build", "a.md")

	escreve(t, root, "a.md", "# Execução Fiscal\n\ncorpo\n")
	if err := idx.Replace(context.Background(), v, "a.md"); err != nil {
		t.Fatal(err)
	}
	confere("apos Replace", "a.md")

	if err := os.Rename(filepath.Join(root, "a.md"), filepath.Join(root, "b.md")); err != nil {
		t.Fatal(err)
	}
	idx.MoveNote(v, "a.md", "b.md")
	confere("apos MoveNote", "b.md")
}
```

#### Verificações além dos passos
Golden da Task 78 idêntico. `Normalize` de um título acentuado tem de dar o
mesmo antes e depois — é o caso que o golden `com-acento` cobre.

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path internal/index/note.go `
  -Anchor 'TitleNorm: search.Normalize(title)' -Replacement 'TitleNorm: title' `
  -Test TestTitleNormAcompanhaOTitulo -Package ./internal/index/
```

#### Contrato de relatório
`benchstat` de `BenchmarkSearchLimit200` e `BenchmarkSearchTermoAmplo`.
Perfil `alloc_space` depois, mostrando `transform.Chain` fora do topo.

**Files:** `internal/index/note.go`, `internal/index/build.go`,
`internal/index/update.go`, `internal/search/bm25.go`, testes
**Commit:** `perf(search): precompute the normalized title at index time`

---

# Task 82 — `avgdl` em cache, invalidado por geração

**Tier: modelo barato.**

#### Onde encaixa
Depois da 81. Toca `bm25.go`, como a 81 — sequencial, nunca em paralelo.

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

#### A evidência medida do defeito
`bm25.go:83`, por busca:
```go
for _, p := range idx.Paths() {
    if dl := ix.DocLength(string(p)); dl > 0 { sumDocLen += float64(dl) }
}
```
`idx.Paths()` aloca fatia de N elementos **e ordena** (`index.go`, `slices.Sort`).
Depois são N chamadas a `DocLength`, **cada uma pegando RLock**. Tudo para uma
constante do corpus.

#### A decisão que esta tarefa tem de acertar
O cache é chaveado por `idx.Generation()`, que **já existe** e sobe a cada
mutação. Guardar `(geracao, avgdl)`; recalcular quando a geração diferir.

**O cache fica no `Inverted`, não em variável de pacote.** Estado global entre
cofres é como dois cofres passam a compartilhar `avgdl` — e o servidor pode
servir mais de um índice no mesmo processo em teste.

**`avgdl` obsoleto muda score sem erro.** É o modo de falha, e o golden não o
pega se a geração não mudar no teste: o teste tem de **forçar** a mudança.

#### O corpo do teste que não é óbvio
```go
// TestAvgdlInvalidaComAGeracao guarda o defeito que o cache INTRODUZ.
//
// avgdl obsoleto nao levanta erro: muda o score em silencio. O teste adiciona
// uma nota MUITO longa depois da primeira busca — se o cache nao invalidar, o
// avgdl continua o da corpus pequeno e a normalizacao por tamanho fica errada.
func TestAvgdlInvalidaComAGeracao(t *testing.T) {
	root := t.TempDir()
	escreve(t, root, "a.md", "# A\n\nprescricao\n")
	escreve(t, root, "b.md", "# B\n\nprescricao prescricao\n")
	svc, v, idx, inv := servicoCompleto(t, root)

	antes, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao"})
	if err != nil {
		t.Fatal(err)
	}

	// Nota longa: muda avgdl de forma detectavel.
	longa := "# C\n\n" + strings.Repeat("palavra ", 5000) + "prescricao\n"
	escreve(t, root, "c.md", longa)
	if err := idx.Replace(context.Background(), v, "c.md"); err != nil {
		t.Fatal(err)
	}
	if err := inv.Update(context.Background(), v, "c.md"); err != nil {
		t.Fatal(err)
	}

	depois, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao"})
	if err != nil {
		t.Fatal(err)
	}
	if len(antes.Results) == 0 || len(depois.Results) == 0 {
		t.Fatal("busca vazia nos dois lados nao prova nada")
	}
	if antes.Results[0].Score == depois.Results[0].Score {
		t.Errorf("score de %q identico (%.6f) depois de o corpus mudar de "+
			"tamanho: avgdl nao foi invalidado",
			antes.Results[0].Path, antes.Results[0].Score)
	}
}
```

#### Verificações além dos passos
- Golden da Task 78 idêntico.
- O teste acima rodado **duas vezes seguidas** no mesmo processo: cache que
  guarda a geração errada acerta na primeira e erra na segunda.
- `go test -race ./internal/search/`: o cache é lido e escrito sob o mesmo
  `RWMutex` do índice, e o detector é quem diz se ficou.

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path internal/search/bm25.go `
  -Anchor 'if ix.avgdlGen == ger {' -Replacement 'if true {' `
  -Test TestAvgdlInvalidaComAGeracao -Package ./internal/search/
```

#### Contrato de relatório
`benchstat` das quatro `BenchmarkSearch*`. Dizer explicitamente quantas
aquisições de RLock por busca foram eliminadas (é N, com N = notas do cofre).

**Files:** `internal/search/bm25.go`, `internal/search/inverted.go`, testes
**Commit:** `perf(search): cache avgdl and invalidate it by index generation`

---

# Task 83 — Buscar as postings de cada termo uma vez

**Tier: modelo barato.**

#### Onde encaixa
Depois da 82. Última das quatro que tocam `bm25.go`/`analyzer.go`.

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

#### A evidência medida do defeito
```
371.40MB 11.74%  (*Inverted).Postings
```
`CalculateBM25` chama `ix.Postings(m.term)` no laço de pontuação e depois
`ix.Postings(qTok.Raw)` / `(qTok.Reduced)` de novo para montar `docsWithTerm`
do IDF. Mesmo termo, mesma fatia, duas alocações.

#### A decisão que esta tarefa tem de acertar
Montar `docsWithTerm` **na primeira passada**, guardando as postings já obtidas.

**D-M7-2 vale aqui com força.** A tentação é reordenar o laço para aproveitar a
estrutura. Se a ordem de acumulação de `score` mudar, o golden falha por motivo
legítimo — e a reação previsível é regenerar. **Não reordene.** Se parecer
necessário, pare e escreva por quê.

#### Verificações além dos passos
Golden idêntico. Se não for, **não regenerar**: a explicação vira parte do
relatório e a mudança volta para revisão.

#### Prova de mutação
Esta tarefa **não tem prova de mutação**: remove trabalho duplicado sem criar
invariante nova. O que prova que nada quebrou é o golden inalterado.

#### Contrato de relatório
Três coisas, e a primeira é a que o auditor não consegue inferir:

1. **A frase "sem prova de mutação, e por quê"**, explícita. Sem ela o auditor
   não distingue "não se aplica" de "esqueci", e `audit_reports.ps1` sinaliza
   tarefa sem seção de mutação.
2. `TestRankingGolden` verde, com a saída colada, provando que os seis `.tsv`
   ficaram idênticos.
3. `benchstat` de `BenchmarkSearchLimit200` e `BenchmarkSearchTermoAmplo`,
   `-count=6`, antes e depois. Se der `~`, a mudança é revertida e o relatório
   diz isso.

**Files:** `internal/search/bm25.go`
**Commit:** `perf(search): fetch each term's postings once per query`

---

# Task 84 — `note_read` aceitando vários caminhos

**Tier: modelo forte.** Muda contrato público de tool: schema, doc e erro parcial. O modo de falha barato é decidir sozinho o que acontece quando **um** dos caminhos falha.

#### Onde encaixa
Depois das otimizações de busca. Não conflita em arquivo nenhum com elas.

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
Depois das otimizações de busca; não conflita com elas em arquivo nenhum.

#### A evidência da lacuna
`note_read` aceita `path` (um). `note_read` p95 medido em **345 µs**. Um fluxo
de pesquisa que lê dez notas paga dez idas e voltas de protocolo por 3,5 ms de
trabalho.

#### A decisão que esta tarefa tem de acertar
**Pré-decidido, não re-litigar:**

1. O campo novo é `paths: []string`, **e `path` continua existindo**. Os dois
   preenchidos é erro de validação (`INVALID_ARGUMENT`), não precedência
   silenciosa.
2. **Falha parcial não derruba o lote.** Cada item do retorno carrega o próprio
   `error` opcional. Uma nota inexistente no meio de dez não pode custar as nove
   boas — mas **também não pode sumir**: o item aparece com o erro, na mesma
   posição. Uma lista que encolhe sem dizer é a falha silenciosa desta tarefa.
3. `max_bytes` aplica-se **por nota**, não ao lote. Um teto de lote faria a
   décima nota truncar por causa das nove anteriores, o que depende da ordem.
4. Limite de `len(paths)`: **50**. Acima disso, erro. Sem teto, uma chamada pede
   o cofre inteiro e o servidor materializa 100 MB em memória para uma resposta
   que o cliente não consegue ler.

#### Armadilhas específicas desta tarefa
- **Schema que promete e código que ignora é pior que parâmetro ausente.**
  `note_list.fields` já foi declarado e descartado. `scripts/check_tool_params.ps1`
  roda no `verify.ps1` e vai reprovar se `paths` não for lido.
- **Handler que devolve `error` Go faz o SDK montar `IsError` sem
  `StructuredContent`.** Erro de validação (os dois campos, ou lote grande
  demais) sai como resultado de erro **com** `Out` preenchido.

#### Verificações além dos passos
- Teste com dez caminhos, um deles inexistente: os nove voltam, o décimo volta
  **na posição certa** com erro.
- Teste com `path` e `paths` juntos: erro de validação.
- Teste com 51 caminhos: erro.
- `docs/TOOLS.md` atualizado, e `scripts/check_doc_refs.ps1` (Task 79) limpo.

#### Prova de mutação
Duas regras, duas provas — uma por regra, não uma por tarefa:
```
pwsh -File scripts/mutate.ps1 -Path internal/mcpsrv/tools_read.go `
  -Anchor 'if len(req.Paths) > maxPathsPorLote {' -Replacement 'if false {' `
  -Test TestNoteReadRecusaLoteAcimaDoTeto -Package ./internal/mcpsrv/

pwsh -File scripts/mutate.ps1 -Path internal/service/read.go `
  -Anchor 'out[i] = ReadNoteItem{Path: p, Err: err}' `
  -Replacement 'continue' `
  -Test TestNoteReadMantemPosicaoNoErroParcial -Package ./internal/service/
```
A segunda é a que importa: substituir o item de erro por `continue` faz a lista
encolher em silêncio, que é exatamente o defeito.

#### Contrato de relatório
Saída de `check_tool_params.ps1` e de `check_doc_refs.ps1`, mais as duas provas
de mutação acima com a saída colada.

**Files:** `internal/mcpsrv/tools_read.go`, `internal/service/read.go`,
`docs/TOOLS.md`, testes
**Commit:** `feat(note_read): read several notes in one call`

---

# Task 85 — Cache do índice de metadados

**Tier: modelo forte.** É o maior ganho absoluto e o maior risco: um cache de metadados errado serve nota errada, não nota lenta.

#### Onde encaixa
Independente das otimizações de busca. Maior ganho absoluto do lote.

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

#### A evidência medida do defeito
```
msg="servidor pronto" notes=3149 index_ms=905
```
905 ms, toda partida, varrendo e parseando o cofre inteiro. **RNF-02 não
atingido**: 832–1183 ms contra teto de 300 ms.


#### A decisão que esta tarefa tem de acertar
**Reaproveitar o codec da Task do formato 5**, não inventar outro. O
`persist_codec.go` já resolve tabela de strings, varint com delta, totais
adiantados e portão de versão. Um segundo formato binário no mesmo projeto é
duas cópias da mesma regra, e a que diverge é a menos usada.

**O cabeçalho confere cobertura, não só versão.** A lição já paga: um cache
parcial passou por completo porque `LoadInvertedCache` conferia versão e não
contagem.

**Invalidação é por mtime e tamanho por arquivo**, com o mesmo raciocínio já
escrito em `ARCHITECTURE.md` §.

#### Armadilhas específicas desta tarefa
- **Nota vazia nunca contava como coberta** e fazia todo boot achar o cache
  parcial. O equivalente aqui é qualquer nota que não gere entrada.
- **`DocLength` divergia entre construído e recarregado.** O mesmo teste
  diferencial vale: índice construído do zero e índice carregado do cache têm de
  responder **igual** em `Get`, `Backlinks`, `Tags`, `ResolvePath` e `Paths`.

#### O corpo do teste que não é óbvio
Diferencial, no molde do que pegou o defeito do `DocLength`:
```go
// TestIndiceDeMetadadosRecarregadoEIdentico compara os dois caminhos de
// construcao campo a campo, em vez de conferir valores escritos a mao.
// Valor escrito a mao codifica o mesmo engano do codigo; o caminho de
// construcao do zero e o que ja estava certo.
func TestIndiceDeMetadadosRecarregadoEIdentico(t *testing.T) {
	// ... corpus com: nota com alias, nota com backlink, nota com anchor
	// quebrada, nota VAZIA, anexo, e nome que colide em caixa.
	// Construir do zero -> salvar -> carregar -> comparar:
	//   Paths(), NoteCount(), AssetCount(), TotalSize(), Tags(""),
	//   e por caminho: Get(), Backlinks(), ResolvePath() do nome curto.
}
```

#### Verificações além dos passos
- O diferencial acima cobre nota vazia, alias, âncora quebrada, anexo e colisão
  de caixa. Faltando qualquer um deles, o teste passa e a cobertura não existe.
- Boot real com o cache presente: o log tem de dizer que ele foi usado, e
  `notes=` tem de bater com a varredura.
- Apagar o cache e reiniciar: reconstrói sem erro.
- Corromper um byte no meio do arquivo: recusa como corrompido, não decodifica.

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path internal/index/persist.go `
  -Anchor 'if h.NoteCount != idx.NoteCount() {' -Replacement 'if false {' `
  -Test TestCacheDeMetadadosParcialERecusado -Package ./internal/index/
```
É a regra que o cache de busca aprendeu na marra: conferir versão e não
cobertura deixou um cache parcial passar por completo.

#### Contrato de relatório
`index_ms` medido em cinco partidas, antes e depois, no cofre real.
Dizer se RNF-02 passou a ser atingido — e **se não passou, dizer o número**.
Prova de mutação da checagem de cobertura do cabeçalho.

**Files:** `internal/index/persist.go` (novo), `cmd/gobsidian/serve.go`,
`docs/PRD.md` (fechar a anotação), `docs/OPERACAO.md`, testes
**Commit:** `perf(index): persist the metadata index and load it at boot`

---

# Task 86 — Re-resolução de links dirigida, não global

**Tier: modelo forte.** Maior risco de correção da batelada. Mexe em resolução de link, que já custou caro aqui.

#### Onde encaixa
Última das de código: maior risco, e quer golden e benchmarks já estáveis
para conseguir atribuir regressão.

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

#### A evidência medida do defeito
`internal/index/update.go:299`, com o comentário do próprio código:
> `reprocessLinksLocked` *"roda em **todo** evento do watcher, sobre **todas**
> as notas"*

**RNF-06 não atingido**: 20,35 ms contra 20 ms para reindexar um arquivo.


#### A decisão que esta tarefa tem de acertar
O índice reverso cobre links **não resolvidos** também. Um link `[[foo]]`
quebrado tem de estar no mapa sob `foo`, senão criar `foo.md` não conserta nada
— e o sintoma é um link que fica quebrado para sempre até reiniciar.

Aliases contam: criar uma nota com `aliases: [STJ]` afeta todo `[[STJ]]`.

#### Verificações além dos passos
Diferencial contra o caminho global: para uma sequência de eventos (criar,
renomear, apagar, criar de novo com alias), o índice resultante da re-resolução
dirigida tem de ser **idêntico** ao da global. O caminho global fica no teste
como referência, não no produto.

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path internal/index/resolve.go `
  -Anchor 'for _, alias := range n.Aliases {' -Replacement 'for _, alias := range []string(nil) {' `
  -Test TestReresolucaoDirigidaCobreAliases -Package ./internal/index/

pwsh -File scripts/mutate.ps1 -Path internal/index/update.go `
  -Anchor 'ix.citantesPorNome[nomeChave(alvo)]' -Replacement 'ix.citantesPorNome[alvo]' `
  -Test TestReresolucaoDirigidaIgualAGlobal -Package ./internal/index/
```
A segunda muta exatamente a armadilha do `aliasKey`: chave crua num lado,
normalizada no outro. Se o teste não reprovar, ele não cobre a divergência.

**As âncoras acima nomeiam código que ainda não existe, e isso é deliberado:
elas são o contrato de nomes desta tarefa.** O mapa reverso chama-se
`citantesPorNome`, e a função única que deriva a chave chama-se `nomeChave`. Se
a implementação usar outros nomes, as duas provas não casam âncora e o
`mutate.ps1` sai `2` (inconclusivo) — o que se lê como "não provado", e é.

#### Contrato de relatório
`benchstat` de reindexação de arquivo único no cofre de 5.000 notas.
Dizer se RNF-06 passou a ser atingido, com o número.

**Files:** `internal/index/update.go`, `internal/index/resolve.go`, testes
**Commit:** `perf(index): re-resolve only the links a change can affect`

---

# Task 87 — Relatórios, ledger e medição de fechamento

**Tier: modelo forte.** O entregável são relatórios com evidência real, e o modo de falha de um modelo barato pedido a "escrever relatório com evidência" é
fabricá-la.

#### Onde encaixa
Fechamento. Não envia código.

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

#### O que entregar
- Tabela dos quatro RNF (02, 04, 06, 07) com o número **antes** e **depois**,
  medidos no cofre real, e a palavra "não atingido" onde for o caso.
- `docs/OPERACAO.md` atualizado; `README.md` com a contagem certa de requisitos
  não atingidos (já errou uma vez: dizia "três" com quatro na tabela).
- Ledger em `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`.
- `pwsh -File scripts/audit_reports.ps1` sem achados nas seções novas.
- **Todo SHA citado conferido com `git cat-file -t`.** A Task 31 foi registrada
  em `14210ee`, que não existe.

#### Verificações além dos passos
- `git cat-file -t <sha>` para **cada** SHA citado, com a saída colada. A Task 31
  foi registrada em `14210ee`, que não existe no repositório.
- `audit_reports.ps1` rodado, e os achados da seção nova em zero — achados
  antigos em relatórios de outros marcos não contam e devem ser distinguidos.
- Validação UTF-8 de todo `.md` tocado.

#### Contrato de relatório
Esta tarefa **não tem prova de mutação**: não envia código.

**Files:** `docs/OPERACAO.md`, `README.md`, ledger
**Commit:** `docs(ledger): record M7 and the measured state of the four RNFs`

---

## Ordem de execução, e por quê

```
78 → 79 → 80 → 81 → 82 → 83 → 84 → 85 → 86 → 87
```

- **78 antes de tudo**: as seis otimizações são medidas contra o golden.
- **80 antes de 81**: o pool beneficia indexação também; medir na ordem inversa
  esconde o ganho do pool atrás do do título.
- **80, 81, 82, 83 tocam `bm25.go`/`analyzer.go`** — sequenciais, nunca em
  paralelo.
- **84 e 85 não conflitam** com nada acima nem entre si.
- **86 por último entre as de código**: maior risco, e quer o golden e os
  benchmarks já estáveis para atribuir regressão.

---

## Prompt de despacho

> **Lote M7 — performance de busca e leitura em lote. Tasks 78 a 87.**
>
> **O que torna este lote diferente:** seis das dez tarefas mudam como o score
> de busca é calculado. **Otimização que muda resultado é defeito, não
> trade-off.** A Task 78 congela o ranking num golden; toda tarefa seguinte tem
> de deixá-lo idêntico. **Nunca regenerar o golden com `-update` para fazer
> passar** — golden que muda exige explicação escrita e volta para revisão.
>
> **Estado inicial:** base `236b135`, árvore limpa, ledger em
> `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`.
>
> **Ordem:** 78 → 79 → 80 → 81 → 82 → 83 → 84 → 85 → 86 → 87. As tarefas 80 a 83
> tocam os mesmos arquivos; **não paralelizar**.
>
> **Laço por tarefa:**
> ```
> pwsh -File scripts/sdd.ps1 base <N>      # ANTES de comecar
> pwsh -File scripts/sdd.ps1 brief <N>
> # executar
> pwsh -File scripts/verify.ps1
> pwsh -File scripts/sdd.ps1 review <N>
> # ledger ANTES de dizer que acabou
> ```
>
> **Aceitação por tarefa:**
> - **78** — os seis `.tsv` colados, e uma frase por consulta dizendo por que
>   aquela ordem está certa. Falha barata: golden gerado sobre corpus que não
>   tem o tamanho que o nome diz. O corpus afirma o próprio tamanho; conferir
>   que a asserção existe.
> - **79** — prova de disparo (inserir `create_dirs` na doc, ver o achado,
>   remover) **e** o volume total no repositório. Falha barata: checador que
>   dispara em prosa legítima vira ruído e para de ser lido.
> - **80** — `benchstat` das DUAS mudanças separadamente. Falha barata: pool sem
>   `Reset()`, que produz string errada só sob concorrência. Exigir o teste
>   rodado com `-race`.
> - **81** — falha barata: `TitleNorm` preenchido em `Build` e esquecido em
>   `Replace` ou `MoveNote`. O teste cobre os três; conferir que cobre.
> - **82** — falha barata: cache em variável de pacote, compartilhado entre
>   cofres. Exigir que esteja no `Inverted`.
> - **83** — **sem prova de mutação**, e o relatório tem de dizer isso. Falha
>   barata: reordenar o laço "para aproveitar a estrutura", quebrar o golden por
>   arredondamento e regenerá-lo.
> - **84** — falha barata: caminho que falha some da lista em vez de aparecer na
>   posição com erro. Exigir o teste dos dez com um inexistente.
> - **85** — falha barata: cache que confere versão e não cobertura, que já
>   aconteceu no cache de busca. Exigir o diferencial construído-vs-recarregado.
> - **86** — falha barata: chave do mapa reverso calculada em dois lugares.
>   Exigir que passe por uma função só, e o diferencial contra o caminho global.
> - **87** — sem prova de mutação; **dizer isso**. Todo SHA conferido com
>   `git cat-file -t`.
>
> **Decisões que não se re-litigam:** D-M7-1 a D-M7-5, na seção de decisões
> fechadas. Em especial: `~` no `benchstat` **reverte a mudança**, e nenhum teto
> de RNF é afrouxado nesta batelada.
>
> **Regras para quem orquestra:** revisor também erra — conferir a afirmação
> contra o código antes de aceitar o achado. Escopo não encolhe em silêncio:
> `BLOCKED` com motivo é resposta melhor que entrega que parece completa. Todo
> SHA que for para o ledger passa por `git cat-file -t`.
>
> **Gate final:**
> ```
> pwsh -File scripts/verify.ps1
> pwsh -File scripts/test_orphans.ps1 -Cycles 100
> pwsh -File scripts/audit_reports.ps1
> pwsh -File scripts/check_doc_refs.ps1
> ```
>
> **O que volta para quem pediu:** mudar o schema de `note_read` é contrato
> público — a Task 84 altera uma tool já publicada. Fechar a anotação do Q3 no
> PRD é decisão de projeto.

### Custo e tiers

| Tier | Tasks | Por quê |
|---|---|---|
| Barato | 79, 80, 81, 82, 83 | corpo dos testes difíceis está escrito; é transcrição |
| Forte | 78, 84, 85, 86, 87 | teste que ainda não existe, contrato público, ou relatório com evidência |

Estimativa: 10 tarefas, ~2 invocações cada com revisão, ~20 no total.

**A que eu não delegaria sem ler a saída inteira: a 86.** Ela mexe em resolução
de link, o defeito que ela pode introduzir resolve para a nota errada com
`state=ok`, e este projeto já pagou exatamente esse.
