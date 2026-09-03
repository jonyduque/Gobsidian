### Task 169: Simplificações mecânicas com eficiência igual — cada uma com `benchstat`

**Files:**
- Modify: `internal/service/graph.go:69-75` (`chaveDaAresta` → struct), `:261,:283-294,:396-402` (`TagNode.Children []any` → `[]TagNode`), `:331-366` (`tagListHierarchical` filtra dentro)
- Modify: `internal/service/search.go:192,:251,:393` (`resultadoVazio()`), `:249-269,:392-411` (`pagina()`), `:231` (`make(0, len)`)
- Modify: `internal/service/write.go:561-628` (`MoveNoteResult` de erro 5× → uma função), `:22` `hashDoConteudo` passa a ser usada em `read.go:404`, `graph.go:481,:583`
- Modify: `internal/index/index.go:104-126` (`TotalSize`: `RLock` no hit, upgrade só no miss)
- Modify: `internal/index/resolve.go:410-421` (usar `vivosLocked :249`)
- Modify: `internal/writer/lock.go:14` (`normalizeKey` delega à conta de chave de `index/chave.go` — ver Step 3)

**Interfaces:**
- Consumes: benchmarks da Baseline (`LinkGraphBothDepth2`, `TagListPlano`, `TagListHierarquico`, `SearchLimit200Cache`, `TotalSizeRepetido`) e os binários `antes_*` em `%LOCALAPPDATA%\gobsidian-bench\2026-09-02\`.

Regra da tarefa: cada item é **um commit**; cada commit que toca caminho quente traz `benchstat` com **≥ 7 amostras intercaladas** contra o binário `antes_<pkg>.test.exe` e o veredito é `~` ou melhora. Uma piora com `p < 0.05` reverte o item (novo commit que desfaz, não `git reset`) e vai para o relatório.

- [ ] **Step 1: `chaveDaAresta` → struct**

```go
// chaveDaAresta identifica uma aresta no BFS. Struct, e nao Sprintf: a
// identidade e exata por construcao e nao custa alocacao por aresta.
type chaveDaAresta struct {
	Source, Target, Kind, Alias, Anchor string
}
```

Substituir o `Sprintf` e o `map[string]…` por `map[chaveDaAresta]…`. Bench: `LinkGraphBothDepth2` — esperado: allocs/op cai (41 na baseline); `sec/op` `~` ou melhor.

- [ ] **Step 2: `TagNode.Children []TagNode`**

Trocar `[]any` por `[]TagNode`; remover o type-assert/write-back em `:283-294` e o boxing em `:396-402`. JSON de saída **idêntico** — prova: golden do `tag_list` hierárquico (se não existir, criar `testdata/tag_list_hierarquico.json` a partir do binário `antes`, e o teste compara byte a byte). Bench: `TagListHierarquico` (6,163 ms / 10 880 allocs na baseline).

- [ ] **Step 3: Os demais**

- `tagListHierarchical`: filtrar dentro do loop e usar `ix.tags` como o ramo plano. Bench: `TagListHierarquico`, `TagListPlano`.
- `resultadoVazio()` + `pagina()` em `search.go`; `make([]…, 0, len(rawHits))` em `:231`. Bench: `SearchLimit200Cache`.
- `MoveNoteResult` de erro: `func moveNoteErro(req MoveNoteRequest, err error) (MoveNoteResult, error)`.
- `hashDoConteudo` nos três sítios (`read.go:404`, `graph.go:481,:583`) — atenção: os três formatam `n.Hash` (já `uint64`), e `hashDoConteudo` recebe `[]byte`; criar `func formatarHash(h uint64) string { return fmt.Sprintf("%016x", h) }` e fazer `hashDoConteudo` chamá-la. Uma conta do formato.
- `TotalSize`: `RLock`; se válido, devolve; senão `RUnlock`, `Lock`, **re-conferir** (outro leitor pode ter preenchido), calcular. Bench: `TotalSizeRepetido` (5,610 µs, 0 allocs) — `~`.
- `resolve.go:410-421` → `vivosLocked`.
- `writer/lock.go:14 normalizeKey`: hoje `ToLower` sem NFC. A chave canônica mora em `index/chave.go`, mas `writer` **não** importa `index` (grafo) e não vai importar. Opção que respeita o grafo: `text.ChaveDeCaminho` (se `chave.go` já delega a `text`, usar; se não, `BLOCKED` com a proposta de mover a conta para `text`, que `writer` pode importar — `writer → parser, vault`, e `parser → text`, então `writer → text` é aresta nova **em folha de teste?** Não: `text` é folha, `writer` ganhar `text` é aresta nova no produto e precisa de justificativa no relatório: "uma conta por regra" é a justificativa; o orquestrador decide).

- [ ] **Step 4: `benchstat` final**

Run: `go test -c -o %LOCALAPPDATA%\gobsidian-bench\2026-09-02\depois169_service.test.exe ./internal/service/` (e `index`), depois 7 rodadas intercaladas `antes`/`depois` de `-run '^$' -bench 'LinkGraphBothDepth2|TagList|SearchLimit200Cache' -benchmem -count=1`, concatenar, `benchstat antes.txt depois.txt`. Colar.

- [ ] **Step 5: Gate**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

Commits (um por item, mensagens):

```
refactor(service): edge key as a struct, not a formatted string
refactor(service): TagNode.Children is []TagNode, JSON unchanged
refactor(service): filter inside tagListHierarchical and reuse the flat branch
refactor(service): one empty result, one pagination, one move error
refactor(service): one account for the hash format
refactor(index): TotalSize takes the read lock on the memoized hit
refactor(index): resolve reuses vivosLocked
refactor(writer): lock key delegates to the shared key account
```

#### Verificações

- `benchstat` colado com ≥ 7 amostras por lado; nenhuma linha com piora `p < 0.05`.
- Golden do `tag_list` hierárquico byte a byte igual ao do binário `antes`.
- Um commit por item; `git log --oneline <base>..HEAD` colado.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. Reverter = commit novo.
- Não rodar `test_orphans.ps1` nem outra medição durante o `benchstat`.
- Nenhum item muda JSON de saída, ordem de resultados ou código de erro.
- Aresta nova no grafo (`writer → text`) só com a justificativa no relatório e a atualização do bloco do grafo em `CLAUDE.md` no mesmo commit.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: são refatorações que preservam comportamento; a prova é o golden byte a byte e o `benchstat`.

#### Contrato de relatório

`task-169-report.md`: status, SHAs (um por item), `benchstat`, resultado do golden, decisão sobre `normalizeKey`, última linha do `verify.ps1`.

