# Relatório de Conclusão — Task 122

Status: DONE
Commit: fa4d959 feat(service): sort link_graph nodes and edges deterministically and enforce limit cap

---

### 1. Evidência de TDD

#### RED (Teste falhando antes da implementação)
**Comando:**
`go test -race ./internal/service/ -run TestLinkGraphOrdemEstavel`

**Saída literal:**
```
--- FAIL: TestLinkGraphOrdemEstavel (0.04s)
    graph_ordem_test.go:76: volta 1 devolveu ordem diferente:
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.120s
```

#### GREEN (Teste passando após a implementação)
**Comando:**
`go test -race ./internal/service/ -run TestLinkGraphOrdemEstavel`

**Saída literal:**
```
--- PASS: TestLinkGraphOrdemEstavel (0.08s)
PASS
ok  	github.com/jonyd/gobsidian/internal/service	1.150s
```

---

### 2. O que mudou por arquivo

- `internal/service/graph.go`: adicionado campo `LimitEfetivo int \`json:"effective_limit"\`` em `GraphResult`, teto `limit = 500` se `limit > 500` em `LinkGraph` preenchendo `res.LimitEfetivo = limit`, e ordenação determinística de `Nodes` por `path` e `Edges` pela tripla `(source, target, kind)` com `slices.SortFunc` e `cmp.Compare`.
- `internal/service/graph_ordem_test.go`: criado teste `TestLinkGraphOrdemEstavel` (50 iterações de `LinkGraph`) e `TestLinkGraphLimitTemTeto`.
- `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`: registrado o progresso da Task 122 no ledger.

---

### 3. Prova de Mutação (`scripts/mutate.ps1`)

#### Prova 1: Ordenação de Nodes
**Comando:**
`pwsh -File scripts/mutate.ps1 -Path internal/service/graph.go -Anchor 'return cmp.Compare(a.Path, b.Path)' -Replacement 'return 0' -Test TestLinkGraphOrdemEstavel -Package ./internal/service/`

**Saída literal:**
```
[...] Mutando internal/service/graph.go
      - return cmp.Compare(a.Path, b.Path)
      + return 0

[...] go test -race -run TestLinkGraphOrdemEstavel ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestLinkGraphOrdemEstavel (0.05s)
    graph_ordem_test.go:76: volta 1 devolveu ordem diferente:
...
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.188s
FAIL
----------------------------------------------------------------------
[OK] internal/service/graph.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```
**EXIT:** 0
**Asserção que falhou:** `graph_ordem_test.go:76: volta 1 devolveu ordem diferente`

---

#### Prova 2: Teto de Limit
**Comando:**
`pwsh -File scripts/mutate.ps1 -Path internal/service/graph.go -Anchor 'if limit > 500 {' -Replacement 'if false {' -Test TestLinkGraphLimitTemTeto -Package ./internal/service/`

**Saída literal:**
```
[...] Mutando internal/service/graph.go
      - if limit > 500 {
      + if false {

[...] go test -race -run TestLinkGraphLimitTemTeto ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestLinkGraphLimitTemTeto (0.02s)
    graph_ordem_test.go:83: LimitEfetivo = 1000000: limit continua sem teto
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.915s
FAIL
----------------------------------------------------------------------
[OK] internal/service/graph.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```
**EXIT:** 0
**Asserção que falhou:** `graph_ordem_test.go:83: LimitEfetivo = 1000000: limit continua sem teto`

---

### 4. Saída de `verify.ps1`

**Contagem de Passos:** 13 passos executados.
- Passos 1 a 10: `go build`, `go test -race`, `go test (latência)`, `go vet (windows/linux/darwin)`, `gofmt`, `golangci-lint (windows/linux)`, `check_net (RNF-30)` — **TODOS [OK]**
- Passo 11: `check_tool_params` — **REPROVADO** (`[!] 2 parametro(s) declarado(s) e nunca lido(s): tagListInput.Sort, tagListInput.Hierarchical` — herdado da Task 120 como documentado)
- Passos 12 e 13: `check_doc_refs`, `check_readme_anchors` — **TODOS [OK]**

**Código de saída:** `1` (causado exclusivamente pelo passo 11 `check_tool_params`, que é a dívida herdada da Task 120).

---

### 5. Teste de Estabilidade (`-count=5`)

**Comando:**
`go test -race -v -count=5 ./internal/service/ -run TestLinkGraphOrdemEstavel`

**Saída literal:**
```
=== RUN   TestLinkGraphOrdemEstavel
--- PASS: TestLinkGraphOrdemEstavel (0.08s)
=== RUN   TestLinkGraphOrdemEstavel
--- PASS: TestLinkGraphOrdemEstavel (0.07s)
=== RUN   TestLinkGraphOrdemEstavel
--- PASS: TestLinkGraphOrdemEstavel (0.07s)
=== RUN   TestLinkGraphOrdemEstavel
--- PASS: TestLinkGraphOrdemEstavel (0.07s)
=== RUN   TestLinkGraphOrdemEstavel
--- PASS: TestLinkGraphOrdemEstavel (0.08s)
PASS
ok  	github.com/jonyd/gobsidian/internal/service	2.841s
```

---

### 6. O que ficou de fora

Nenhum item ficou de fora. A Task 122 foi totalmente implementada e verificada conforme o brief e as regras do plano.

---

### 7. `git status --porcelain`

```
M  .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
?? .superpowers/sdd/task-122-report.md
```
