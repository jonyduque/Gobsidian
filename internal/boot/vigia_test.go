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
