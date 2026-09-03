### Task 162: Confundidores — corpus do ranking sem empate, contexto de backlink, ordem de mapa, `idx == nil`

**Files:**
- Modify: `internal/service/ranking_golden_test.go:60-110` e `testdata/ranking/*.tsv` (regenerar)
- Modify: `internal/index/backlink_contexto_test.go:66`, `internal/index/backlink_heading_test.go:94`
- Modify: `internal/index/persist_test.go:171`
- Modify: `internal/search/bm25_test.go:127`

**Interfaces:** nenhuma.

- [ ] **Step 1: `ranking_golden_test.go` — corpus com variação, não com um vencedor**

Mecanismo: 299 de 300 notas são idênticas; `n0150` tem uma frase três tokens mais curta e vence os quatro goldens **por ser a mais curta** — inclusive `so-em-heading`, cujo heading é igual nas 300. As linhas 2..N congelam a ordem de inserção. O golden não testa ranking; testa o desempate.

Conserto do gerador (`:60-110`, a função que monta as 300 notas):

- Cada nota `i` recebe: tamanho de corpo `20 + (i*7)%60` tokens de enchimento (varia comprimento); o termo de `termo-amplo` aparece `1 + i%4` vezes; o termo de `dois-termos` só nas notas `i%3 == 0`, segundo termo só em `i%5 == 0`; o termo acentuado só em `i%7 == 0`; o heading de `so-em-heading` só em `i%11 == 0` **e o termo não aparece no corpo dessas**.
- Determinístico (sem `rand`): as fórmulas acima bastam.

Regenerar os quatro `.tsv` com o mecanismo que o teste já tem (`-update` flag ou equivalente — ler o arquivo; se não houver, criar `var atualizar = flag.Bool("update", false, "reescreve os goldens")`).

Verificação de que o novo corpus discrimina: em cada golden, o **primeiro** resultado tem score estritamente maior que o **segundo** (`>`), e os 20 primeiros não são 20 scores iguais. Acrescentar ao teste:

```go
	if len(got) >= 2 && got[0].Score <= got[1].Score {
		t.Fatalf("%s: os dois primeiros empatam (%.6f); o corpus nao discrimina", nome, got[0].Score)
	}
```

Atualizar a docstring `:60-70`: o sintoma diagnosticado ("exatamente 20 linhas") era o desempate de inserção, e a causa era o corpus.

Prova: `mutate.ps1` sobre `internal/search/bm25.go` — Anchor na linha que aplica o peso de heading (`WeightHeading` ou o nome real), Replacement com o peso igual ao de corpo; Test `TestRankingGolden`; Expected exit 0 (o golden `so-em-heading` muda de ordem).

- [ ] **Step 2: `backlink_contexto_test.go:66`, `backlink_heading_test.go:94`**

Mecanismo: uma implementação que devolve só o `Raw` do link como contexto passa, porque o contexto esperado contém o link.

Conserto: afirmar que o contexto contém texto **ao redor** do link que não é o link — escolher no fixture uma palavra que aparece só na linha do link e fora dos colchetes, e afirmar `strings.Contains(ctx, "essa-palavra")`. Nos dois arquivos.

Prova manual: no produto, trocar temporariamente o contexto por `link.Raw`, rodar os dois testes, ver FAIL, restaurar. Colar.

- [ ] **Step 3: `persist_test.go:171` — ordem de mapa**

Mecanismo: `reflect.DeepEqual` sobre slice derivada de iteração de mapa; passa porque há uma origem só.

Conserto: fixture com **duas** origens e comparação por conjunto (ordenar as duas slices com `slices.SortFunc` antes do `DeepEqual`, ou usar `cmp.Diff` com `cmpopts.SortSlices` se `go-cmp` já for dep — conferir `go.mod`; **não** adicionar dep).

- [ ] **Step 4: `bm25_test.go:127` — `idx == nil`**

Mecanismo: com `idx == nil` todo campo é `WeightBody` por definição; o teste de pesos por campo não distingue heading de corpo.

Conserto: montar um índice mínimo (o helper de "vault+Build+Inverted" que o pacote já tem em `persist_test.go:271`, com strip de BOM) com uma nota cujo termo está só no heading e outra só no corpo, e afirmar `scoreHeading > scoreBody`.

Prova: a mesma mutação do Step 1 (peso de heading = peso de corpo) deve fazer este teste falhar também. `mutate.ps1 … -Test TestBM25PesoDeHeading -Package ./internal/search/` — exit 0.

- [ ] **Step 5: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/service/ranking_golden_test.go testdata/ranking/ internal/index/backlink_contexto_test.go internal/index/backlink_heading_test.go internal/index/persist_test.go internal/search/bm25_test.go
git commit -m "test: ranking corpus that discriminates, backlink context beyond the link, map order and nil-index confounders"
```

#### Verificações

- Os quatro `.tsv` regenerados têm primeira linha com score estritamente maior que a segunda (colar `head -2` de cada).
- `mutate.ps1` do Step 1 e do Step 4 com exit 0, saídas coladas.
- Step 2 com o FAIL colado.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Sem dependência nova em `go.mod`; nunca `go mod tidy`.
- Não tocar `internal/search/bm25.go` fora das mutações temporárias; `git diff --stat` do commit só tem `_test.go` e `testdata/`.

#### Comando de mutação

`pwsh -File scripts/mutate.ps1 -Path internal/search/bm25.go -Anchor "<linha do peso de heading>" -Replacement "<mesma linha com peso de corpo>" -Test 'TestRankingGolden|TestBM25PesoDeHeading' -Package ./internal/service/` — e o mesmo com `-Package ./internal/search/`. O implementador cola os dois comandos como rodaram. Exit esperado: 0 nos dois.

#### Contrato de relatório

`task-162-report.md`: status, SHA, `head -2` dos quatro `.tsv`, as duas saídas de `mutate.ps1`, o FAIL do Step 2, `git diff --stat`.

