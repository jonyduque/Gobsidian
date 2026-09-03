### Task 167: `*ForTest` sai do produto para `export_test.go`

**Files:**
- Modify: `internal/index/persist.go:86` (`WriteIndexCacheForTest` → apagar)
- Create: `internal/index/export_test.go`
- Modify: `internal/search/persist.go:46` (`WriteCacheForTest` → apagar)
- Create: `internal/search/export_test.go`
- Modify: `internal/mcpsrv/server.go:123` (`RegisterPanicProbeForTest` → apagar)
- Create: `internal/mcpsrv/export_test.go`

**Interfaces:**
- Consumes: os únicos chamadores são `internal/index/persist_test.go` (`package index_test`), `internal/search/persist_test.go` (`package search_test`), `internal/mcpsrv/server_test.go` (`package mcpsrv_test`) — conferido em 2026-09-02 com `grep -rln`. Um `export_test.go` em `package index` exporta o símbolo **só para os testes do mesmo diretório**, que é exatamente o caso.

- [ ] **Step 1: Um `export_test.go` por pacote**

`internal/index/export_test.go`:

```go
package index

import "io"

// WriteIndexCacheForTest expoe o codificador do cache aos testes externos do
// pacote. Vive aqui, e nao em persist.go, porque um simbolo que so teste chama
// nao pertence ao binario do produto.
func WriteIndexCacheForTest(w io.Writer, h CacheHeader, notes []*Note, assets []*Asset) error {
	return escreveIndexCache(w, h, notes, assets)
}
```

`internal/search/export_test.go` — mesmo desenho sobre `escreveCache` (copiar a assinatura de `persist.go:46`).

`internal/mcpsrv/export_test.go` — mover o corpo de `RegisterPanicProbeForTest` de `server.go:123` para cá, sem alterar.

- [ ] **Step 2: Apagar as três do produto e rodar**

Run: `go test -race ./internal/index/ ./internal/search/ ./internal/mcpsrv/ 2>&1 | tail -5` — Expected: `ok` nos três.
Run: `go build ./... && grep -rn "ForTest" --include=*.go internal cmd | grep -v _test.go` — Expected: vazio.

- [ ] **Step 3: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/index/persist.go internal/index/export_test.go internal/search/persist.go internal/search/export_test.go internal/mcpsrv/server.go internal/mcpsrv/export_test.go
git commit -m "refactor: move test-only exports out of the product into export_test.go"
```

#### Verificações

- `grep -rn "ForTest" --include=*.go internal cmd | grep -v _test.go` vazio, colado.
- `go test -race` dos três pacotes, colado.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Não renomear os símbolos: os testes que os chamam não mudam.
- Se `gopls references` mostrar chamador **fora** do diretório do pacote, `BLOCKED` — `export_test.go` não alcança.

#### Comando de mutação

Esta tarefa não tem prova de mutação: é um move sem regra. A prova é o `grep` vazio e a suíte verde.

#### Contrato de relatório

`task-167-report.md`: status, SHA, o `grep`, a saída dos testes, última linha do `verify.ps1`.

