package watcher

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

func TestWatcher(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dir := t.TempDir()

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()

	w, err := New(v, idx, nil, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errc := make(chan error, 1)
	go func() {
		errc <- w.Run(ctx)
	}()

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Create a note
	notePath := filepath.Join(dir, "teste.md")
	if err := os.WriteFile(notePath, []byte("conteudo"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Verify event via index
	found := false
	canon, _ := vault.Canonicalize(dir, notePath)
	for range 50 {
		if _, ok := idx.Get(canon); ok {
			found = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !found {
		t.Fatal("timeout waiting for index to update with 'teste.md'")
	}

	// Shutdown test
	cancel()
	select {
	case err := <-errc:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Run error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for Run to exit")
	}

	if err := w.Close(); err != nil {
		t.Errorf("Close() = %v", err)
	}
}

func TestWatcher_CloseReleasesHandles(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("teste de travamento de handle de diretório é específico do Windows")
	}

	tmp := t.TempDir()
	vaultDir := filepath.Join(tmp, "vault_root")
	if err := os.MkdirAll(vaultDir, 0755); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(vaultDir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	w, err := New(v, idx, nil, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() {
		errc <- w.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	cancel()
	<-errc
	if err := w.Close(); err != nil {
		t.Fatalf("w.Close: %v", err)
	}

	targetDir := filepath.Join(tmp, "vault_renamed")
	if err := os.Rename(vaultDir, targetDir); err != nil {
		t.Fatalf("os.Rename da raiz falhou após Close: %v", err)
	}
	if err := os.RemoveAll(targetDir); err != nil {
		t.Fatalf("os.RemoveAll da raiz falhou após Close: %v", err)
	}
}

func TestWatcher_EventsChannelClosedOnShutdown(t *testing.T) {
	tmp := t.TempDir()
	v, _ := vault.New(tmp)
	idx := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	w, err := New(v, idx, nil, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = w.Run(ctx) }()

	time.Sleep(50 * time.Millisecond)

	cancel()
	_ = w.Close()

	select {
	case _, ok := <-w.events:
		if ok {
			t.Error("w.events aberto após shutdown, esperava ok == false")
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout aguardando o fechamento do canal w.events")
	}
}

func TestWatcher_DirCreatedAfterStartIsWatched(t *testing.T) {
	tmp := t.TempDir()
	v, _ := vault.New(tmp)
	idx := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	w, err := New(v, idx, nil, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() { _ = w.Close() }()

	go func() { _ = w.Run(ctx) }()
	time.Sleep(50 * time.Millisecond)

	subDir := filepath.Join(tmp, "nova_pasta")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	notePath := filepath.Join(subDir, "subnota.md")
	if err := os.WriteFile(notePath, []byte("# Subnota\n"), 0644); err != nil {
		t.Fatal(err)
	}

	canon, _ := vault.Canonicalize(tmp, notePath)
	deadline := time.Now().Add(3 * time.Second)
	found := false
	for time.Now().Before(deadline) {
		if _, ok := idx.Get(canon); ok {
			found = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !found {
		t.Errorf("nota em subdiretório criado dinamicamente (%s) não foi indexada", canon)
	}
}

func TestNew_FailsOnUnwatchablePath(t *testing.T) {
	nonExistentDir := filepath.Join(t.TempDir(), "subpasta_inexistente")
	v, err := vault.New(nonExistentDir)
	if err != nil {
		t.Skipf("vault.New recusou caminho inexistente: %v", err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	idx := index.New()

	w, err := New(v, idx, nil, 10*time.Millisecond, log)
	if err == nil {
		_ = w.Close()
		t.Fatal("New esperava erro ao observar caminho inexistente, mas obteve nil")
	}
}

// TestWatcherUpdatesSearchIndex e o teste que prova que a busca acompanha o
// cofre. Sem ele, o gate fica verde com search.Update sendo codigo morto.
func TestWatcherUpdatesSearchIndex(t *testing.T) {
	tmp := t.TempDir()
	v, err := vault.New(tmp)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	inv := search.NewInverted()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	w, err := New(v, idx, inv, 10*time.Millisecond, log)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = w.Run(ctx) }()
	time.Sleep(100 * time.Millisecond) // watcher precisa estar observando

	// CRIACAO: a nota nova tem de passar a ser encontravel.
	if err := os.WriteFile(filepath.Join(tmp, "nova.md"), []byte("prescricao intercorrente"), 0644); err != nil {
		t.Fatal(err)
	}
	esperaTermo(t, inv, "prescricao", "nova.md", true)

	// REMOCAO: a nota removida tem de deixar de ser encontravel. Um teste que
	// so cobre a criacao passa com um Remove que nunca acontece.
	if err := os.Remove(filepath.Join(tmp, "nova.md")); err != nil {
		t.Fatal(err)
	}
	esperaTermo(t, inv, "prescricao", "nova.md", false)
}

// esperaTermo espera em laco com condicao de saida. time.Sleep fixo como
// assercao e o que faz um teste passar sem o mecanismo existir.
func esperaTermo(t *testing.T, inv *search.Inverted, termo, path string, quer bool) {
	t.Helper()
	limite := time.Now().Add(3 * time.Second)
	for time.Now().Before(limite) {
		presente := false
		for _, p := range inv.Postings(termo) {
			if p.Path == path {
				presente = true
			}
		}
		if presente == quer {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("apos 3s, %q em %q: presente=%v, quer %v — a busca nao acompanhou o cofre",
		termo, path, !quer, quer)
}
