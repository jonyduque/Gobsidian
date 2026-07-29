### Task 5: Ciclo de vida — vigília do processo pai

**Files:**
- Delete: `internal/lifecycle/parent.go` (stub da Task 3)
- Create: `internal/lifecycle/parent.go` (parte compartilhada), `internal/lifecycle/parent_windows.go`, `internal/lifecycle/parent_unix.go`
- Create: `internal/lifecycle/parent_test.go`

**Interfaces:**
- Consumes: `(*Lifecycle).trigger` da Task 3
- Produces: `parentIdentity(pid int) (identity, error)` e `sameProcess(a, b identity) bool` por build tag; cancelamento com `Reason() == "parent-gone"`

- [ ] **Step 1: Escrever o teste**

`internal/lifecycle/parent_test.go`:

```go
package lifecycle_test

import (
	"context"
	"io"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/lifecycle"
)

func TestParentGoneCancelsContext(t *testing.T) {
	// Um processo de vida longa serve de "pai" sintetico. Ele precisa estar
	// VIVO quando New e chamado: parentIdentity captura a identidade inicial
	// no startup, e se essa captura falhar a vigilia se desabilita em vez de
	// disparar — comportamento correto para um PID que nunca foi observavel,
	// e que tornaria este teste vacuo se o processo ja estivesse morto aqui.
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "ping", "-n", "30", "127.0.0.1")
	} else {
		cmd = exec.Command("sleep", "30")
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	pid := cmd.Process.Pid

	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })

	ctx, lc := lifecycle.New(context.Background(), lifecycle.Options{
		Stdin:               pr,
		ParentPID:           pid,
		ParentCheckInterval: 50 * time.Millisecond,
	})

	// Agora que a identidade inicial foi capturada, mate o pai.
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	_ = cmd.Wait() // libera o handle; sem isso o PID continua consultavel

	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("context nao foi cancelado apos o pai morrer")
	}

	if got := lc.Reason(); got != "parent-gone" {
		t.Errorf("Reason() = %q, quer %q", got, "parent-gone")
	}
	lc.Wait()
}

func TestLiveParentKeepsContextAlive(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })

	// O proprio processo de teste como pai: esta vivo por definicao.
	ctx, lc := lifecycle.New(context.Background(), lifecycle.Options{
		Stdin:               pr,
		ParentPID:           os.Getpid(),
		ParentCheckInterval: 50 * time.Millisecond,
	})

	select {
	case <-ctx.Done():
		t.Fatal("context cancelado com o pai vivo")
	case <-time.After(500 * time.Millisecond):
	}
	lc.Wait()
}
```

Adicione `"os"` aos imports.

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/lifecycle/ -run TestParent -v`
Esperado: FAIL — `TestParentGoneCancelsContext` estoura o timeout, porque o stub não faz nada.

- [ ] **Step 3: Implementar a parte compartilhada**

Apague o stub e crie `internal/lifecycle/parent.go`:

```go
package lifecycle

import (
	"context"
	"time"
)

// watchParent e a rede de seguranca, para o caso em que o host morre de forma
// que deixe o stdin do filho sem fechar: reparentamento, heranca de handle por
// outro processo, comportamento anomalo do host.
//
// A verificacao compara identidade, nao apenas existencia. PIDs sao reciclados
// agressivamente no Windows, e checar so o PID produz falso negativo quando o
// PID do pai morto e atribuido a um processo novo.
func (l *Lifecycle) watchParent(ctx context.Context, pid int, interval time.Duration) {
	initial, err := parentIdentity(pid)
	if err != nil {
		// Nao da para vigiar o que nao da para identificar. Os outros dois
		// mecanismos continuam de pe.
		l.log.Warn("vigilia do processo pai desabilitada", "pid", pid, "err", err)
		return
	}

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				current, err := parentIdentity(pid)
				if err != nil || !sameProcess(initial, current) {
					l.trigger("parent-gone")
					return
				}
			}
		}
	}()
}
```

- [ ] **Step 4: Implementar a variante Windows**

`internal/lifecycle/parent_windows.go`:

```go
//go:build windows

package lifecycle

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// identity e PID mais creation time. So o par identifica um processo de forma
// estavel: o PID sozinho e reciclado, e o creation time sozinho colide entre
// processos iniciados no mesmo tick.
type identity struct {
	pid     int
	created windows.Filetime
}

func parentIdentity(pid int) (identity, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return identity{}, fmt.Errorf("abrindo processo %d: %w", pid, err)
	}
	defer func() { _ = windows.CloseHandle(h) }()

	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return identity{}, fmt.Errorf("lendo tempos do processo %d: %w", pid, err)
	}

	return identity{pid: pid, created: creation}, nil
}

func sameProcess(a, b identity) bool {
	return a.pid == b.pid &&
		a.created.HighDateTime == b.created.HighDateTime &&
		a.created.LowDateTime == b.created.LowDateTime
}
```

- [ ] **Step 5: Implementar a variante Unix**

`internal/lifecycle/parent_unix.go`:

```go
//go:build !windows

package lifecycle

import (
	"fmt"
	"os"
	"syscall"
)

// No Unix nao ha equivalente barato e portavel de creation time, e nao ha
// necessidade: quando o pai morre, o filho e reparentado para o init, e
// os.Getppid() passa a devolver 1. Esse e o sinal, e ele nao sofre de
// reutilizacao de PID.
type identity struct {
	pid     int
	reaped  bool
}

func parentIdentity(pid int) (identity, error) {
	if os.Getppid() == 1 && pid != 1 {
		return identity{pid: pid, reaped: true}, nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return identity{}, fmt.Errorf("localizando processo %d: %w", pid, err)
	}
	// Sinal 0 nao entrega nada; so testa existencia e permissao.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return identity{}, fmt.Errorf("processo %d inacessivel: %w", pid, err)
	}

	return identity{pid: pid}, nil
}

func sameProcess(a, b identity) bool {
	return a.pid == b.pid && !b.reaped
}
```

- [ ] **Step 6: Rodar para confirmar que passa**

Run: `go test -race ./internal/lifecycle/ -v`
Esperado: PASS, quatro testes (um SKIP no Windows).

- [ ] **Step 7: Commit**

```bash
git add internal/lifecycle
git commit -m "feat(lifecycle): watch parent process identity, not just PID"
```

---

