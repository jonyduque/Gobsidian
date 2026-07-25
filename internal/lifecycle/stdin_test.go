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

// TestWaitReturnsWhenShutdownTriggeredWithStdinOpen cobre o cenario que a
// Task 4 (sinais) e a Task 5 (vigilia do pai) vao introduzir: um mecanismo
// diferente de stdin dispara o desligamento enquanto stdin continua aberto.
// Como nao ha como usar sinais ou vigilia do pai ainda (sao stubs), o
// disparo e simulado cancelando o context pai passado para New — e assim
// que um chamador real acionaria qualquer um dos outros mecanismos. Se a
// goroutine de watchStdin estiver registrada no WaitGroup, ela fica presa
// para sempre em Read (o pipe nunca e fechado neste teste) e Wait() trava.
func TestWaitReturnsWhenShutdownTriggeredWithStdinOpen(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })

	parentCtx, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()

	ctx, lc := lifecycle.New(parentCtx, lifecycle.Options{Stdin: pr})

	// Dispara o desligamento por um caminho que nao e stdin: cancelar o
	// context pai, como sinais ou vigilia do pai fariam.
	cancelParent()

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context nao foi cancelado apos cancelar o context pai")
	}

	done := make(chan struct{})
	go func() {
		lc.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() nao retornou com stdin ainda aberto apos desligamento externo; a goroutine de watchStdin nao deveria estar no WaitGroup")
	}
}
