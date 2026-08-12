//go:build windows

package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestToolsDepoisDeReplaceEmNotaSomenteNuvem fixa a MUDANCA DE COMPORTAMENTO
// DE TOOL que a Task 95 trouxe.
//
// Antes, index.Replace registrava um `.md` somente-nuvem como Asset. Um evento
// do watcher sobre esse arquivo REBAIXAVA a nota que o boot tinha indexado:
// idx.Get passava a devolver false, e note_read, note_metadata e note_list
// respondiam como se o arquivo nao existisse. Agora ele continua Note com
// CloudOnly, e as tres respondem sobre uma nota que existe e nao pode ser lida.
//
// A diferenca que importa para quem chama: note_read devolvia NOTE_NOT_FOUND e
// passa a devolver CLOUD_ONLY_FILE. O segundo e verdade e o primeiro nao era.
//
// Nenhuma das tres pode ABRIR o arquivo: abrir placeholder dispara download
// sincrono. O teste segura um handle exclusivo sobre ele e confere que o handle
// barra a leitura antes de afirmar qualquer coisa.
func TestToolsDepoisDeReplaceEmNotaSomenteNuvem(t *testing.T) {
	root := t.TempDir()
	caminho := filepath.Join(root, "nuvem.md")
	if err := os.WriteFile(caminho, []byte("# Titulo da nuvem\n\ncorpo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := windows.UTF16PtrFromString(vault.LongPath(caminho))
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Skipf("nao foi possivel marcar FILE_ATTRIBUTE_OFFLINE: %v", err)
	}
	t.Cleanup(func() {
		_ = windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_NORMAL)
	})

	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	if n, ok := idx.Get("nuvem.md"); !ok || !n.CloudOnly {
		t.Fatal("a nota nao ficou marcada CloudOnly; o atributo nao pegou")
	}

	// Handle exclusivo: leitura E escrita, medido como o unico modo que barra o
	// os.Open do vault.ReadAll nesta maquina.
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Skipf("nao foi possivel abrir o arquivo em modo exclusivo: %v", err)
	}
	t.Cleanup(func() { _ = windows.CloseHandle(h) })
	if _, err := os.ReadFile(caminho); err == nil {
		t.Fatal("o handle exclusivo nao barrou a leitura; a prova de 'nao abriu' seria vazia")
	}

	// O evento do watcher.
	if err := idx.Replace(context.Background(), v, "nuvem.md"); err != nil {
		t.Fatalf("Replace: %v — o arquivo foi aberto", err)
	}

	svc := service.New(v, idx, nil, nil, service.Options{})

	_, err = svc.ReadNote(context.Background(), service.ReadRequest{Path: "nuvem.md"})
	if err == nil {
		t.Fatal("note_read devolveu sucesso sobre um placeholder de nuvem")
	}
	if got := service.CodeOf(err); got != service.CodeCloudOnlyFile {
		t.Fatalf("note_read devolveu %s, quer %s", got, service.CodeCloudOnlyFile)
	}

	meta, err := svc.NoteMetadata(context.Background(), service.MetadataRequest{Path: "nuvem.md"})
	if err != nil {
		t.Fatalf("note_metadata: %v", err)
	}
	if meta.Title != "" {
		t.Fatalf("note_metadata devolveu Title = %q; o placeholder foi lido", meta.Title)
	}
	if len(meta.Headings) != 0 {
		t.Fatalf("note_metadata devolveu %d headings; o placeholder foi parseado", len(meta.Headings))
	}

	lista, err := svc.ListNotes(context.Background(), service.ListRequest{})
	if err != nil {
		t.Fatalf("note_list: %v", err)
	}
	var achou bool
	for _, it := range lista.Notes {
		if it.Path == "nuvem.md" {
			achou = true
		}
	}
	if !achou {
		t.Fatalf("note_list nao trouxe a nota somente-nuvem: %+v", lista.Notes)
	}
}
