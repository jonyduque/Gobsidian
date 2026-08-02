package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func TestWatcher_Burst(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
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
	go func() {
		_ = w.Run(ctx)
	}()

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	count := 500
	for i := 0; i < count; i++ {
		path := filepath.Join(dir, fmt.Sprintf("note_%d.md", i))
		_ = os.WriteFile(path, []byte(fmt.Sprintf("note %d", i)), 0644)
	}

	// O que este teste mede e CONVERGENCIA — as 500 notas chegam ao indice —,
	// nao o tempo que ela leva. O prazo era de 10 s (100 x 100 ms), e sob -race
	// com a suite inteira rodando em paralelo ele estourou com 427 de 500: o
	// watcher estava progredindo, so nao terminou dentro da janela. Prazo curto
	// num teste de convergencia mede a carga da maquina.
	//
	// 60 s e generoso de proposito. Ele nao afrouxa nada: se a convergencia nao
	// acontecer, o teste continua reprovando, e com o numero parcial na
	// mensagem para distinguir "parou" de "estava devagar".
	prazo := time.Now().Add(60 * time.Second)
	for time.Now().Before(prazo) {
		if idx.NoteCount() == count {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if idx.NoteCount() != count {
		t.Fatalf("esperava %d notas no indice, encontrou %d", count, idx.NoteCount())
	}

	var walkCount int
	_ = v.Walk(context.Background(), func(e vault.Entry) error {
		if e.IsNote {
			walkCount++
		}
		return nil
	})
	if idx.NoteCount() != walkCount {
		t.Fatalf("esperava que NoteCount (%d) correspondesse a v.Walk (%d)", idx.NoteCount(), walkCount)
	}
}
