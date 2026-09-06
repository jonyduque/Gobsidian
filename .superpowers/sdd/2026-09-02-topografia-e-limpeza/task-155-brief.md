### Task 155: `ipc.EhDesconexaoLimpa` é a conta única de "o outro lado foi embora" (1.10)

Três lugares decidem se um erro de transporte é encerramento normal:
`cmd/gobsidian/serve.go:73-84` (`shutdownExitCode`: `Canceled`, `EOF`,
`ErrClosedPipe`), `cmd/gobsidian/ponte.go:249-253` (os três + `os.ErrClosed`)
e `internal/daemon/daemon.go:254-258` (os quatro). `shutdownExitCode` não
conhece `os.ErrClosed` — um `serve` em processo que encerra por stdin fechado
pelo SDK com `ErrClosed` sai com código 1 e o host loga falha.

**Files:**
- Create: `internal/ipc/desconexao.go`
- Test: `internal/ipc/desconexao_test.go`
- Modify: `cmd/gobsidian/serve.go:73-84`, `cmd/gobsidian/ponte.go:249-253`, `internal/daemon/daemon.go:254-258`
- Modify: `cmd/gobsidian/serve_test.go:236-246` (acrescentar caso `os.ErrClosed`)

**Interfaces:**
- Produces: `func EhDesconexaoLimpa(err error) bool` em `internal/ipc`. `cmd` e `daemon` já importam `ipc`; nenhuma aresta nova.

- [ ] **Step 1: Teste em `ipc`**

```go
package ipc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestEhDesconexaoLimpaReconheceAsQuatroFormas(t *testing.T) {
	limpos := []error{nil, context.Canceled, io.EOF, io.ErrClosedPipe, os.ErrClosed,
		fmt.Errorf("copiando: %w", os.ErrClosed)}
	for _, e := range limpos {
		if !EhDesconexaoLimpa(e) {
			t.Errorf("EhDesconexaoLimpa(%v) = false", e)
		}
	}
	sujos := []error{errors.New("falha real"), context.DeadlineExceeded, io.ErrUnexpectedEOF}
	for _, e := range sujos {
		if EhDesconexaoLimpa(e) {
			t.Errorf("EhDesconexaoLimpa(%v) = true", e)
		}
	}
}
```

`DeadlineExceeded` fica de fora de propósito: prazo estourado é falha, não o
outro lado indo embora.

- [ ] **Step 2: `desconexao.go`**

```go
package ipc

import (
	"context"
	"errors"
	"io"
	"os"
)

// EhDesconexaoLimpa diz se um erro devolvido por um loop de transporte
// significa "o outro lado foi embora", e nao falha.
//
// context.Canceled vem do proprio lifecycle; io.EOF e io.ErrClosedPipe sao
// como o SDK reporta o fim do stdin; os.ErrClosed e como o fechamento de um
// pipe ou conn aparece do lado que ainda estava copiando. Tres lugares
// tinham essa lista, e um deles (shutdownExitCode) nao tinha os.ErrClosed —
// um serve que encerrava por essa via saia com codigo 1.
func EhDesconexaoLimpa(err error) bool {
	return err == nil ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, os.ErrClosed)
}
```

- [ ] **Step 3: Os três consumidores**

`serve.go`:

```go
func shutdownExitCode(err error) int {
	if ipc.EhDesconexaoLimpa(err) {
		return 0
	}
	return 1
}
```

`ponte.go:249-253` → `if !ipc.EhDesconexaoLimpa(loopErr) { return loopErr }` — o comentário de `:242-248` fica, com "ver ipc.EhDesconexaoLimpa" no lugar da lista.
`daemon.go:254-258` → `if !ipc.EhDesconexaoLimpa(err) { d.log.Warn(...) }`.
Tire os imports que ficarem sem uso (`io`, `os`, `context` conforme o arquivo).

- [ ] **Step 4: Caso novo em `serve_test.go`**: `{"os.ErrClosed", os.ErrClosed, 0}` e `{"prazo estourado", context.DeadlineExceeded, 1}`. Rodar: `go test ./cmd/gobsidian ./internal/ipc ./internal/daemon -run 'shutdownExitCode|Desconexao' -v`.

- [ ] **Step 5: Os quatro cenários de encerramento**

Run: `pwsh -File scripts/test_orphans.ps1`
Expected: os quatro `[OK]`. Cole a saída no relatório.

- [ ] **Step 6: Gate, commit**

```bash
git add internal/ipc/desconexao.go internal/ipc/desconexao_test.go cmd/gobsidian/serve.go cmd/gobsidian/ponte.go internal/daemon/daemon.go cmd/gobsidian/serve_test.go
git commit -m "fix(ipc): one account of a clean disconnect, and serve exits 0 on os.ErrClosed like the bridge does"
```

#### Verificações
Além dos passos:
1. Os três consumidores (`serve.go`, `ponte.go`, `daemon/daemon.go`) chamam `ipc.EhDesconexaoLimpa`; `grep -rn "io.ErrClosedPipe" cmd internal` só encontra `internal/ipc/desconexao.go` e testes.
2. `pwsh -File scripts/test_orphans.ps1` — os quatro cenários `[OK]`, saída colada. **Não rode em paralelo com nenhuma medição.**
3. `go list -f '{{.Imports}}' ./cmd/gobsidian ./internal/daemon` — `ipc` já era importado pelos dois; nenhuma aresta nova.

#### Regras de execução
- Gate: `pwsh -File scripts/verify.ps1` verde antes do commit, com a contagem de etapas colada no relatório. `-SkipCross -SkipNet` só para iterar.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`. Há trabalho não commitado no repositório (`test-vault/`, `.claude/skills/`, `Resume-Claude.ps1`): `git diff <caminho>` antes de `git add <caminho>`; nunca `git add -A`.
- Nunca `go mod tidy`.
- Nunca despache subagentes. Nunca mate processo por nome (`Stop-Process -Name`); só por PID que você lançou.
- Referências e renames por gopls (LSP), não por grep. `grep` só para confirmar tags e reflection.
- Se um teste falhar por motivo que este brief não explica, **pare e reporte `BLOCKED`**; não ajuste a expectativa para o código passar.
- Este commit é de UMA categoria (o prefixo do assunto diz qual). Se você se pegar corrigindo outra coisa no caminho, anote em "O que ficou de fora" e não corrija.

#### Comando de mutação
Copie a âncora **do arquivo**, não de memória — âncora digitada sai `EXIT=2`. `0` = o teste reprovou sob mutação (o que se quer); `1` = a regra está escrita e não verificada; `2` = âncora ambígua ou build quebrado.

```bash
pwsh -File scripts/mutate.ps1 -Path internal/ipc/desconexao.go `
  -Anchor 'errors.Is(err, os.ErrClosed)' `
  -Replacement 'false' `
  -Test TestEhDesconexaoLimpaReconheceAsQuatroFormas -Package ./internal/ipc/
```
E no consumidor:
```bash
pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/serve.go `
  -Anchor 'if ipc.EhDesconexaoLimpa(err) {' `
  -Replacement 'if err == nil {' `
  -Test TestShutdownExitCode -Package ./cmd/gobsidian/
```
(Confira o nome real do teste de tabela em `serve_test.go:236`.)

#### Contrato de relatório
Escreva o relatório completo no arquivo de relatório indicado no despacho; devolva no chat só status, SHA, uma linha de testes e as preocupações. O relatório traz:
- **Status** — `DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`.
- **Commit** — SHA curto e assunto (`git log -1 --oneline`).
- **Evidência de TDD** — comando do RED com a saída falhando; comando do GREEN com a saída passando. Não "segui TDD".
- **Prova de mutação** — para cada regra reivindicada: o comando `mutate.ps1` (ou a mutação manual, com o diff), qual teste reprovou **pelo nome e pela linha**, a saída colada, e a confirmação de restauro (`git diff --stat` limpo no arquivo mutado).
- **As verificações do brief** — cada uma com o resultado real, inclusive as que deram certo.
- **`verify.ps1`** — a última linha, com a contagem de etapas.
- **O que ficou de fora** — e por quê. Vazio é resposta aceitável; ausente não é.
- **`git status --porcelain`** — colado. Nenhum arquivo do usuário tocado.

---

