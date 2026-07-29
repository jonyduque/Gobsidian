### Task 3: Ciclo de vida — EOF em stdin

**Files:**
- Create: `internal/lifecycle/lifecycle.go`, `internal/lifecycle/stdin.go`, `internal/lifecycle/stdin_test.go`

**Interfaces:**
- Consumes: nada
- Produces: `lifecycle.Options{Stdin io.Reader, ParentPID int, ParentCheckInterval time.Duration, Logger *slog.Logger}`; `lifecycle.New(ctx context.Context, opts Options) (context.Context, *Lifecycle)`; `(*Lifecycle).Wait()`; `(*Lifecycle).Reason() string`

- [ ] **Step 1: Escrever o teste de EOF**

`internal/lifecycle/stdin_test.go`:

```go
package lifecycle_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/lifecycle"
)

func TestStdinEOFCancelsContext(t *testing.T) {
	ctx, lc := lifecycle.New(context.Background(), lifecycle.Options{
		Stdin: strings.NewReader(""), // leitura imediata devolve EOF
	})

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context nao foi cancelado apos EOF em stdin")
	}

	if got := lc.Reason(); got != "stdin-eof" {
		t.Errorf("Reason() = %q, quer %q", got, "stdin-eof")
	}
	lc.Wait()
}

func TestStdinOpenKeepsContextAlive(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })

	ctx, lc := lifecycle.New(context.Background(), lifecycle.Options{Stdin: pr})

	select {
	case <-ctx.Done():
		t.Fatal("context cancelado com stdin ainda aberto")
	case <-time.After(200 * time.Millisecond):
	}

	_ = pw.Close()
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context nao foi cancelado apos fechar o writer")
	}
	lc.Wait()
}
```

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/lifecycle/ -v`
Esperado: FAIL — `undefined: lifecycle.New`.

- [ ] **Step 3: Implementar a composição e o monitor de stdin**

`internal/lifecycle/lifecycle.go`:

```go
// Package lifecycle decide quando o processo deve morrer, e garante que ele
// morra. Tres mecanismos independentes cancelam o mesmo context raiz:
// EOF em stdin, sinais do sistema operacional e vigilia do processo pai.
// A redundancia e deliberada — cada um falha em cenarios diferentes.
package lifecycle

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

type Options struct {
	// Stdin e a fonte cujo EOF encerra o processo. os.Stdin em producao;
	// um io.Reader controlado nos testes.
	Stdin io.Reader

	// ParentPID e o processo cuja morte deve encerrar este. Zero desliga
	// a vigilia.
	ParentPID int

	ParentCheckInterval time.Duration

	Logger *slog.Logger
}

type Lifecycle struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu     sync.Mutex
	reason string

	log *slog.Logger
}

// New devolve um context que e cancelado quando qualquer um dos mecanismos
// dispara, e o Lifecycle que os coordena.
func New(parent context.Context, opts Options) (context.Context, *Lifecycle) {
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if opts.ParentCheckInterval == 0 {
		opts.ParentCheckInterval = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(parent)
	lc := &Lifecycle{cancel: cancel, log: opts.Logger}

	if opts.Stdin != nil {
		lc.watchStdin(ctx, opts.Stdin)
	}
	lc.watchSignals(ctx)
	if opts.ParentPID > 0 {
		lc.watchParent(ctx, opts.ParentPID, opts.ParentCheckInterval)
	}

	return ctx, lc
}

// trigger cancela o context, registrando o primeiro motivo que chegou.
// Chamadas subsequentes nao sobrescrevem o motivo: o que importa e quem
// disparou primeiro.
func (l *Lifecycle) trigger(reason string) {
	l.mu.Lock()
	if l.reason == "" {
		l.reason = reason
		l.log.Info("encerramento solicitado", "reason", reason)
	}
	l.mu.Unlock()
	l.cancel()
}

func (l *Lifecycle) Reason() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reason
}

// Wait bloqueia ate todas as goroutines do lifecycle terminarem.
func (l *Lifecycle) Wait() { l.wg.Wait() }

// ParentPID devolve o PID do processo pai no momento da chamada.
func ParentPID() int { return os.Getppid() }
```

`internal/lifecycle/stdin.go`:

```go
package lifecycle

import (
	"context"
	"errors"
	"io"
)

// watchStdin encerra o processo quando o host MCP fecha o stdin do filho.
// E o mecanismo primario e o mais confiavel no Windows: o sistema operacional
// fecha os handles de um processo morto, inclusive apos taskkill /F.
func (l *Lifecycle) watchStdin(ctx context.Context, r io.Reader) {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()

		buf := make([]byte, 4096)
		for {
			_, err := r.Read(buf)
			if err == nil {
				continue
			}
			if errors.Is(err, io.EOF) {
				l.trigger("stdin-eof")
				return
			}
			// Qualquer outro erro de leitura em stdin tambem significa que
			// o canal com o host morreu. Tratar diferente seria fingir que
			// ha um estado de recuperacao que nao existe.
			l.trigger("stdin-error")
			return
		}
	}()
}
```

**Atenção ao ponto sutil.** O monitor de stdin **consome bytes**. Em `serve`, o stdin é do JSON-RPC e não pode ser lido por duas goroutines. A solução aparece na Task 9: o transporte do SDK recebe um `io.TeeReader`, e o lifecycle observa a metade espelhada. Este monitor lê e descarta, o que só é correto quando ele é o único leitor — nos testes, e no `doctor`.

- [ ] **Step 4: Adicionar stubs de sinais e de pai para compilar**

`internal/lifecycle/signals.go` e a vigília do pai são implementados nas Tasks 4 e 5. Por ora, para que o pacote compile:

```go
// signals.go
package lifecycle

import "context"

func (l *Lifecycle) watchSignals(ctx context.Context) {}
```

```go
// parent.go — removido na Task 5
package lifecycle

import (
	"context"
	"time"
)

func (l *Lifecycle) watchParent(ctx context.Context, pid int, interval time.Duration) {}
```

- [ ] **Step 5: Rodar para confirmar que passa**

Run: `go test -race ./internal/lifecycle/ -v`
Esperado: PASS, dois testes.

- [ ] **Step 6: Commit**

```bash
git add internal/lifecycle
git commit -m "feat(lifecycle): cancel root context on stdin EOF"
```

---

