### Task 168: `mcpsrv` para de aplicar defaults que o `service` já aplica

**Files:**
- Modify: `internal/mcpsrv/tools_read.go:27-38,:185-209,:257-265`
- Modify: `internal/mcpsrv/tools_read_test.go` (o teste que prova que o default vem do serviço)

**Interfaces:**
- Consumes: `service.ComTeto(int) int` (`errors.go:147`, padrão `LimitePadrao = 100`, teto 500), `service.ValidarEnum` (`errors.go:166`, vazio → padrão), `service.Search` (`search.go:146-163`: `Limit <= 0 → 20`, `> 200 → 200`, `SnippetChars <= 0 → search.DefaultSnippetChars`), `service.LinkGraph` (`graph.go:101-103`), `service.ListNotes`/`NoteList` (`graph.go:462-470`).

Mecanismo: `tools_read.go` aplica `limit := 20`, `snippetChars := 240`, `offset := 0`, `limit := 100`, `tagMode := "all"`, `sort := "path"`, `order := "asc"`, `depth := 1`, `direction := "both"` antes de chamar o serviço, que aplica os mesmos defaults de novo. Dois lugares com o mesmo número são dois lugares para o número divergir. O `service` é a conta (é ele que a CLI usa); o boundary MCP só desembrulha ponteiros.

- [ ] **Step 1: Tabela dos defaults**

Antes de tocar, montar no relatório: parâmetro, default em `mcpsrv`, default no `service`, default em `TOOLS.md`. Só apagar onde os três coincidem. Onde divergirem: o `TOOLS.md` é o contrato; ajustar o `service` para ele e registrar como "divergência encontrada". Onde o `service` **não** tem default (candidatos: `depth`, `recursive`, `include_broken`, `include_embeds`), o default fica no `mcpsrv` — **não** inventar um no serviço nesta tarefa; anotar.

- [ ] **Step 2: Teste que prova que o default vem do serviço**

Em `tools_read_test.go`, um subteste por tool: chamar `vault_search` sem `limit`/`snippet_chars`, afirmar `effective_limit == 20` e `effective_snippet_chars == 240` na resposta; `note_list` sem `sort`/`order`, afirmar ordem por `path` ascendente; `link_graph` sem `direction`, afirmar que vêm arestas nos dois sentidos (fixture com um link de ida e um de volta).

- [ ] **Step 3: Apagar os blocos**

Os `if in.X != nil { x = *in.X }` viram `x := 0; if in.X != nil { x = *in.X }` (o zero é o que o serviço interpreta como "não informado") — ou, mais simples, passar o ponteiro desreferenciado com `valorOuZero(in.Limit)`:

```go
// valorOuZero desembrulha um parametro opcional. Zero e "nao informado" para
// o service, que aplica o padrao — a UNICA conta de cada padrao.
func valorOuZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
```

Para strings: passar `in.Sort` cru (vazio = padrão em `ValidarEnum`).

Prova de que a conta é uma: `mutate.ps1 -Path internal/service/search.go -Anchor "opts.Limit = 20" -Replacement "opts.Limit = 21" -Test TestVaultSearchDefaultVemDoServico -Package ./internal/mcpsrv/` — exit 0 (antes desta tarefa, o `20` do `mcpsrv` mascarava a mutação).

- [ ] **Step 4: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde (`check_tool_params` confere schema × código).

```bash
git add internal/mcpsrv/tools_read.go internal/mcpsrv/tools_read_test.go
git commit -m "refactor(mcpsrv): stop re-applying defaults the service owns"
```

#### Verificações

- A tabela do Step 1 no relatório, com uma linha por parâmetro.
- `mutate.ps1` do Step 3, exit 0, colado.
- `wc -l internal/mcpsrv/tools_read.go` antes e depois.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nenhum default novo no `service` — se falta, fica no `mcpsrv` e vai para o relatório.
- Schema (`jsonschema` tags) não muda nesta tarefa.

#### Comando de mutação

`pwsh -File scripts/mutate.ps1 -Path internal/service/search.go -Anchor "opts.Limit = 20" -Replacement "opts.Limit = 21" -Test TestVaultSearchDefaultVemDoServico -Package ./internal/mcpsrv/` — exit 0.

#### Contrato de relatório

`task-168-report.md`: status, SHA, a tabela, a saída do `mutate.ps1`, `wc -l`, última linha do `verify.ps1`.

