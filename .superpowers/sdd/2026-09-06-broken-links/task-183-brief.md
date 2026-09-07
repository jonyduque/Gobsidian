### Task 183: Tool `vault_broken_links`

**Files:**
- Create: `internal/service/broken.go`
- Create: `internal/service/broken_test.go`
- Modify: `internal/mcpsrv/tools_read.go` (registro da tool, struct de input `vaultBrokenLinksInput`)
- Modify: `internal/mcpsrv/*_test.go` onde a lista de tools e afirmada (procure por `"tag_list"` nos testes do pacote)
- Modify: `docs/TOOLS.md` (secao nova, mesmo formato das outras), `README.md` (secao "MCP tools", tabela), `docs/wiki/features/tools-mcp.md`, `docs/PRD.md` (se houver tabela de tools/RF), `docs/ESTRUTURA.md` (se listar arquivos de `service`)
- Verify: `pwsh -File scripts/check_tool_params.ps1` e `pwsh -File scripts/check_readme_anchors.ps1`

**Interfaces:**
- Consumes (Task 182): `index.ResolvedLink{State, Resolved, Context, Link{Target, Anchor, Kind, Alias, Start}}`, `s.index.NotePaths()`, o acessor de nota que `VaultStats` usa, `ComTeto`, `LimitePadrao`, `LimiteTeto`, `ValidarEnum`, `CodeInvalidArgument`.
- Produces:

```go
// BrokenLinksRequest sao os parametros de vault_broken_links.
type BrokenLinksRequest struct {
	// State filtra: "target_missing", "anchor_missing" ou "" (os dois).
	State string `json:"state"`
	// Prefix restringe a origem a um caminho canonico (pasta ou nota). Vazio = cofre inteiro.
	Prefix string `json:"prefix"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// BrokenLink e um link sem alvo ou sem ancora, com o bastante para achar e corrigir.
type BrokenLink struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Anchor  string `json:"anchor,omitempty"`
	Alias   string `json:"alias,omitempty"`
	Kind    string `json:"kind"`  // link.Kind.String()
	State   string `json:"state"` // "target_missing" | "anchor_missing"
	Context string `json:"context,omitempty"`
}

// BrokenLinksResult: Total e a contagem ANTES de offset/limit, como em note_list.
type BrokenLinksResult struct {
	Links []BrokenLink `json:"links"`
	Total int          `json:"total"`
}

func (s *Service) BrokenLinks(_ context.Context, req BrokenLinksRequest) (BrokenLinksResult, error)
```

Regras: ordenacao deterministica por `Source` e depois `Start`; `LinkExternal` nunca entra; `State` invalido -> `CodeInvalidArgument` via `ValidarEnum` (o mesmo caminho de `direction`); `Limit` passa por `ComTeto`; `Offset` negativo vira 0; `Prefix` compara pela mesma chave que o indice usa para caminho (`text.ChaveDeCaminho` ou o que `NotePaths` devolve — nao invente comparacao nova; um `Prefix` que nao casa nada devolve `Total: 0`, nao erro). `Total` e a contagem apos o filtro de `State` e `Prefix`, antes de `Offset`/`Limit`.

- [ ] **Step 1: Teste do service (falha antes)**

`internal/service/broken_test.go`, com `newTestService`/`writeFile` de `limites_enums_test.go`: cofre com `a.md` (`# Topo`, `[[b]]`, `[[nada]]`, `[[b#Nada]]`, `[x](#Topo)`, `[s](https://ex.com)`), `sub/c.md` (`[[outra]]`), `b.md` (`# Sec`). Afirme:
- sem filtro: `Total == 3`, ordem `a.md`(`nada`), `a.md`(`b#Nada`), `sub/c.md`(`outra`); `https://ex.com` e `[x](#Topo)` ausentes;
- `State: "anchor_missing"`: so `b#Nada`, `State == "anchor_missing"`, `Anchor == "Nada"`;
- `Prefix: "sub"`: so `outra`, `Total == 1`;
- `Limit: 1, Offset: 1`: um item, `Total == 3`;
- `State: "quebrado"`: erro com `CodeOf(err) == CodeInvalidArgument`;
- `Limit: 100000` num cofre com `LimiteTeto+1` links quebrados: `len(Links) == LimiteTeto` e `Total == LimiteTeto+1` (o mesmo desenho de `TestNoteListAplicaOTetoDeLimit`: sem o cofre acima do teto, o clamp e inobservavel).

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/service -run TestBrokenLinks -v`
Expected: FAIL de compilacao (`BrokenLinks` indefinido).

- [ ] **Step 3: Implementar `internal/service/broken.go`**

Percorra `s.index.NotePaths()` em ordem (ordene se `NotePaths` nao garantir), para cada nota os `ResolvedLinks` na ordem de `Start`, filtre por estado e prefixo, conte `Total`, aplique offset e limite. Comentario de abertura: o que `vault_stats` conta, esta tool lista — e o motivo de `external` ficar de fora e o mesmo de `graph.go`.

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/service -run TestBrokenLinks -v -race`
Expected: PASS.

- [ ] **Step 5: Registrar em `mcpsrv`**

Em `tools_read.go`, ao lado de `link_graph`:

```go
type vaultBrokenLinksInput struct {
	State  string `json:"state,omitempty" jsonschema:"Filtra por estado: target_missing ou anchor_missing. Omitido devolve os dois."`
	Prefix string `json:"prefix,omitempty" jsonschema:"Restringe a origem a uma pasta ou nota (caminho relativo ao cofre)."`
	Limit  *int   `json:"limit,omitempty" jsonschema:"Maximo de links devolvidos (padrao 100, teto 500)."`
	Offset *int   `json:"offset,omitempty" jsonschema:"Quantos links pular, para paginar."`
}
```

Copie o estilo das tags `jsonschema` que `linkGraphInput` usa (leia-as; se as descricoes la nao usam esse formato, siga o delas). Handler: `valorOuZero` para `Limit`/`Offset`, repassa cru para `s.svc.BrokenLinks`, `toolErr(err)`. `Name: "vault_broken_links"`, `Description: "Todos os links quebrados do cofre: alvo ausente ou ancora ausente, com origem e contexto."`. Confira que os valores de `LimitePadrao`/`LimiteTeto` batem com os numeros na descricao — se nao forem 100/500, escreva os reais.

Teste em `mcpsrv`: onde os testes do pacote listam tools (procure `"tag_list"`), acrescente `vault_broken_links`. Se houver teste que chama tools por nome com um cofre, acrescente uma chamada com `state: "anchor_missing"` e afirme que a resposta tem `total`.

- [ ] **Step 6: check_tool_params e docs**

Run: `pwsh -File scripts/check_tool_params.ps1` — precisa passar (os quatro campos sao lidos no handler como `in.State`, `in.Prefix`, `in.Limit`, `in.Offset`, e no dominio como `.State`, `.Prefix`, `.Limit`, `.Offset`).

`docs/TOOLS.md`: secao `vault_broken_links` no formato das vizinhas (schema, retorno com exemplo, erros: `INVALID_ARGUMENT` para `state` fora do enum). `README.md` secao MCP tools: uma linha na tabela. `docs/wiki/features/tools-mcp.md`: uma linha. Rode `pwsh -File scripts/check_readme_anchors.ps1` e `pwsh -File scripts/check_doc_refs.ps1`. Valide UTF-8 de cada `.md` editado.

- [ ] **Step 7: Prova de mutacao (no passado, saida colada)**

1. Trocar o filtro para deixar `LinkExternal` passar -> teste sem filtro falha (`Total == 4`).
2. Remover o `ComTeto` -> teste do teto falha.
3. Trocar a ordenacao -> teste de ordem falha.
4. Remover `ValidarEnum` -> teste de `state` invalido falha.

- [ ] **Step 8: Gate e commit**

Run: `pwsh -File scripts/verify.ps1`.
Commit: `feat(tools): vault_broken_links lists every missing target and anchor in the vault`

---

