### Task 53: Lote mecânico da revisão do M3

Nove itens, nenhum com decisão de projeto. Todos com o antes e o depois escritos.

#### Os itens

**1. `internal/search/inverted_test.go:69` — teste que não pode falhar.**

```go
func TestInvertedConcurrencyRace(_ *testing.T) {
```

Descarta o `t`, então não pode chamar `t.Error` nem `t.Fatal`: depende só de o `-race` tropeçar, e qualquer defeito que não seja corrida de dados passa sem relato. Receba `t *testing.T` e **afirme** algo depois das goroutines — que o número de termos é o esperado, que nenhuma leitura devolveu lista corrompida. A regra do projeto manda `_` para parâmetro que existe por consistência de assinatura; `t` num teste é a única coisa que não é.

**2. `cmd/gobsidian/serve.go:121` — `_ = search.SaveInvertedCache(...)`.** Falha ao gravar o cache não é reportada, e o boot seguinte reconstrói tudo sem ninguém saber por quê. Logue em `Warn`.

**3. `cmd/gobsidian/serve.go:116` — nota ilegível descartada em silêncio.**

```go
if data, err := v.ReadAll(ctx, p); err == nil { ... }
```

Sem `else`, sem contador, sem log. É a regra que `vault.SkippedEntries()` e os contadores do watcher existem para cumprir: **entrada descartada em silêncio fica inalcançável e indiagnosticável**. Conte e logue.

**4. As constantes do BM25 estão fixadas por ordenação, não por valor.** As cinco mutações da Task 46 usaram valores extremos — k1 1.2→10.0, b 0.75→0.0, título 3.0→1.0, headings 2.0→0.5, corpo 1.0→0.0 —, todos cruzando fronteira qualitativa. Medido na revisão: perturbação pequena **sobrevive**.

```
k1 1.2 -> 0.9        [!!] SEM COBERTURA
b 0.75 -> 0.5        [!!] SEM COBERTURA
headings 2.0 -> 1.5  [!!] SEM COBERTURA
```

**A decisão aqui é deliberada e não é "fixe os floats":** travar o valor exato de `k1` e `b` por teste engessa o ranking e transforma qualquer ajuste futuro em quebra de suíte, e ajustar esses parâmetros é uma medição com corpus, não um refactor. O que precisa mudar é a **honestidade do que o teste diz**: renomeie `TestBM25ParamK1` e `TestBM25ParamB` para o que eles verificam de fato — que a saturação e a normalização por comprimento **estão ligadas** —, e escreva no comentário de cada um que o valor exato **não** está fixado, e por quê. Um teste chamado `ParamB` que sobrevive a `b=0.5` mente pelo nome.

**5. `WeightRaw = 1.5` e `WeightReduced = 1.0` entraram além das cinco constantes que o plano nomeou.** Não é defeito — `ARCHITECTURE.md` §6.2 exige que a forma crua pontue acima da reduzida. Documente as duas no código com o motivo, e registre no relatório que são sete constantes, não cinco.

**6. `internal/service/search_test.go:264` — verificação mais fraca que a pedida.** `TestVaultSearchEmptyQueryMetadataOnly` afirma `Score == 0.0` como prova de que o índice de texto não foi tocado. O plano pedia *"prove contando"*. Acrescente um contador de consultas ao índice invertido, ou uma asserção que só o caminho de metadados pode satisfazer.

**7. Apague `.superpowers/sdd/2026-07-25-gobsidian-v01/review-5cf441d..5cf441d.diff`** — 104 bytes, pacote de revisão de intervalo vazio, gerado antes de a tarefa ter commit.

**8. Rode `pwsh -File scripts/audit_reports.ps1`** e resolva o que sobrar nos relatórios 43–50.

**9. Registre no ledger** as Tasks 51, 52 e 53 com o intervalo real, conferindo cada SHA com `git cat-file -t`.

#### Verificações além dos passos

- Os nove itens, um por linha, com o antes e o depois.
- `golangci-lint run ./internal/... ./cmd/...` sai limpo? Cole a saída **e** a de `golangci-lint version`.
- **Prova de mutação:** para o item 3, remova o contador de descarte e confirme que um teste nomeado reprova. Se não houver teste que o pegue, escreva-o — contador que ninguém verifica é o `alias_collisions: 0` de novo.
- Os testes renomeados no item 4 dizem, no nome e no comentário, o que verificam de fato?

#### Regras de execução e contrato de relatório

Idênticos aos da Task 51. Relatório em `.superpowers/sdd/task-53-report.md`.

**Files:** Modify `internal/search/inverted_test.go`, `internal/search/bm25.go`, `internal/search/bm25_test.go`, `internal/service/search_test.go`, `cmd/gobsidian/serve.go`, ledger; Delete o pacote de revisão vazio
**Commit:** `fix(search,cmd): stop swallowing errors and name the tests for what they check`

---

## M3 a M6 — escopo em altitude de tarefa

Cada marco recebe seu próprio plano detalhado quando chegar a vez, escrito contra o código que existir então. Planejá-los agora em nível de passo produziria detalhe que envelhece antes de ser usado.

### M2 — Watcher

**Detalhado acima, como Tasks 27–32.** Deixou a altitude de tarefa porque chegou a vez: o plano por passo agora é escrito contra o código que existe, que era a condição para escrevê-lo.

### M3 — Busca (1 semana)

**Detalhado acima, como Tasks 43–50.** Deixou a altitude de tarefa porque chegou a vez: o plano por passo agora é escrito contra o código que existe, que era a condição para escrevê-lo. A tabela abaixo fica como o mapa de origem — S1 virou a Task 44, S2 a 45, S3 a 46, S4 a 47, S5 a 48, S6 a 49; a Task 43 não estava aqui e entrou por promoção de um Minor do M1, e a 50 é o fechamento.

| Tarefa | Entregável | Critério |
|---|---|---|
| S1 | `search/analyzer.go` | Indexação dupla: forma crua e reduzida; teste com corpus jurídico real |
| S2 | `search/inverted.go` | Dicionário de termos e posting lists |
| S3 | `search/bm25.go` | k1=1.2, b=0.75; pesos título 3x, headings 2x, corpo 1x |
| S4 | `search/snippet.go` | Trecho com destaque, `snippet_chars` respeitado |
| S5 | Tool `vault_search` | Filtros combináveis; query vazia redireciona para o caminho de metadados |
| S6 | `index/cache.go` e `search/persist.go` | Cabeçalho com `format_version`, `parser_version`, `analyzer_version`; RNF-02 medido |

Q3 do PRD §11 fecha aqui: medir o custo real de reconstruir a busca a partir do índice de metadados já carregado decide se o índice invertido também vai para o cache.

### M4 — Escrita (2 semanas)

| Tarefa | Entregável | Critério |
|---|---|---|
| E1 | `writer/lock.go` | Mutex por caminho; teste com escritas concorrentes na mesma nota |
| E2 | `writer/atomic.go` | Temporário no mesmo diretório, `Sync`, rename com retry; teste de crash injetado, 1.000 iterações |
| E3 | `writer/diff.go` | Myers sobre linhas, ~150 linhas, sem dependência |
| E4 | `writer/section.go` | `note_append` e `note_patch` por heading; EOL e BOM preservados |
| E5 | `writer/block.go` | `replace_block` por `^id` |
| E6 | Tools `note_create`, `note_append`, `note_patch` | `dry_run` e `expected_hash` em todas |
| E7 | `--read-only` verificado | Tools de escrita ausentes de `ListTools`, não apenas rejeitadas |

RNF-11 é o critério de bloqueio deste marco: zero notas corrompidas em 1.000 iterações de crash injetado. É a única parte do produto que pode destruir dados, e vem depois de tudo por isso.

### M5 — Refatoração do cofre (1 semana)

| Tarefa | Entregável | Critério |
|---|---|---|
| R1 | `writer/linkrewrite.go` | Alias, âncora e forma original preservados; `Link.Raw` é o insumo |
| R2 | Tool `note_move` | Falha parcial reporta com precisão o que foi aplicado |
| R3 | Tool `note_delete` | `to_trash` padrão; relatório prévio de links que quebrarão |
| R4 | Âncoras quebradas em `vault_stats` | Já implementado em M1; exposto aqui no relatório de impacto |

### M6 — Endurecimento (1 semana) — v1.0

| Tarefa | Entregável | Critério |
|---|---|---|
| H1 | `bench.yml` no CI | RNF-01 a RNF-08 medidos; regressão acima de 20% falha o build |
| H2 | `gen_vault.ps1` determinístico | Cofre sintético de 5.000 notas reproduzível |
| H3 | Subcomandos `index`, `search`, `inspect` | RF-52 |
| H4 | `go vet -vettool` com `netcheck` no CI | Segunda parte da garantia de RNF-30 |
| H5 | Release v1.0.0 | Binários para os três sistemas |

---

## Auto-revisão do plano

**Cobertura da especificação.** Requisitos do PRD mapeados a tarefas: RF-01, RF-02 e RF-08 → Task 19; RF-03 a RF-05 → M2; RF-06 → M3/S6; RF-07 → M2 (o parsing de `.gitignore` entra com o watcher, que é quem precisa dele); RF-10 a RF-17 → Tasks 12–18; RF-18 → Task 17; RF-19 (callouts, P2) → pós-1.0, não planejado; RF-20 a RF-25 → M3; RF-26 (P2) → Q1 em aberto; RF-30 a RF-38 → M4; RF-35 e RF-36 → M5; RF-40 a RF-44 → Tasks 3–6 e 11; RF-50 e RF-51 → Tasks 9 e 10; RF-52 → M6/H3; RF-53 → Global Constraints e Task 9; RF-54 (P2) → fora da v1 por D7; RF-55 → Tasks 9 e M4/E7; RF-60 a RF-63 → Tasks 8, 19 e 20. RNF-01 e RNF-07 → Task 26/Step 2; RNF-02 a RNF-06, RNF-08, RNF-09 → M6/H1; RNF-10 → Task 11; RNF-11 → M4/E2; RNF-12 → Tasks 19 e 21; RNF-13 → Task 9; RNF-20 a RNF-23 → Tasks 7, 8 e 10; RNF-24 → Task 9; RNF-30 → Task 1 e M6/H4; RNF-31 e RNF-32 → Task 7; RNF-33 → Tasks 9 e M4/E7.

**Lacuna conhecida.** RNF-32 (links simbólicos que apontem para fora do cofre não são seguidos) tem teste apenas indireto na Task 7, porque criar symlink no Windows exige privilégio elevado e o teste falharia em máquina de desenvolvimento comum. `vault.Walk` usa `filepath.WalkDir`, que não segue symlinks por padrão — a propriedade vale por construção. Adicione um teste explícito em M6, marcado com `t.Skip` quando a criação de symlink falhar por permissão.

**Consistência de tipos.** `vault.CanonicalPath` é o tipo de caminho em toda a cadeia — `vault`, `index`, `service` — e nunca vira `string` solto. `parser.Link` é embutido em `index.ResolvedLink`, e não duplicado. `Note.Hash` é `uint64` no índice e string hexadecimal no contrato MCP; a conversão acontece em `service/read.go` e em nenhum outro lugar. `service.Index` é declarado como interface na Task 9 e satisfeito por `*index.Index` na Task 23, o que é o que permite testar o serviço sem construir um índice completo.

**Sem marcadores pendentes.** Nenhum passo contém "TBD", "implementar depois" ou "similar à tarefa N". As Tasks 15 a 17, 20 a 24 descrevem regras em prosa numerada em vez de código completo por escolha: são extensões cujo formato já foi estabelecido integralmente na Task 14, e as regras são o que o implementador precisa saber que os testes não dizem sozinhos. Todo teste está escrito por inteiro.

---

## Referências

| Documento | Papel |
|---|---|
| `docs/PRD.md` | Requisitos, prioridades, decisões fechadas (D1–D13), questões em aberto |
| `docs/ARCHITECTURE.md` | Camadas, fluxos, modelo de dados, decisões arquiteturais (AD-01–AD-09) |
| `docs/ESTRUTURA.md` | Árvore de diretórios, convenções de código, build |
| `docs/TOOLS.md` | Contrato de cada tool: schemas, retornos, códigos de erro |
| `docs/WINDOWS.md` | OneDrive, MAX_PATH, casing, fsnotify, registro no Claude Desktop |
