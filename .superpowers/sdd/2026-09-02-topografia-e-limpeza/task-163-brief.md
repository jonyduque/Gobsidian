### Task 163: RNF-5000 afirma ou não existe; paridade itera `got` e falha em vez de pular; orçamentos onde só havia `Logf`

**Files:**
- Modify: `internal/service/rnf5000_test.go:85-160`
- Modify: `internal/index/parity_test.go:120-210`
- Modify: `internal/writer/diff_test.go:71`, `internal/search/persist_test.go:281`, `internal/index/movenote_semvault_test.go:77`
- Modify: `scripts/gen_vault.ps1` (só se o Step 1 exigir termos no corpus — ver)

**Interfaces:**
- Consumes: os tetos de RNF-01 e RNF-07 em `docs/PRD.md` (ler os números lá; **não** inventar).

- [ ] **Step 1: `rnf5000_test.go` — cada consulta tem guarda e teto**

Mecanismo: 3 das 8 consultas (`servidor mcp`, `"algoritmo BM25 com pesos"`, `comportamento do watcher`) casam **zero** documentos no `gen_vault.ps1` atual (`grep` = 0), e o teste só faz `t.Logf` — nem teto, nem guarda de corpus. `bench_test.go:173-178` registra que essa mesma troca de corpus quebrou o benchmark em 2026-09-01; o teste RNF não pôde reportar porque não afirma nada.

Conserto:

1. Guarda de corpus, no início: para cada consulta da tabela, rodar e exigir `len(res.Results) > 0`; se zero, `t.Fatalf("a consulta %q nao casa nada no corpus %s; o corpus mudou ou a consulta esta errada", q, caminho)`. **Fatal, não Skip** — um corpus que não serve é defeito de ambiente que o gate deve ver.
2. Para as três consultas sem hits: substituir por três que casam (conferir com `grep -c` no cofre gerado por `gen_vault.ps1 -Notes 5000 -Seed 42`); registrar no comentário da tabela o `grep -c` de cada uma.
3. Teto: `if elapsed > teto { t.Fatalf(...) }` com `teto` = o RNF-01 do `PRD.md` × 2 (folga de máquina), atrás de `//go:build !race` **ou** do mesmo guard que `search`/`service` já usam para tetos (ler `internal/service/latencia_*_test.go` e copiar o padrão). Com `-race` o teste roda a guarda de corpus e pula só o teto — não o teste.
4. RNF-07 (`:91-96`): mesmo tratamento.

- [ ] **Step 2: `parity_test.go:138-205` — iterar `got`, e falhar**

Mecanismo: as asserções iteram `want` (a referência do plugin); uma referência com listas vazias corre zero comparações e reporta paridade. E o teste **pula** quando o corpus de paridade falta, logo `verify.ps1` fica verde sem paridade.

Conserto:

1. Cada bloco de comparação vira comparação de **conjuntos**: `slices.Sort` em `got` e em `want`, `if !slices.Equal(got, want) { t.Errorf("%s: got %v want %v", campo, got, want) }`. Isso cobre os dois sentidos.
2. Guarda de referência vazia: `if len(want.Links)+len(want.Tags)+len(want.Headings) == 0 { t.Fatalf("referencia %s vazia; o dump nao rodou", nome) }` (adaptar aos campos reais).
3. Skip → Fatal quando `testdata/parity/` não existe? **Não** — o corpus é gerado por um plugin de dev do Obsidian e pode não estar em toda máquina. Regra: o teste pula **só** se o diretório não existe, com `t.Skipf` cujo texto diz o caminho; se existe e está incompleto, falha. E `scripts/verify.ps1` ganha uma etapa que **lista** os testes pulados (`go test -v … | grep -c SKIP`) e imprime `[!] N testes pulados` — não falha, mas aparece.

- [ ] **Step 3: Orçamentos onde só havia `Logf`**

- `writer/diff_test.go:71`: `AllocsPerRun` sem orçamento. Medir 7 vezes nesta máquina, anotar o máximo, orçamento = máximo × 1,5 arredondado para cima, `if allocs > orcamento { t.Fatalf(...) }`, com o número medido no comentário e a data.
- `search/persist_test.go:281`, `index/movenote_semvault_test.go:77`: ler o que cada `Logf` imprime; se é uma métrica com RNF, mesmo tratamento; se é diagnóstico sem regra, apagar o `Logf` (ruído no `-v`).

- [ ] **Step 4: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde, e a etapa nova imprime `[!] N testes pulados`.

```bash
git add internal/service/rnf5000_test.go internal/index/parity_test.go internal/writer/diff_test.go internal/search/persist_test.go internal/index/movenote_semvault_test.go scripts/verify.ps1
git commit -m "test: rnf5000 asserts its corpus and ceilings, parity compares both directions, budgets replace Logf"
```

#### Verificações

- `grep -c` de cada consulta de `rnf5000_test.go` no cofre gerado, colado (8 números, todos > 0).
- Prova da guarda: trocar temporariamente uma consulta por `"xyzzy-inexistente"`, rodar, FAIL nomeando a consulta, restaurar. Colar.
- Prova da paridade: editar temporariamente um `.json` de referência apagando um link, rodar `TestParity…`, FAIL; restaurar (é `testdata/`, `git diff` confere). Colar.
- Os 7 valores medidos de `AllocsPerRun` e o orçamento derivado, no relatório e no comentário do teste.
- `verify.ps1` verde com a contagem de SKIP visível.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nenhum número no teste que não tenha sido medido nesta máquina nesta tarefa; a data e a máquina no comentário.
- Teto de latência atrás do mesmo mecanismo que `search`/`service` já usam (copiar, não inventar).
- Não rodar `test_orphans.ps1` ao mesmo tempo que as medições.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: as regras provadas são guardas de teste, e as provas são as três edições temporárias das Verificações (consulta inexistente, referência mutilada, e — para o orçamento — `orcamento = 0` deve falhar).

#### Contrato de relatório

`task-163-report.md`: status, SHA, os `grep -c`, as três saídas de FAIL, as sete medições, última linha do `verify.ps1` e a linha `[!] N testes pulados`.

