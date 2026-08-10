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

