### Task 179: a — `vault_search` devolve só `results`; `hits` some

**Files:**
- Modify: `internal/service/search.go:105` (campo `Hits`), `:193-194`, `:252-253`, `:370-371` (atribuições)
- Modify: `internal/service/match_offset_test.go:35,:39,:78-81,:118,:122`, `internal/service/max_results_test.go:27-28,:38`, `internal/mcpsrv/filtro_data_test.go:85-99` (`Hits` → `Results`)
- Create: `internal/mcpsrv/search_sem_hits_test.go`
- Modify: `docs/TOOLS.md:62` ("`results` (e `hits`)" → "`results`"), `docs/ESTADO.md:139` (medição depois), `docs/SUGESTOES.md:1372` (P5 marcado feito)

**Interfaces:**
- Consumes: `sessaoComBusca(t, root)` (`filtro_data_test.go:23`) para abrir uma sessão MCP em memória.
- Produces: `SearchResponse` sem `Hits`. Nada depois desta Task depende de `hits`.

- [ ] **Step 1: O teste do contrato — falha enquanto `hits` existe**

`internal/mcpsrv/search_sem_hits_test.go`:

```go
package mcpsrv_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestVaultSearchNaoDevolveHits(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# A\n\npalavra comum\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	session, ctx := sessaoComBusca(t, root)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "vault_search", Arguments: map[string]any{"query": "comum"},
	})
	if err != nil || res.IsError {
		t.Fatalf("CallTool: err=%v isError=%v", err, res.IsError)
	}
	bruto, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(bruto, &out); err != nil {
		t.Fatalf("resposta ilegivel: %v\n%s", err, bruto)
	}
	if _, tem := out["hits"]; tem {
		t.Fatalf("a resposta ainda traz \"hits\":\n%s", bruto)
	}
	var results []json.RawMessage
	if err := json.Unmarshal(out["results"], &results); err != nil || len(results) != 1 {
		t.Fatalf("results = %s (err=%v), quero 1 item", out["results"], err)
	}
}
```

O import do SDK é o mesmo que `filtro_data_test.go` usa — copiar o caminho de lá. Run: `go test ./internal/mcpsrv/ -run TestVaultSearchNaoDevolveHits` — Expected: FAIL (`a resposta ainda traz "hits"`).

- [ ] **Step 2: Apagar o campo e as atribuições**

`internal/service/search.go:105`: apagar `Hits []SearchHit \`json:"hits,omitempty"\``. Em `:193-194`, `:252-253`, `:370-371`: apagar a linha `Hits: …` (ou `resp.Hits = …`), manter `Results`. `go build ./...` — Expected: os testes que leem `Hits` não compilam — é a lista do Step 3.

- [ ] **Step 3: Testes migram para `Results`**

- `match_offset_test.go:35,:39,:78-81,:118,:122`: `res.Hits` → `res.Results`.
- `max_results_test.go:27-28`: `if len(res.Results) != 5 { t.Fatalf("results = %d, quero 5", len(res.Results)) }`; `:38` idem com o número que o teste esperava para `Hits`.
- `mcpsrv/filtro_data_test.go:85-99`: apagar o campo `Hits` da struct anônima e o `if n == 0 && len(out.Results) > 0`; `n := len(out.Results)`.

Run: `go test -race ./internal/service/ ./internal/mcpsrv/` — Expected: PASS, inclusive `TestVaultSearchNaoDevolveHits`.

- [ ] **Step 4: Prova manual de mutação**

Recolocar temporariamente o campo `Hits` e **uma** atribuição (`:193`), rodar `go test ./internal/mcpsrv/ -run TestVaultSearchNaoDevolveHits` — Expected: FAIL. Remover de novo. Colar. `grep -rn '"hits"\|\.Hits\b' --include=*.go . docs/` — Expected: vazio (fora de `ESTADO.md`, que registra a história).

- [ ] **Step 5: Medição M3 depois**

`search --json --limit 200 --vault <vault_5000> --cache-dir <dir> "execucao"` com o binário deste commit: bytes do arquivo indentado (o que o comando imprime) e do compacto (`python -c "import json,sys;print(len(json.dumps(json.load(open(sys.argv[1])),separators=(',',':'))))" saida.json`). Expected: compacto ≈ 97 787 (Baseline M3 "sem `hits`"), indentado bem abaixo de 216 304. Colar os dois números em `docs/ESTADO.md:139` ao lado dos de antes.

- [ ] **Step 6: Documentação e commit**

`TOOLS.md:62`: "Objeto contendo `results`, `total`, `truncated`, …". `SUGESTOES.md:1372`: acrescentar "— feito em 2026-09 (Task 179)". `ESTADO.md:139`: os números do Step 5.

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/service/search.go internal/service/match_offset_test.go internal/service/max_results_test.go internal/mcpsrv/filtro_data_test.go internal/mcpsrv/search_sem_hits_test.go docs/TOOLS.md docs/ESTADO.md docs/SUGESTOES.md
git commit -m "feat(service)!: vault_search returns results only

BREAKING CHANGE: the hits field, a byte-for-byte copy of results that
doubled the payload, is gone. Read results."
```

#### Verificações

- Step 1 FAIL→PASS colado; Step 4 FAIL colado; `grep` vazio colado.
- Dois números do Step 5 colados e publicados em `ESTADO.md`.
- `git log -1 --format=%B` mostra o rodapé `BREAKING CHANGE:`.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nenhum outro campo de `SearchResponse` muda: `total`, `truncated`, `effective_limit`, `effective_snippet_chars`, `unavailable_snippets` ficam.
- Teste que lia `Hits` passa a ler `Results` com a **mesma** asserção — não afrouxar.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: a prova é a reintrodução manual do Step 4 com o FAIL colado.

#### Contrato de relatório

`task-179-report.md`: status, SHA, saídas dos Steps 1, 3, 4, 5, última linha do `verify.ps1`.

---
