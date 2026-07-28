package watcher

import (
	"log/slog"
	"os"
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/jonyd/gobsidian/internal/vault"
)

func TestFilter(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	root := "C:\\test\\vault"

	tests := []struct {
		name     string
		event    fsnotify.Event
		wantEmit bool
		wantOp   Op
		wantPath vault.CanonicalPath
	}{
		{
			name: "ignora Chmod",
			event: fsnotify.Event{
				Name: "C:\\test\\vault\\nota.md",
				Op:   fsnotify.Chmod,
			},
			wantEmit: false,
		},
		{
			name: "ignora pasta excluida",
			event: fsnotify.Event{
				Name: "C:\\test\\vault\\.git\\config",
				Op:   fsnotify.Write,
			},
			wantEmit: false,
		},
		{
			name: "ignora ruido desktop.ini",
			event: fsnotify.Event{
				Name: "C:\\test\\vault\\desktop.ini",
				Op:   fsnotify.Create,
			},
			wantEmit: false,
		},
		{
			name: "ignora extensao desconhecida",
			event: fsnotify.Event{
				Name: "C:\\test\\vault\\arquivo.txt",
				Op:   fsnotify.Create,
			},
			wantEmit: false,
		},
		{
			name: "emite para nota md",
			event: fsnotify.Event{
				Name: "C:\\test\\vault\\nota.md",
				Op:   fsnotify.Write,
			},
			wantEmit: true,
			wantOp:   OpWrite,
			wantPath: "nota.md",
		},
		{
			name: "emite para anexo asset",
			event: fsnotify.Event{
				Name: "C:\\test\\vault\\imagem.png",
				Op:   fsnotify.Create,
			},
			wantEmit: true,
			wantOp:   OpCreate,
			wantPath: "imagem.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, emitted := filter(tt.event, root, log)
			if emitted != tt.wantEmit {
				t.Fatalf("emitted = %v, want %v", emitted, tt.wantEmit)
			}
			if emitted {
				if got.Op != tt.wantOp {
					t.Errorf("Op = %v, want %v", got.Op, tt.wantOp)
				}
				if got.Path != tt.wantPath {
					t.Errorf("Path = %q, want %q", got.Path, tt.wantPath)
				}
			}
		})
	}
}
