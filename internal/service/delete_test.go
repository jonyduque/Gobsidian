package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

func createDeleteService(t *testing.T, files map[string]string) (*service.Service, *vault.Vault, *index.Index, string) {
	t.Helper()
	root := t.TempDir()

	for relPath, content := range files {
		full := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	inv := search.NewInverted()
	svc := service.New(v, idx, inv, nil, service.Options{})
	return svc, v, idx, root
}

func TestDeleteNote_ReportBrokenLinksBeforeDeletion(t *testing.T) {
	files := map[string]string{
		"alvo.md": "# Alvo",
		"ref.md":  "Link para [[alvo]]",
	}

	svc, _, _, root := createDeleteService(t, files)

	res, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{
		Path:              "alvo.md",
		ToTrash:           true,
		ReportBrokenLinks: true,
	})
	if err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	if len(res.BrokenLinks) != 1 || res.BrokenLinks[0] != "ref.md" {
		t.Errorf("BrokenLinks = %v; quer [ref.md]", res.BrokenLinks)
	}

	if _, err := os.Stat(filepath.Join(root, "alvo.md")); err == nil {
		t.Error("alvo.md ainda existe no caminho original apos DeleteNote")
	}

	if _, err := os.Stat(filepath.Join(root, ".trash", "alvo.md")); err != nil {
		t.Errorf("alvo.md nao foi encontrado em .trash/: %v", err)
	}
}

func TestDeleteNote_ToTrashFalseDefiniteDelete(t *testing.T) {
	files := map[string]string{
		"alvo.md": "Conteudo",
	}

	svc, _, _, root := createDeleteService(t, files)

	res, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{
		Path:    "alvo.md",
		ToTrash: false,
	})
	if err != nil {
		t.Fatalf("DeleteNote to_trash=false: %v", err)
	}

	if !res.Deleted || res.MovedToTrash {
		t.Errorf("res = %+v; quer Deleted=true, MovedToTrash=false", res)
	}

	if _, err := os.Stat(filepath.Join(root, "alvo.md")); err == nil {
		t.Error("alvo.md ainda existe apos exclusao definitiva")
	}

	if _, err := os.Stat(filepath.Join(root, ".trash", "alvo.md")); err == nil {
		t.Error("alvo.md foi encontrado em .trash/ mas a exclusao era definitiva")
	}
}

func TestDeleteNote_TrashNameCollision(t *testing.T) {
	files := map[string]string{
		"a.md": "Versao 1",
	}

	svc, v, idx, root := createDeleteService(t, files)

	// Primeira exclusao
	res1, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{
		Path:    "a.md",
		ToTrash: true,
	})
	if err != nil {
		t.Fatalf("Primeira exclusao: %v", err)
	}

	// Recria a.md com outro conteudo
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("Versao 2"), 0644); err != nil {
		t.Fatalf("WriteFile a.md: %v", err)
	}
	_ = idx.Build(context.Background(), v)

	time.Sleep(10 * time.Millisecond)

	// Segunda exclusao com mesmo nome
	res2, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{
		Path:    "a.md",
		ToTrash: true,
	})
	if err != nil {
		t.Fatalf("Segunda exclusao: %v", err)
	}

	if res1.TrashPath == res2.TrashPath {
		t.Errorf("colisao de nomes na lixeira: res1=%q, res2=%q", res1.TrashPath, res2.TrashPath)
	}

	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(res1.TrashPath))); err != nil {
		t.Errorf("primeiro arquivo da lixeira sumiu: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(res2.TrashPath))); err != nil {
		t.Errorf("segundo arquivo da lixeira sumiu: %v", err)
	}
}

func TestDeleteNote_DryRunDoesNotDelete(t *testing.T) {
	files := map[string]string{
		"alvo.md": "Conteudo",
		"ref.md":  "Link [[alvo]]",
	}

	svc, _, _, root := createDeleteService(t, files)

	alvoPath := filepath.Join(root, "alvo.md")
	infoBefore, _ := os.Stat(alvoPath)

	res, err := svc.DeleteNote(context.Background(), service.DeleteNoteRequest{
		Path:              "alvo.md",
		ToTrash:           true,
		ReportBrokenLinks: true,
		DryRun:            true,
	})
	if err != nil {
		t.Fatalf("DeleteNote dry_run: %v", err)
	}

	if !res.DryRun || res.Deleted {
		t.Errorf("res = %+v; quer DryRun=true, Deleted=false", res)
	}

	if len(res.BrokenLinks) != 1 || res.BrokenLinks[0] != "ref.md" {
		t.Errorf("BrokenLinks em dry_run = %v; quer [ref.md]", res.BrokenLinks)
	}

	infoAfter, err := os.Stat(alvoPath)
	if err != nil {
		t.Fatalf("alvo.md sumiu apos dry_run: %v", err)
	}

	if !infoBefore.ModTime().Equal(infoAfter.ModTime()) {
		t.Error("mtime de alvo.md foi alterado durante dry_run")
	}
}
