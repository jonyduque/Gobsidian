package watcher

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/jonyd/gobsidian/internal/vault"
)

func TestFilter(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	// A raiz e os caminhos vinham escritos como "C:\\test\\vault". Em Linux e
	// macOS isso nao e um caminho com componentes: e um nome de arquivo unico
	// com contrabarras, nenhum evento cai dentro da raiz, e QUATRO dos seis
	// casos passavam a medir outra coisa — dois deles reprovavam com
	// reason="outside_vault", e os dois que esperavam emissao reprovavam com
	// emitted=false. O CI ficou vermelho de 2026-07-28 a 2026-08-01 por isto,
	// invisivel para scripts/verify.ps1, que so roda em Windows.
	root := t.TempDir()
	fora := t.TempDir()
	dentro := func(partes ...string) string {
		return filepath.Join(append([]string{root}, partes...)...)
	}

	tests := []struct {
		name       string
		event      fsnotify.Event
		wantEmit   bool
		wantOp     Op
		wantPath   vault.CanonicalPath
		wantReason DropReason
	}{
		{
			name: "ignora Chmod",
			event: fsnotify.Event{
				Name: dentro("nota.md"),
				Op:   fsnotify.Chmod,
			},
			wantEmit:   false,
			wantReason: DropChmod,
		},
		{
			name: "ignora fora do vault",
			event: fsnotify.Event{
				Name: filepath.Join(fora, "nota.md"),
				Op:   fsnotify.Write,
			},
			wantEmit:   false,
			wantReason: DropOutsideVault,
		},
		{
			name: "ignora pasta excluida",
			event: fsnotify.Event{
				Name: dentro(".git", "config"),
				Op:   fsnotify.Write,
			},
			wantEmit:   false,
			wantReason: DropExcluded,
		},
		{
			name: "ignora op desconhecida",
			event: fsnotify.Event{
				Name: dentro("nota.md"),
				Op:   0,
			},
			wantEmit:   false,
			wantReason: DropUnknownOp,
		},
		{
			name: "emite para nota md",
			event: fsnotify.Event{
				Name: dentro("nota.md"),
				Op:   fsnotify.Write,
			},
			wantEmit: true,
			wantOp:   OpWrite,
			wantPath: "nota.md",
		},
		{
			name: "emite para anexo asset",
			event: fsnotify.Event{
				Name: dentro("imagem.png"),
				Op:   fsnotify.Create,
			},
			wantEmit: true,
			wantOp:   OpCreate,
			wantPath: "imagem.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, emitted, reason := filter(tt.event, root, log)
			if emitted != tt.wantEmit {
				t.Fatalf("emitted = %v, want %v", emitted, tt.wantEmit)
			}
			if !emitted {
				if reason != tt.wantReason {
					t.Errorf("reason = %q, want %q", reason, tt.wantReason)
				}
			} else {
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

func TestFilter_OutsideVaultIsDropped(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	root := "C:\\test\\vault"

	evt := fsnotify.Event{
		Name: "D:\\fora_do_vault\\nota.md",
		Op:   fsnotify.Write,
	}

	_, emitted, reason := filter(evt, root, log)
	if emitted {
		t.Errorf("emitted = true, want false para evento fora do vault")
	}
	if reason != DropOutsideVault {
		t.Errorf("reason = %q, want %q", reason, DropOutsideVault)
	}
}
