package watcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Injetor de erro falso pra teste
type mockFsnotify struct {
	errors chan error
}

func TestOverflowReconciliationFull(t *testing.T) {
	root := t.TempDir()

	// 1. Arquivo modificado
	f1 := filepath.Join(root, "file1.md")
	os.WriteFile(f1, []byte("old"), 0644)

	// 2. Arquivo removido
	f2 := filepath.Join(root, "file2.md")
	os.WriteFile(f2, []byte("will be removed"), 0644)

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	idx.Build(context.Background(), v)

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Setup watcher
	w, err := New(v, idx, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Inicia watcher
	go w.Run(ctx)
	time.Sleep(50 * time.Millisecond) // espera iniciar

	// 3. Modifica file1 (muda tamanho e modtime)
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(f1, []byte("new content"), 0644)

	// 4. Remove file2
	os.Remove(f2)

	// 5. Adiciona file3
	f3 := filepath.Join(root, "file3.md")
	os.WriteFile(f3, []byte("added content"), 0644)

	// Injete o erro de overflow no canal do fsnotify original
	w.fsWatcher.Errors <- fsnotify.ErrEventOverflow

	// Dá tempo para a reconciliação ocorrer
	time.Sleep(200 * time.Millisecond)

	// Verifica: file1 modificado, file2 removido, file3 adicionado
	p1, _ := vault.Canonicalize(root, f1)
	if n, ok := idx.Get(p1); !ok || n.Size != int64(len("new content")) {
		t.Errorf("file1 não foi atualizado corretamente: %+v", n)
	}

	p2, _ := vault.Canonicalize(root, f2)
	if _, ok := idx.Get(p2); ok {
		t.Errorf("file2 não foi removido do índice")
	}

	p3, _ := vault.Canonicalize(root, f3)
	if n, ok := idx.Get(p3); !ok || n.Size != int64(len("added content")) {
		t.Errorf("file3 não foi adicionado ao índice: %+v", n)
	}
}
