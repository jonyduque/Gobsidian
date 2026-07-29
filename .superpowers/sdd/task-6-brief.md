### Task 6: Ciclo de vida — sequência de encerramento com orçamento

**Files:**
- Create: `internal/lifecycle/shutdown.go`, `internal/lifecycle/shutdown_test.go`

**Interfaces:**
- Consumes: nada de tarefas anteriores
- Produces: `lifecycle.Step{Name string, Budget time.Duration, Fn func(context.Context) error}`; `lifecycle.Shutdown(log *slog.Logger, hardLimit time.Duration, steps ...Step)`, sem valor de retorno

- [ ] **Step 1: Escrever o teste**

`internal/lifecycle/shutdown_test.go`:

```go
package lifecycle_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/lifecycle"
)

// capturingLogger devolve um logger e o buffer que ele escreve, para que os
// testes possam afirmar sobre o que foi registrado. Varias garantias deste
// pacote sao observaveis apenas pelo log — Shutdown nao devolve nada, e uma
// etapa que falha nao muda o comportamento visivel do processo.
func capturingLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})), &buf
}

func TestShutdownRunsStepsInOrder(t *testing.T) {
	log, _ := capturingLogger()
	var order []string

	lifecycle.Shutdown(log, 5*time.Second,
		lifecycle.Step{Name: "a", Budget: time.Second, Fn: func(context.Context) error {
			order = append(order, "a")
			return nil
		}},
		lifecycle.Step{Name: "b", Budget: time.Second, Fn: func(context.Context) error {
			order = append(order, "b")
			return nil
		}},
	)

	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Errorf("ordem = %v, quer [a b]", order)
	}
}

func TestShutdownStepExceedingBudgetDoesNotBlockNext(t *testing.T) {
	log, buf := capturingLogger()
	ran := false

	start := time.Now()
	lifecycle.Shutdown(log, 5*time.Second,
		lifecycle.Step{Name: "lenta", Budget: 100 * time.Millisecond, Fn: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}},
		lifecycle.Step{Name: "rapida", Budget: time.Second, Fn: func(context.Context) error {
			ran = true
			return nil
		}},
	)
	elapsed := time.Since(start)

	if !ran {
		t.Error("etapa seguinte nao rodou apos a anterior estourar o orcamento")
	}
	if elapsed > time.Second {
		t.Errorf("shutdown levou %v, deveria abandonar a etapa lenta em ~100ms", elapsed)
	}
	if !strings.Contains(buf.String(), "lenta") {
		t.Errorf("o abandono da etapa nao foi registrado; log = %q", buf.String())
	}
}

// Uma etapa que falha e registrada e nao interrompe o encerramento. Sem
// afirmar sobre o log, este teste nao afirmaria nada: Shutdown nao devolve
// valor, e o processo se comporta identicamente com e sem a falha.
func TestShutdownLogsStepErrorAndContinues(t *testing.T) {
	log, buf := capturingLogger()
	ran := false

	lifecycle.Shutdown(log, 5*time.Second,
		lifecycle.Step{Name: "falha", Budget: time.Second, Fn: func(context.Context) error {
			return errors.New("boom")
		}},
		lifecycle.Step{Name: "seguinte", Budget: time.Second, Fn: func(context.Context) error {
			ran = true
			return nil
		}},
	)

	if !ran {
		t.Error("etapa seguinte nao rodou apos a anterior falhar")
	}
	logged := buf.String()
	if !strings.Contains(logged, "falha") || !strings.Contains(logged, "boom") {
		t.Errorf("a falha da etapa nao foi registrada com nome e causa; log = %q", logged)
	}
}
```

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/lifecycle/ -run TestShutdown -v`
Esperado: FAIL — `undefined: lifecycle.Shutdown`.

- [ ] **Step 3: Implementar**

`internal/lifecycle/shutdown.go`:

```go
package lifecycle

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// Step e uma etapa da sequencia de encerramento, com orcamento proprio.
// Um timeout global unico nao basta: se as chamadas em voo consumirem o
// orcamento inteiro, a persistencia do cache comeca ja estourada.
type Step struct {
	Name   string
	Budget time.Duration
	Fn     func(context.Context) error
}

// Shutdown executa as etapas em ordem, cada uma com seu proprio relogio.
//
// Nao devolve codigo de saida, e a ausencia e deliberada. Falha de etapa e
// registrada, nunca fatal: o cache de indice que nao gravou custa alguns
// segundos no proximo boot, e transformar isso em codigo de erro faria o host
// reportar falha de um encerramento que deu certo. O unico caminho que sai
// com codigo diferente de zero e o guarda-chuva abaixo.
//
// hardLimit arma esse guarda-chuva: se qualquer etapa travar de um jeito que o
// orcamento dela nao previu, o processo morre mesmo assim. Encerrar com codigo
// de erro e ruim; sobreviver ao pai e pior.
func Shutdown(log *slog.Logger, hardLimit time.Duration, steps ...Step) {
	guard := time.AfterFunc(hardLimit, func() {
		log.Error("encerramento travou alem do limite rigido", "limit", hardLimit)
		os.Exit(1)
	})
	defer guard.Stop()

	for _, step := range steps {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), step.Budget)

		done := make(chan error, 1)
		go func() { done <- step.Fn(ctx) }()

		select {
		case err := <-done:
			if err != nil {
				log.Warn("etapa de encerramento falhou", "step", step.Name, "err", err, "duration", time.Since(start))
			} else {
				log.Debug("etapa de encerramento concluida", "step", step.Name, "duration", time.Since(start))
			}
		case <-ctx.Done():
			// A goroutine da etapa fica orfa de proposito. Ela recebeu um ctx
			// cancelado e deve sair sozinha; esperar por ela e exatamente o
			// que o orcamento existe para evitar.
			log.Warn("etapa de encerramento abandonada por estouro de orcamento",
				"step", step.Name, "budget", step.Budget)
		}
		cancel()
	}
}
```

- [ ] **Step 4: Rodar para confirmar que passa**

Run: `go test -race ./internal/lifecycle/ -v`
Esperado: PASS, sete testes.

- [ ] **Step 5: Commit**

```bash
git add internal/lifecycle
git commit -m "feat(lifecycle): shutdown sequence with per-step budget and hard limit"
```

---

