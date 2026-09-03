### Task 176: `index` e `inspect` abrem pelo cache; flags de cofre registradas uma vez

**Files:**
- Modify: `cmd/gobsidian/index.go:41-52` (`boot.AbrirIndice`), `indexSummaryJSON` (campo `Origin`), console (`Origem:`), flags
- Modify: `cmd/gobsidian/inspect.go:37-51` (`boot.AbrirIndice`), flags
- Create: `cmd/gobsidian/flags.go` (`flagsDeCofre`, `flagsDeCache`)
- Modify: `cmd/gobsidian/search.go`, `serve.go:47-51`, `daemon.go:56-60`, `doctor.go:67-71` (usam `flagsDeCofre`/`flagsDeCache` em vez de registrar `--vault`, `--follow-symlinks`, `--cache-dir`, `--log-level` à mão)
- Modify: `cmd/gobsidian/cli_subcommands_test.go` (`TestIndexCmd_StdoutAndJSON:30`, `TestInspectCmd_StdoutAndJSON:119` levam `--cache-dir`)
- Create: `cmd/gobsidian/index_origem_test.go`
- Modify: `README.md` (flags de `index`/`inspect`; campo `origin` do `index --json`), `docs/OPERACAO.md` (idem)

**Interfaces:**
- Consumes: `boot.AbrirIndice` (Task 174), `loggerDeCLI` e `Config.LogLevelExplicito` (Task 175). Task 156 já removeu `--read-only/--debounce-ms/--max-results` de `index`/`inspect`.
- Produces: `indexSummaryJSON.Origin string json:"origin"` ("build" | "cache"); `func flagsDeCofre(cmd *cobra.Command, f *config.Flags)` e `func flagsDeCache(cmd *cobra.Command, f *config.Flags)`.

- [ ] **Step 1: Teste da origem — falha porque o campo não existe**

`cmd/gobsidian/index_origem_test.go`:

```go
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestIndexCmdJSONTrazOrigem(t *testing.T) {
	cofre := t.TempDir()
	if err := os.WriteFile(filepath.Join(cofre, "n.md"), []byte("# N\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	roda := func() string {
		var stdout, stderr bytes.Buffer
		cmd := newIndexCmd()
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"--vault", cofre, "--cache-dir", cache, "--json"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("index: %v\nstderr:\n%s", err, stderr.String())
		}
		var out struct {
			Origin string `json:"origin"`
			Notes  int    `json:"notes"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
			t.Fatalf("JSON invalido: %v\n%s", err, stdout.String())
		}
		if out.Notes != 1 {
			t.Fatalf("notes = %d, quero 1", out.Notes)
		}
		return out.Origin
	}
	if o := roda(); o != "build" {
		t.Fatalf("primeira execucao: origin = %q, quero \"build\"", o)
	}
	if o := roda(); o != "cache" {
		t.Fatalf("segunda execucao: origin = %q, quero \"cache\"", o)
	}
}
```

`newIndexCmd` e a chave JSON `notes` são as reais de `index.go` (conferir `indexSummaryJSON`; se a tag for outra, usar a outra). Run: `go test ./cmd/gobsidian/ -run TestIndexCmdJSONTrazOrigem` — Expected: FAIL (`origin` vazio).

- [ ] **Step 2: `index` e `inspect` via `boot.AbrirIndice`**

`index.go:41-52`:

```go
			log := loggerDeCLI(cmd, cfg)
			start := time.Now()
			idx, origem, err := boot.AbrirIndice(cmd.Context(), v, cfg, log)
			if err != nil {
				return err
			}
			dur := time.Since(start)
```

`indexSummaryJSON` ganha `Origin string \`json:"origin"\`` preenchido com `origem`; na saída de console, uma linha `con.Item("Origem: %s", origem)` junto das demais. `inspect.go:37-51`: mesma troca (`idx, _, err := boot.AbrirIndice(...)`).

Run: `go test ./cmd/gobsidian/ -run TestIndexCmdJSONTrazOrigem` — Expected: PASS.

- [ ] **Step 3: Flags registradas uma vez**

`cmd/gobsidian/flags.go`:

```go
package main

import (
	"github.com/spf13/cobra"

	"github.com/jonyd/gobsidian/internal/config"
)

// flagsDeCofre registra as flags que todo subcomando que abre um cofre
// aceita. Seis arquivos registravam --vault e --follow-symlinks com o mesmo
// texto; quando o texto mudou, mudou em cinco.
func flagsDeCofre(cmd *cobra.Command, f *config.Flags) {
	cmd.Flags().StringVar(&f.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
	cmd.Flags().BoolVar(&f.FollowSymlinks, "follow-symlinks", false,
		"segue symlink dentro do cofre; o padrao recusa, porque o confinamento nao alcanca o alvo")
}

// flagsDeCache registra as flags de quem le ou grava o cache de indice.
func flagsDeCache(cmd *cobra.Command, f *config.Flags) {
	cmd.Flags().StringVar(&f.CacheDir, "cache-dir", "", "diretorio do cache de indice")
	cmd.Flags().StringVar(&f.LogLevel, "log-level", "", "debug, info, warn ou error")
}
```

Trocar os registros em `search.go`, `index.go`, `inspect.go`, `serve.go`, `daemon.go`, `doctor.go` (`doctor` só `flagsDeCofre`). O texto de ajuda é o de `serve.go` hoje; se algum arquivo tinha texto diferente, o de `serve` vence e a diferença vai para o relatório.

Run: `go build ./... && go vet ./...`; `grep -rn '"vault", ""' cmd/gobsidian/*.go` — Expected: só `flags.go`.
Run: `go test -race ./cmd/gobsidian/` — Expected: PASS (`TestSubcommands_FlagsSetPopulated:168` inclusive; se ele testava flags que a Task 156 apagou, já foi ajustado lá).

- [ ] **Step 4: `--cache-dir` nos testes de CLI existentes**

`TestIndexCmd_StdoutAndJSON:30` e `TestInspectCmd_StdoutAndJSON:119`: acrescentar `"--cache-dir", t.TempDir()` a cada `SetArgs`. Motivo idêntico ao da Task 175 — sem isso o teste grava no cache real do usuário.

- [ ] **Step 5: Prova de mutação**

```powershell
pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/index.go -Anchor 'Origin: origem,' -Replacement 'Origin: "build",' -Test TestIndexCmdJSONTrazOrigem -Package ./cmd/gobsidian/
```

Expected: exit 0 (segunda execução falha: `origin = "build", quero "cache"`). Se o literal da âncora não bater com o código escrito, ajustar a âncora ao texto real — a mutação é "origem fixa", não a grafia.

- [ ] **Step 6: Documentação**

`README.md`: flags de `index` e `inspect` ganham `--cache-dir`/`--log-level`; `index --json` documenta `origin` ("build" na primeira execução, "cache" quando o cache está fresco). `docs/OPERACAO.md`: idem, na seção de `index`.

- [ ] **Step 7: Gate e dois commits**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add cmd/gobsidian/index.go cmd/gobsidian/inspect.go cmd/gobsidian/index_origem_test.go cmd/gobsidian/cli_subcommands_test.go README.md docs/OPERACAO.md
git commit -m "feat(cli): index and inspect load from the cache"
git add cmd/gobsidian/flags.go cmd/gobsidian/search.go cmd/gobsidian/index.go cmd/gobsidian/inspect.go cmd/gobsidian/serve.go cmd/gobsidian/daemon.go cmd/gobsidian/doctor.go
git commit -m "refactor(cli): register shared vault flags once"
```

O segundo commit é estrutural puro; se o primeiro já tiver tocado flags em `index.go`/`inspect.go` (registrar `--cache-dir` à mão para o teste passar), tudo bem — o segundo substitui pelo registro comum.

#### Verificações

- Step 1 FAIL→PASS colado; `grep` do Step 3 colado.
- `mutate.ps1` exit 0 colado.
- `git log --oneline -2` mostra os dois commits na ordem.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Todo teste de CLI com `--vault` leva `--cache-dir` de `t.TempDir()`.
- `doctor` não ganha `flagsDeCache`: não lê cache.
- Estrutura (commit 2) separada de comportamento (commit 1).

#### Comando de mutação

Step 5 (`mutate.ps1`, âncora `Origin: origem,` em `cmd/gobsidian/index.go`).

#### Contrato de relatório

`task-176-report.md`: status, dois SHAs, saídas dos Steps 1, 3, 5, última linha do `verify.ps1`, e a lista de textos de ajuda que divergiam (se houver).

---

