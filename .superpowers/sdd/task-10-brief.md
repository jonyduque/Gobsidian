### Task 10: `gobsidian doctor`

**Files:**
- Create: `internal/doctor/doctor.go`, `internal/doctor/checks.go`, `internal/doctor/checks_windows.go`, `internal/doctor/checks_other.go`
- Create: `internal/doctor/doctor_test.go`
- Create: `cmd/gobsidian/doctor.go`

**Interfaces:**
- Consumes: `vault.Vault`, `vault.IsCloudOnly`, `config.Config`
- Produces: `doctor.Result{Name string, Status Status, Detail string}`; `doctor.Status` com `StatusOK`, `StatusWarn`, `StatusFail`; `doctor.Run(ctx context.Context, cfg config.Config) []Result`; `doctor.ExitCode([]Result) int`

- [ ] **Step 1: Escrever o teste**

`internal/doctor/doctor_test.go`:

```go
package doctor_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/doctor"
)

func TestRunFlagsMissingVault(t *testing.T) {
	cfg := config.Defaults()
	cfg.VaultPath = filepath.Join(t.TempDir(), "nao-existe")

	results := doctor.Run(context.Background(), cfg)

	if doctor.ExitCode(results) == 0 {
		t.Fatal("doctor deveria falhar com raiz inexistente")
	}
	if !hasFailure(results, "raiz do cofre") {
		t.Errorf("nenhuma verificacao de raiz falhou: %+v", results)
	}
}

func TestRunWarnsWithoutObsidianDir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "A.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := config.Defaults()
	cfg.VaultPath = root

	results := doctor.Run(context.Background(), cfg)

	// Ausencia de .obsidian/ e aviso, nao falha: o produto funciona sobre
	// qualquer pasta de Markdown, e forcar a presenca seria arbitrario.
	if doctor.ExitCode(results) != 0 {
		t.Errorf("doctor falhou em cofre valido sem .obsidian: %+v", results)
	}
	if !hasStatus(results, ".obsidian", doctor.StatusWarn) {
		t.Errorf("esperava aviso sobre .obsidian ausente: %+v", results)
	}
}

func hasFailure(results []doctor.Result, substr string) bool {
	return hasStatus(results, substr, doctor.StatusFail)
}

func hasStatus(results []doctor.Result, substr string, want doctor.Status) bool {
	for _, r := range results {
		if r.Status == want && contains(r.Name, substr) {
			return true
		}
	}
	return false
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || len(haystack) >= len(needle) &&
		(haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/doctor/ -v`
Esperado: FAIL — `undefined: doctor.Run`.

- [ ] **Step 3: Implementar**

`internal/doctor/doctor.go`:

```go
// Package doctor verifica o ambiente antes que o resto do produto tente
// funcionar nele. E o primeiro comando a rodar quando algo nao funciona.
package doctor

import (
	"context"

	"github.com/jonyd/gobsidian/internal/config"
)

type Status int

const (
	StatusOK Status = iota
	StatusWarn
	StatusFail
)

func (s Status) Marker() string {
	switch s {
	case StatusOK:
		return "[OK]"
	case StatusWarn:
		return "[!]"
	default:
		return "[!]"
	}
}

type Result struct {
	Name   string
	Status Status
	Detail string
}

type check func(context.Context, config.Config) Result

// Run executa todas as verificacoes na ordem em que importam: as que
// invalidam as seguintes vem primeiro.
func Run(ctx context.Context, cfg config.Config) []Result {
	checks := []check{
		checkRootExists,
		checkReadable,
		checkWritable,
		checkObsidianDir,
		checkNoteCount,
		checkLongestPath,
		checkCacheDir,
		checkFreeSpace,
	}
	checks = append(checks, platformChecks()...)

	out := make([]Result, 0, len(checks))
	for _, c := range checks {
		res := c(ctx, cfg)
		out = append(out, res)

		// Sem raiz acessivel, todas as verificacoes seguintes reportariam
		// falhas derivadas e o relatorio viraria ruido.
		if res.Status == StatusFail && res.Name == "raiz do cofre existe" {
			break
		}
	}
	return out
}

// ExitCode e zero quando nao ha falha bloqueante. Avisos nao alteram o codigo:
// um cofre com arquivos somente-nuvem funciona, apenas de forma incompleta.
func ExitCode(results []Result) int {
	for _, r := range results {
		if r.Status == StatusFail {
			return 1
		}
	}
	return 0
}
```

`internal/doctor/checks.go` implementa as oito funções acima. Cada uma devolve um `Result` com `Name` estável e `Detail` acionável. Regras de status:

| Verificação | `Name` | Falha quando | Aviso quando |
|---|---|---|---|
| Raiz existe e é diretório | `raiz do cofre existe` | Não existe, ou não é diretório | — |
| Leitura na raiz | `permissao de leitura` | `os.ReadDir` falha | — |
| Escrita na raiz | `permissao de escrita` | Criar e apagar um temporário falha, e `--read-only` está desligado | Falha com `--read-only` ligado |
| `.obsidian/` presente | `.obsidian presente` | Nunca | Ausente |
| Contagem de notas | `contagem de notas` | Nunca | Zero notas |
| Caminho mais longo | `comprimento de caminho` | Nunca | Acima de 240 caracteres |
| Diretório de cache | `diretorio de cache` | Nunca | Não é criável |
| Espaço livre | `espaco em disco` | Menos de 10 MB livres | Menos de 100 MB |

`internal/doctor/checks_windows.go` adiciona, via `platformChecks()`:

| Verificação | `Name` | Falha quando | Aviso quando |
|---|---|---|---|
| Caminhos longos no registro | `caminhos longos habilitados` | Nunca | `LongPathsEnabled != 1` **e** existe caminho acima de 240 |
| Arquivos somente-nuvem | `arquivos somente-nuvem` | Nunca | Qualquer nota com `IsCloudOnly` verdadeiro |
| Colisões de casing | `colisoes de casing` | Nunca | Duas notas com o mesmo caminho em minúsculas |

`internal/doctor/checks_other.go` devolve `platformChecks()` vazio, atrás de `//go:build !windows`.

- [ ] **Step 4: Implementar o subcomando**

`cmd/gobsidian/doctor.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var flags config.Flags

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostica o ambiente: permissoes, OneDrive, MAX_PATH, casing",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Sem esta linha, --read-only nao chega a Config e a verificacao
			// de permissao de escrita roda mesmo quando o usuario pediu para
			// nao rodar. Toda chamada a config.Load precisa preencher os
			// companheiros das flags que o comando expoe.
			flags.ReadOnlySet = cmd.Flags().Changed("read-only")

			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}

			results := doctor.Run(cmd.Context(), cfg)

			// doctor imprime em stdout de proposito: e um comando de CLI,
			// nao um servidor. Nenhum JSON-RPC trafega aqui.
			out := cmd.OutOrStdout()
			for _, r := range results {
				fmt.Fprintf(out, "%s %s\n", r.Status.Marker(), r.Name)
				if r.Detail != "" {
					fmt.Fprintf(out, "     %s\n", r.Detail)
				}
			}

			code := doctor.ExitCode(results)
			if code != 0 {
				fmt.Fprintln(out, "[!] Ha falhas bloqueantes acima")
				os.Exit(code)
			}
			fmt.Fprintln(out, "[OK] Ambiente apto")
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "nao verifica permissao de escrita")

	return cmd
}
```

Saída em ASCII puro. Nada de emoji: em console PowerShell com página de código CP-850 ou CP-1252, caracteres fora de ASCII produzem saída ilegível.

- [ ] **Step 5: Rodar para confirmar que passa**

Run: `go test -race ./internal/doctor/ -v`
Esperado: PASS, dois testes.

Verificação manual: `go run ./cmd/gobsidian doctor --vault "C:\caminho\do\cofre"` deve produzir uma linha por verificação e sair com código 0.

- [ ] **Step 6: Commit**

```bash
git add internal/doctor cmd/gobsidian
git commit -m "feat(doctor): environment diagnostics with ASCII report and blocking exit code"
```

---

