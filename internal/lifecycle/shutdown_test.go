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
