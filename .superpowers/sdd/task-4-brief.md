### Task 4: Ciclo de vida — sinais do sistema operacional

**Files:**
- Modify: `internal/lifecycle/signals.go` (substitui o stub da Task 3)
- Create: `internal/lifecycle/signals_test.go`

**Interfaces:**
- Consumes: `(*Lifecycle).trigger`, `(*Lifecycle).wg` da Task 3
- Produces: cancelamento com `Reason() == "signal"` ao receber `os.Interrupt` ou `syscall.SIGTERM`

- [ ] **Step 1: Escrever o teste**

`internal/lifecycle/signals_test.go`:

```go
package lifecycle_test

import (
	"context"
	"io"
	"syscall"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/lifecycle"
)

func TestSignalCancelsContext(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })

	ctx, lc := lifecycle.New(context.Background(), lifecycle.Options{Stdin: pr})

	// Da tempo de o handler ser instalado antes de disparar o sinal.
	time.Sleep(100 * time.Millisecond)

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess: %v", err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		t.Skipf("sinal nao entregavel nesta plataforma: %v", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context nao foi cancelado apos SIGTERM")
	}

	if got := lc.Reason(); got != "signal" {
		t.Errorf("Reason() = %q, quer %q", got, "signal")
	}
	lc.Wait()
}
```

Adicione `"os"` aos imports do arquivo de teste.

**Nota sobre Windows.** `proc.Signal(syscall.SIGTERM)` não é entregue no Windows; o teste chama `t.Skipf` nesse caso, e é correto que ele pule. É exatamente por isso que os outros dois mecanismos existem (WINDOWS.md §5.1) — o teste que importa no Windows é o de órfãos, na Task 11.

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/lifecycle/ -run TestSignal -v`
Esperado: FAIL no Linux/macOS — o context não é cancelado, porque o handler é um stub. No Windows, SKIP.

- [ ] **Step 3: Implementar**

`internal/lifecycle/signals.go`:

```go
package lifecycle

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// watchSignals cobre o encerramento cooperativo: Ctrl+C no terminal, SIGTERM
// de um supervisor. No Windows, os.Interrupt cobre CTRL_C_EVENT e
// CTRL_BREAK_EVENT; SIGTERM e aceito pela API mas nao e entregue por
// taskkill sem /F. Nao cobre SIGKILL nem taskkill /F — e nao precisa:
// nesses casos o processo morre de fato, que e o resultado desejado.
func (l *Lifecycle) watchSignals(ctx context.Context) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		defer signal.Stop(ch)

		select {
		case <-ch:
			l.trigger("signal")
		case <-ctx.Done():
		}
	}()
}
```

- [ ] **Step 4: Rodar para confirmar que passa**

Run: `go test -race ./internal/lifecycle/ -v`
Esperado: PASS em todas as plataformas (SKIP do teste de sinal no Windows).

- [ ] **Step 5: Commit**

```bash
git add internal/lifecycle
git commit -m "feat(lifecycle): cancel root context on interrupt and SIGTERM"
```

---

