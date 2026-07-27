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

	lifecycle.Shutdown(context.Background(), log, 5*time.Second,
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
	lifecycle.Shutdown(context.Background(), log, 5*time.Second,
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
	// A assercao precisa mirar uma palavra que so aparece no ramo de abandono.
	// O nome da etapa nao serve: ele e registrado tanto no abandono quanto na
	// falha, entao "lenta" no buffer provaria apenas que a etapa foi
	// mencionada em algum lugar, que nunca esteve em duvida.
	if !strings.Contains(buf.String(), "abandonada") {
		t.Errorf("o abandono da etapa nao foi registrado; log = %q", buf.String())
	}
}

// Uma etapa que falha e registrada e nao interrompe o encerramento. Sem
// afirmar sobre o log, este teste nao afirmaria nada: Shutdown nao devolve
// valor, e o processo se comporta identicamente com e sem a falha.
func TestShutdownLogsStepErrorAndContinues(t *testing.T) {
	log, buf := capturingLogger()
	ran := false

	lifecycle.Shutdown(context.Background(), log, 5*time.Second,
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

// TestShutdownIgnoresParentCancellation trava a razao de Shutdown usar
// context.WithoutCancel. O context raiz JA esta cancelado toda vez que
// Shutdown roda de verdade — e o cancelamento dele que traz o processo ate
// aqui. Derivar os orcamentos das etapas desse context faria cada etapa nascer
// expirada e ser abandonada antes de terminar, e o encerramento ordenado
// (gravar cache, drenar chamadas em voo) nunca aconteceria.
//
// A etapa precisa DEMORAR para o teste valer: step.Fn e lancada
// incondicionalmente numa goroutine, entao "a etapa rodou" e verdade mesmo com
// o orcamento expirado. O que muda e se Shutdown ESPERA por ela ou a abandona.
func TestShutdownIgnoresParentCancellation(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // raiz ja cancelada, como em producao

	concluiu := false
	start := time.Now()

	lifecycle.Shutdown(ctx, log, 5*time.Second,
		lifecycle.Step{Name: "lenta", Budget: 2 * time.Second, Fn: func(context.Context) error {
			time.Sleep(200 * time.Millisecond)
			concluiu = true
			return nil
		}},
	)
	elapsed := time.Since(start)

	if strings.Contains(buf.String(), "abandonada") {
		t.Fatalf("etapa abandonada sob raiz cancelada: o orcamento nasceu expirado; log = %q", buf.String())
	}
	if !concluiu {
		t.Error("Shutdown nao esperou a etapa terminar")
	}
	if elapsed < 150*time.Millisecond {
		t.Errorf("Shutdown retornou em %v, antes da etapa de 200ms terminar", elapsed)
	}
}
