### Task 170: Otimizações medidas — só entra o que o `benchstat` aprova

**Files:**
- Modify (candidatos): `internal/search/inverted.go:214-222` (`removeLocked`: índice reverso doc→termos no delta), `:232,:311,:346` (variante sem `Normalize` para termo já normalizado), `internal/parser/ast.go:336-337` (`bytes.Index`/`bytes.Count`), `internal/writer/linkrewrite.go:52-67` (`bytes.Buffer` com `Grow` único), `internal/service/search.go:443-453` (`EqualFold` + normalização içada)
- Create/Modify: benchmark faltante para `removeLocked` se `update_bench_test.go` não o isolar

**Interfaces:**
- Consumes: Baseline `InvertedUpdateLote` (6,549 s), `RewriteLinksMuitos` (1,496 ms), `ParseNotaLonga` (2,611 ms), `DetectCandidatesNotaLonga` (309,9 µs), `SearchFiltroFrontmatter`, `IndexBuild`, `BuildComHub`; binários `antes_*`.

Regra: **otimização é mudança de comportamento de desempenho, e só entra com número.** Um candidato sem melhora `p < 0.05` **não entra** e vai para o relatório como "sem ganho medido: <benchstat>". Um por commit.

- [ ] **Step 1: `removeLocked` — o único com potencial de ordem de grandeza**

Hoje varre todo termo do delta por remoção: O(termos do delta) por nota removida. `InvertedUpdateLote` na baseline: **6,549 s**. Índice reverso `docTermos map[docID][]termoID` no delta, preenchido no add, consumido no remove. Custo: memória proporcional ao delta (medir `B/op`).

Bench antes/depois de `InvertedUpdateLote`, 7 × 2 intercalados. Se `sec/op` cair com `p < 0.05`: entra, com o `B/op` no commit. Se não: relatório.

- [ ] **Step 2: Os quatro pequenos**

Cada um com o bench da Baseline que o cobre. `bytes.Index` no parser (`ParseNotaLonga`); `bytes.Buffer` no rewrite (`RewriteLinksMuitos`); termo sem re-`Normalize` (`SearchTermoAmploCache`); `EqualFold` no filtro de tags (`SearchFiltroFrontmatter` — **atenção**: a Task 180 vai trocar esse filtro pela chave de tag; se esta tarefa rodar antes, fazer só o içamento da normalização e deixar a comparação para a 180).

- [ ] **Step 3: Gate**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

Commits: `perf(search): reverse doc->terms map makes delta removal O(terms of doc)` etc., cada um com o `benchstat` resumido no corpo da mensagem (a linha do bench: antes → depois, p).

#### Verificações

- Um `benchstat` por candidato, ≥ 7 amostras por lado, colado.
- Para cada candidato: "entrou (p=…)" ou "sem ganho medido".
- `go test -race ./internal/search/ ./internal/parser/ ./internal/writer/ ./internal/service/` verde.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nenhum número no commit ou no relatório que não venha do `benchstat` desta tarefa.
- Não rodar duas medições ao mesmo tempo, nem `test_orphans.ps1`.
- Formato de cache não muda (`CacheFormatVersion` intacto); se a otimização exigir mudar, `BLOCKED`.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: a regra provada é de desempenho e a prova é o `benchstat`. A correção é coberta pela suíte (`TestIndiceRecarregadoEIdenticoAoConstruido` para o delta).

#### Contrato de relatório

`task-170-report.md`: status, SHAs, os `benchstat`, a lista entrou/não entrou, última linha do `verify.ps1`.

---

## Fase 4 — Movimentos entre pacotes (Tasks 171–177)

Todos os movimentos seguem `golang-refactoring`: alias na origem, migração de chamadores, remoção do alias — dois PRs por movimento, nenhum commit com estrutura e comportamento juntos.

Modelo por tarefa: 171 Opus (move cross-package, testes de durabilidade migram); 172 Sonnet com revisor Opus; 173 Sonnet; 174 Opus; 175 Sonnet; 176 Sonnet; 177 Opus.

