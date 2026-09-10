package watcher

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jonyduque/Gobsidian/internal/index"
	"github.com/jonyduque/Gobsidian/internal/search"
	"github.com/jonyduque/Gobsidian/internal/vault"
	"github.com/jonyduque/Gobsidian/internal/vaulttest"
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

	EsperarWatcherAtivo(t, w)

	// Create a note
	notePath := filepath.Join(dir, "teste.md")
	if err := os.WriteFile(notePath, []byte("conteudo"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Verify event via index
	canon, _ := vault.Canonicalize(dir, notePath)
	if !EsperarAte(vaulttest.Prazo, func() bool {
		_, ok := idx.Get(canon)
		return ok
	}) {
		t.Fatalf("o indice nao recebeu 'teste.md' em %v\n%s", vaulttest.Prazo, diagnostico(w))
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

	EsperarWatcherAtivo(t, w)

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

	EsperarWatcherAtivo(t, w)

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
	EsperarWatcherAtivo(t, w)

	subDir := filepath.Join(tmp, "nova_pasta")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// O sinal observavel de "o watch do diretorio novo ja foi registrado" e a
	// lista de watches do proprio fsnotify — leitura, nao escrita em canal
	// interno. Os 100 ms que estavam aqui eram um palpite sobre quanto o Run
	// leva para tratar o evento de criacao, e errar esse palpite nao produz uma
	// falha honesta: a nota seria escrita ANTES do Add, o evento dela nunca
	// existiria, e o teste reprovaria dizendo "subdiretorio nao vigiado" quando
	// o que houve foi uma corrida do proprio teste.
	if !EsperarAte(vaulttest.Prazo, func() bool { return estaVigiado(w, subDir) }) {
		t.Fatalf("o watch em %s nao foi registrado em %v; lista=%v\n%s",
			subDir, vaulttest.Prazo, w.fsWatcher.WatchList(), diagnostico(w))
	}

	notePath := filepath.Join(subDir, "subnota.md")
	if err := os.WriteFile(notePath, []byte("# Subnota\n"), 0644); err != nil {
		t.Fatal(err)
	}

	canon, _ := vault.Canonicalize(tmp, notePath)
	if !EsperarAte(vaulttest.Prazo, func() bool {
		_, ok := idx.Get(canon)
		return ok
	}) {
		t.Errorf("nota em subdiretório criado dinamicamente (%s) não foi indexada\n%s",
			canon, diagnostico(w))
	}
}

// estaVigiado diz se o fsnotify tem watch registrado naquele diretorio.
//
// A comparacao e por caminho limpo e sem distincao de caixa porque o nome que
// chega em WatchList e o que o backend recebeu no Add — no Windows ele vem do
// evento do sistema operacional, e a caixa dele nao e a que o teste escreveu.
// Comparar as strings cruas daria um "nao vigiado" falso.
func estaVigiado(w *Watcher, dir string) bool {
	alvo := filepath.Clean(dir)
	for _, p := range w.fsWatcher.WatchList() {
		if strings.EqualFold(filepath.Clean(p), alvo) {
			return true
		}
	}
	return false
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
	EsperarWatcherAtivo(t, w)

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

// esperaTermo espera pelo estado do indice de busca. time.Sleep fixo como
// assercao e o que faz um teste passar sem o mecanismo existir.
func esperaTermo(t *testing.T, inv *search.Inverted, termo, path string, quer bool) {
	t.Helper()
	ok := EsperarAte(vaulttest.Prazo, func() bool {
		for _, p := range inv.Postings(termo) {
			if p.Path == path {
				return quer
			}
		}
		return !quer
	})
	if !ok {
		t.Fatalf("apos %v, %q em %q: presente=%v, quer %v — a busca nao acompanhou o cofre",
			vaulttest.Prazo, termo, path, !quer, quer)
	}
}
