### Task 1: Bootstrap do módulo e do CI

**Files:**
- Create: `go.mod`, `.gitignore`, `.golangci.yml`, `LICENSE`, `Makefile`
- Create: `.github/workflows/ci.yml`
- Create: `tools/netcheck/netcheck.go`, `tools/netcheck/netcheck_test.go`
- Create: `scripts/check_net.ps1`

**Interfaces:**
- Consumes: nada
- Produces: módulo `github.com/jonyd/gobsidian` compilável; `make test` e `make lint` funcionais; analisador `netcheck.Analyzer` usável em `go vet -vettool`

- [ ] **Step 1: Inicializar o módulo e fixar dependências**

```powershell
cd C:\Users\jonyd\Projetos\Gobsidian
git init
go mod init github.com/jonyd/gobsidian

go get github.com/modelcontextprotocol/go-sdk@v1.5.0
go get github.com/yuin/goldmark
go get github.com/fsnotify/fsnotify
go get github.com/cespare/xxhash/v2
go get github.com/spf13/cobra
go get golang.org/x/text
go get golang.org/x/sys
go get gopkg.in/yaml.v3
go get golang.org/x/tools/go/analysis

go mod tidy
```

Esperado: `go.mod` com `go 1.24` e as oito dependências. Se `v1.5.0` do SDK não resolver, pare e reporte — não caia para `@latest`, a versão está fixada por decisão (PRD D6).

- [ ] **Step 2: Escrever o teste do analisador de rede**

`tools/netcheck/netcheck_test.go`:

```go
package netcheck_test

import (
	"testing"

	"github.com/jonyd/gobsidian/tools/netcheck"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestNetCheck(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), netcheck.Analyzer, "a")
}
```

`tools/netcheck/testdata/src/a/a.go`:

```go
package a

import (
	"net/http" // want `pacote de rede proibido: net/http`
	"os"
)

var _ = os.Getenv
var _ = http.Get
```

- [ ] **Step 3: Rodar o teste para confirmar que falha**

Run: `go test ./tools/netcheck/ -run TestNetCheck -v`
Esperado: FAIL — `undefined: netcheck.Analyzer`.

- [ ] **Step 4: Implementar o analisador**

`tools/netcheck/netcheck.go`:

```go
// Package netcheck implementa a verificação de RNF-30: nenhum pacote do
// produto importa rede. Ver PRD §6.4 para a formulação completa da garantia.
package netcheck

import (
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "netcheck",
	Doc:  "reporta importacao de pacotes de rede em codigo do produto",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if !isNetwork(path) {
				continue
			}
			pass.Reportf(imp.Pos(), "pacote de rede proibido: %s", path)
		}
	}
	return nil, nil
}

func isNetwork(path string) bool {
	return path == "net" || strings.HasPrefix(path, "net/")
}
```

- [ ] **Step 5: Rodar o teste para confirmar que passa**

Run: `go test ./tools/netcheck/ -run TestNetCheck -v`
Esperado: PASS.

- [ ] **Step 6: Escrever o script de verificação e o CI**

`scripts/check_net.ps1` — copiar integralmente o script de `docs/ESTRUTURA.md` §4 (bloco "Verificar que nenhum pacote NOSSO importa rede"), precedido de:

```powershell
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
```

`.github/workflows/ci.yml`:

```yaml
name: ci

on:
  push:
  pull_request:

jobs:
  test:
    strategy:
      fail-fast: false
      matrix:
        os: [windows-latest, ubuntu-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go vet ./...
      - run: go test -race ./...

  netcheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: nenhum pacote do produto importa rede
        shell: pwsh
        run: ./scripts/check_net.ps1

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - uses: golangci/golangci-lint-action@v6
```

`.golangci.yml`:

```yaml
version: "2"
linters:
  enable:
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
    - bodyclose
    - contextcheck
    - errorlint
    - gocritic
    - revive
```

`Makefile`:

```makefile
.PHONY: test lint build bench netcheck

test:
	go test -race ./...

lint:
	golangci-lint run

netcheck:
	pwsh -NoProfile -File scripts/check_net.ps1

build:
	pwsh -NoProfile -File scripts/build.ps1

bench:
	go test -bench=. -benchmem ./internal/index ./internal/search ./internal/parser
```

`.gitignore`:

```
bin/
coverage.out
*.exe
cofre-debug.log
gobsidian-debug.log
```

`LICENSE`: texto MIT, titular `jonyd`, ano 2026.

- [ ] **Step 7: Verificar que tudo roda**

Run: `go vet ./... && go test -race ./...`
Esperado: sem erros; `netcheck` passa; nenhum outro pacote existe ainda.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "chore: bootstrap module, CI, and network-import analyzer"
```

---

