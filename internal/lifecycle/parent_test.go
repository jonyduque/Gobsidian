package lifecycle_test

import (
	"context"
	"io"
	"os"
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
	parent, cancelParent := context.WithCancel(context.Background())
	ctx, lc := lifecycle.New(parent, lifecycle.Options{
		Stdin:               pr,
		ParentPID:           os.Getpid(),
		ParentCheckInterval: 50 * time.Millisecond,
	})

	select {
	case <-ctx.Done():
		t.Fatal("context cancelado com o pai vivo")
	case <-time.After(500 * time.Millisecond):
	}

	// Esta e a unica assercao do teste, e ela ja foi feita. O cancelamento
	// abaixo existe para desenrolar as goroutines: tanto o vigia de sinais
	// quanto o do pai bloqueiam em ctx.Done(), entao Wait() so retorna depois
	// que o context morre. Chamar Wait() sem cancelar antes trava para sempre.
	cancelParent()

	done := make(chan struct{})
	go func() { lc.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() nao retornou apos o cancelamento do context")
	}
}
