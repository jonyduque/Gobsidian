package boot_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/boot"
	"github.com/jonyd/gobsidian/internal/vaulttest"
)

func TestVigiarHostEOFDoStdinCancelaCtx(t *testing.T) {
	pr, pw := io.Pipe()
	ctx, v := boot.VigiarHost(context.Background(), pr, logSilencioso())
	// O servidor leria v.Stdin; aqui basta drenar para o espelho nao travar.
	go func() { _, _ = io.Copy(io.Discard, v.Stdin) }()
	if _, err := pw.Write([]byte("{}\n")); err != nil {
		t.Fatal(err)
	}
	_ = pw.Close()
	select {
	case <-ctx.Done():
	case <-time.After(vaulttest.Prazo):
		t.Fatal("EOF em stdin nao cancelou o ctx")
	}
	if r := v.LC.Reason(); r != "stdin-eof" {
		t.Fatalf("Reason = %q, quero \"stdin-eof\"", r)
	}
	passo := v.PassoFecharEspelho()
	if passo.Name != "close-pipe" || passo.Budget != 500*time.Millisecond {
		t.Fatalf("PassoFecharEspelho = {%q %v}", passo.Name, passo.Budget)
	}
	if err := passo.Fn(context.Background()); err != nil {
		t.Fatalf("fechar o espelho: %v", err)
	}
}

// TestPassoFecharEspelhoFechaOEspelho observa o EFEITO do passo, e nao o que
// ele devolve. O teste acima afirma o nome e o orcamento e depois so confere
// que Fn devolve nil — e um Fn que nao faz nada devolve nil tambem, entao
// `return v.pw.Close()` podia virar `return nil` com a suite inteira verde
// (achado N1 da revisao da Task 177).
//
// O truque e manter o stdin do host ABERTO: sem EOF por conta propria, o unico
// caminho ate o cancelamento e o passo fechar a ponta de escrita do espelho.
// io.Pipe.CloseWrite faz o lado de leitura devolver io.EOF, e watchStdin
// responde a io.EOF com trigger("stdin-eof") (internal/lifecycle/stdin.go).
func TestPassoFecharEspelhoFechaOEspelho(t *testing.T) {
	pr, pw := io.Pipe() // stdin do host, que NUNCA fecha por conta propria
	t.Cleanup(func() {
		_ = pw.Close()
		_ = pr.Close()
	})

	ctx, v := boot.VigiarHost(context.Background(), pr, logSilencioso())
	// O servidor leria v.Stdin; aqui basta drenar para o espelho nao travar.
	go func() { _, _ = io.Copy(io.Discard, v.Stdin) }()

	if err := v.PassoFecharEspelho().Fn(context.Background()); err != nil {
		t.Fatalf("fechar o espelho: %v", err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(vaulttest.Prazo):
		t.Fatal("fechar o espelho nao levou EOF ao monitor de stdin")
	}
	if r := v.LC.Reason(); r != "stdin-eof" {
		t.Fatalf("Reason = %q, quero \"stdin-eof\"", r)
	}
}
