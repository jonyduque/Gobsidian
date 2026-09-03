### Task 175: `search` de CLI abre índice e busca como o `serve` — e ganha `--cache-dir`/`--log-level`

**Files:**
- Modify: `cmd/gobsidian/search.go:35-49` (troca `index.New()+Build` + laço `inv.Update` por `boot.AbrirIndice` + `boot.PrepararBusca`), `:89-97` (flags `--cache-dir`, `--log-level`)
- Create: `cmd/gobsidian/cli_log.go` (`loggerDeCLI`)
- Modify: `internal/config/config.go:29,:55,:82-95` (`Config.LogLevelExplicito bool`)
- Modify: `cmd/gobsidian/cli_subcommands_test.go` (`TestSearchCmd_StdoutAndJSON:77` passa `--cache-dir` de `t.TempDir()`)
- Create: `cmd/gobsidian/search_cache_test.go` (`TestSearchCLISegundaExecucaoUsaCache`)
- Modify: `README.md` (tabela de flags de `search`), `docs/ESTADO.md` e `docs/OPERACAO.md` (medição M4 depois)

**Interfaces:**
- Consumes: `boot.AbrirIndice(ctx, v, cfg, log) (*index.Index, string, error)`, `boot.PrepararBusca(ctx, v, idx, inv, cfg, log)` (Task 174); `service.New(v, idx *index.Index, inv, nil, opts)` (Task 173); Task 156 já removeu `--read-only`/`--debounce-ms` de `search` e manteve `--max-results`.
- Produces: `func loggerDeCLI(cmd *cobra.Command, cfg config.Config) *slog.Logger` em `cmd/gobsidian/cli_log.go`; `config.Config.LogLevelExplicito bool`. Task 176 usa os dois.

- [ ] **Step 1: `LogLevelExplicito` em `config`**

`internal/config/config.go`, no `Config` (`:55`): campo `LogLevelExplicito bool` com o comentário `// LogLevelExplicito e true quando GOBSIDIAN_LOG_LEVEL ou --log-level foi dado; os subcomandos de CLI so mostram log acima de Warn sem ele.` Em `Load` (`:82-95`), nos dois ramos que atribuem `cfg.LogLevel`, também `cfg.LogLevelExplicito = true`.

Teste em `internal/config/config_test.go` (acrescentar ao arquivo existente):

```go
func TestLoadMarcaLogLevelExplicito(t *testing.T) {
	t.Setenv("GOBSIDIAN_LOG_LEVEL", "")
	cfg, err := Load(Flags{VaultPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogLevelExplicito {
		t.Fatal("sem env e sem flag, LogLevelExplicito devia ser false")
	}
	cfg, err = Load(Flags{VaultPath: t.TempDir(), LogLevel: "debug"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.LogLevelExplicito || cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("flag dada: Explicito=%v LogLevel=%v", cfg.LogLevelExplicito, cfg.LogLevel)
	}
	t.Setenv("GOBSIDIAN_LOG_LEVEL", "warn")
	cfg, err = Load(Flags{VaultPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.LogLevelExplicito || cfg.LogLevel != slog.LevelWarn {
		t.Fatalf("env dada: Explicito=%v LogLevel=%v", cfg.LogLevelExplicito, cfg.LogLevel)
	}
}
```

Run: `go test ./internal/config/ -run TestLoadMarcaLogLevelExplicito -v` — Expected: FAIL (campo não existe) e depois PASS.

- [ ] **Step 2: `loggerDeCLI`**

`cmd/gobsidian/cli_log.go`:

```go
package main

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/jonyd/gobsidian/internal/config"
)

// loggerDeCLI e o logger dos subcomandos de CLI (search, index, inspect):
// escreve em cmd.ErrOrStderr() para que teste capture, e cala tudo abaixo de
// Warn a menos que o operador tenha pedido um nivel — um subcomando que
// imprime "servidor pronto" a cada chamada polui o terminal de quem so
// queria o resultado.
func loggerDeCLI(cmd *cobra.Command, cfg config.Config) *slog.Logger {
	nivel := slog.LevelWarn
	if cfg.LogLevelExplicito {
		nivel = cfg.LogLevel
	}
	return slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: nivel}))
}
```

- [ ] **Step 3: `search` usa `boot`**

`cmd/gobsidian/search.go`, substituir `:44-49` (o `index.New()`, o `Build` e o laço `inv.Update`) por:

```go
			log := loggerDeCLI(cmd, cfg)
			idx, _, err := boot.AbrirIndice(cmd.Context(), v, cfg, log)
			if err != nil {
				return err
			}
			inv := search.NewInverted()
			inv.MarkBuilding()
			boot.PrepararBusca(cmd.Context(), v, idx, inv, cfg, log)
			defer inv.Close()
```

Flags (`:89-97`): acrescentar

```go
	cmd.Flags().StringVar(&flags.CacheDir, "cache-dir", "", "diretorio do cache de indice")
	cmd.Flags().StringVar(&flags.LogLevel, "log-level", "", "debug, info, warn ou error")
```

Remover o import de `index` se ficar sem uso. `go build ./... && go vet ./...` — Expected: limpo.

- [ ] **Step 4: Testes**

`cli_subcommands_test.go:77 TestSearchCmd_StdoutAndJSON`: acrescentar `"--cache-dir", t.TempDir()` aos `SetArgs` das duas execuções. Sem isso o teste grava em `os.UserCacheDir()/gobsidian/<chave>` — o cache real do usuário para um cofre de teste. A asserção "stderr vazio" continua: `loggerDeCLI` cala Info sem `--log-level`.

`cmd/gobsidian/search_cache_test.go`:

```go
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchCLISegundaExecucaoUsaCache(t *testing.T) {
	cofre := t.TempDir()
	if err := os.WriteFile(filepath.Join(cofre, "n.md"), []byte("# N\n\nexecucao do teste\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	roda := func() string {
		var stdout, stderr bytes.Buffer
		cmd := newSearchCmd()
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"--vault", cofre, "--cache-dir", cache, "--log-level", "info", "--json", "execucao"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("search: %v\nstderr:\n%s", err, stderr.String())
		}
		if !strings.Contains(stdout.String(), `"n.md"`) {
			t.Fatalf("stdout sem o resultado esperado:\n%s", stdout.String())
		}
		return stderr.String()
	}
	primeira := roda()
	if !strings.Contains(primeira, "origem=construcao") {
		t.Fatalf("primeira execucao devia construir o indice de busca:\n%s", primeira)
	}
	segunda := roda()
	if !strings.Contains(segunda, "origem=cache") {
		t.Fatalf("segunda execucao devia adotar o cache de busca:\n%s", segunda)
	}
}
```

`newSearchCmd` é o nome real do construtor em `search.go` (conferir; se for `searchCmd()` ou outro, usar o real). O literal `"n.md"` supõe que o JSON serializa `Path` como caminho canônico relativo — `TestSearchCmd_StdoutAndJSON` já prende esse formato; copiar dele a asserção se for diferente.

Run: `go test -race ./cmd/gobsidian/ -run 'TestSearch' -v` — Expected: PASS nos dois.

- [ ] **Step 5: Prova manual de mutação**

Substituir temporariamente o bloco do Step 3 pelo antigo (`index.New()` + `Build` + laço `inv.Update` + `inv.MarkReady()`), rodar `go test ./cmd/gobsidian/ -run TestSearchCLISegundaExecucaoUsaCache` — Expected: FAIL ("primeira execucao devia construir…" ou "segunda execucao devia adotar…": sem `PrepararBusca` nenhuma linha `origem=` aparece). Restaurar. Colar.

- [ ] **Step 6: Medição M4 — o objetivo da Task**

Binário `antes`: `scripts/build.ps1` no commit da Task 174 (o `search` ainda constrói tudo) → `gobsidian_antes.exe`. Binário `depois`: build deste commit.

Para cada binário, com o cofre `vault_5000` (Baseline) e `--cache-dir` num diretório novo:
- 3 execuções **frias** (apagar o diretório de cache antes de cada uma): `Measure-Command { .\gobsidian_X.exe search --json --limit 200 --vault <vault_5000> --cache-dir <dir> "execucao" }`.
- 3 execuções **quentes** (mesmo diretório, cache já gravado).

Expected: `antes` fria ≈ quente ≈ M4 (10 520 / 10 819 / 11 046 ms); `depois` fria ≈ M4 + custo de gravar os dois caches; `depois` quente perto do `serve` quente de M4 (wall 475–528 ms). Colar as 12 linhas em `docs/ESTADO.md` (medições publicadas) e resumir em `docs/OPERACAO.md` na seção do `search`. Se `depois` quente não ficar abaixo de 1/5 da fria, a Task não entregou o que promete — reportar `DONE_WITH_CONCERNS` com os números.

- [ ] **Step 7: README**

Tabela de flags de `search` (`README.md:196-206`, já editada pela Task 156): acrescentar `--cache-dir` e `--log-level` com a mesma descrição das flags de `serve`; nota de uma linha: "`search` reaproveita o cache de índice e de busca do `serve`; a primeira execução constrói e grava, as seguintes carregam."

- [ ] **Step 8: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add cmd/gobsidian/search.go cmd/gobsidian/cli_log.go cmd/gobsidian/search_cache_test.go cmd/gobsidian/cli_subcommands_test.go internal/config/config.go internal/config/config_test.go README.md docs/ESTADO.md docs/OPERACAO.md
git commit -m "feat(cli): search loads the index and the search cache like serve does"
```

#### Verificações

- Step 1 FAIL→PASS colado; Step 4 PASS colado; Step 5 FAIL colado.
- 12 linhas de medição do Step 6 coladas e publicadas.
- `grep -n "cache-dir" cmd/gobsidian/cli_subcommands_test.go` mostra o `search` com `--cache-dir`.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nenhum teste de CLI grava fora de `t.TempDir()`: todo `SetArgs` com `--vault` de teste leva `--cache-dir`.
- `search` continua CLI: stdout é do resultado, stderr do log; o comentário existente em `search.go` sobre stdout permanece.
- Não rodar `test_orphans.ps1` em paralelo com a medição do Step 6.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: a prova é a substituição manual do Step 5 com o FAIL colado.

#### Contrato de relatório

`task-175-report.md`: status, SHA, saídas dos Steps 1, 4, 5, 6 (as 12 linhas), última linha do `verify.ps1`.

---

