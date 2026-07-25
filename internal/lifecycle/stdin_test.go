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
