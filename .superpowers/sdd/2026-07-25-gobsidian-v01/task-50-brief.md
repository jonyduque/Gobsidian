### Task 50: Fechamento do M3

#### O que fazer

1. **Rode o portão completo e cole a saída de cada comando:**

```bash
pwsh -File scripts/verify.ps1
golangci-lint version
golangci-lint run ./internal/... ./cmd/...
pwsh -File scripts/build.ps1
pwsh -File scripts/test_orphans.ps1 -Cycles 100
pwsh -File scripts/audit_reports.ps1
```

O gate de órfãos precisa de **zero órfãos em 100 ciclos**, e a linha de motivos precisa mostrar que um mecanismo disparou — uma contagem limpa sem mecanismo é o falso verde que este gate já produziu. **Nunca enfraqueça o teste para fazê-lo passar.**

2. **Atualize `docs/OPERACAO.md`** com os números que as Tasks 48 e 49 mediram. Onde não houve medição, escreva **"não medido"**. Alvo não atingido e registrado é informação; alvo não medido apresentado como resultado é ficção com aparência de tabela.

3. **Feche a Q3 no PRD §11** com a data e o número, se a Task 49 não tiver feito.

4. **Registre as Tasks 43–50 no ledger**, com o intervalo de commits real de cada uma. **Confira cada SHA com `git cat-file -t`** — o ledger já apontou uma tarefa para o commit de outra, e `scripts/audit_reports.ps1` agora acusa isso.

5. **Tague o marco:** `git tag -a m3-search`.

6. **Triagem dos Minors diferidos.** A lista de M1 continua no ledger sob `## Minor diferidos de M1`. Os itens de `slug.go` não são mais dívida: a colisão de slug foi **decidida como aceita** em 2026-07-29. Os demais carregam para o M4 com o motivo escrito.

#### Regras de execução

- **Não afirme estado que você não verificou.** Não diga que o marco está pronto sem a saída do gate colada.
- Demais regras idênticas às da Task 43.

**Files:** Modify `docs/OPERACAO.md`, `docs/PRD.md`, ledger
**Commit:** `docs: close M3 with measured numbers and the Q3 decision`

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
