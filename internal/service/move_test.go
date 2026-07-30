package service_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

func createMoveService(t *testing.T, files map[string]string) (*service.Service, *vault.Vault, *index.Index, string) {
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

func TestNoteMovePartialFailureReportsWhatWasApplied(t *testing.T) {
	files := map[string]string{
		"alvo.md": "# Alvo\nConteudo",
		"a.md":    "Link para [[alvo]]",
		"b.md":    "Link para [[alvo]]",
	}

	svc, _, _, root := createMoveService(t, files)

	// Torna b.md somente leitura no Windows / Unix
	pathB := filepath.Join(root, "b.md")
	if err := os.Chmod(pathB, 0400); err != nil {
		t.Fatalf("Chmod b.md: %v", err)
	}
	defer func() { _ = os.Chmod(pathB, 0644) }()

	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:          "alvo.md",
		To:            "Novo/alvo.md",
		UpdateLinks:   true,
		CreateFolders: true,
	})

	if err == nil {
		t.Fatal("esperava falha parcial; o cenario tornou uma nota inescrivel")
	}

	// 1. O relatorio deve listar a.md que foi reescrita antes da falha
	if len(res.Rewritten) == 0 {
		t.Errorf("Rewritten esta vazio; esperava registrar [a.md] antes da falha")
	}

	// 2. O arquivo alvo.md NAO pode ter se movido
	if _, err := os.Stat(filepath.Join(root, "alvo.md")); err != nil {
		t.Errorf("o alvo.md saiu do lugar apesar da falha: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "Novo", "alvo.md")); err == nil {
		t.Error("o alvo foi movido para Novo/alvo.md apesar da falha antes do passo de move")
	}
}

func TestMoveNote_DryRunLeavesMtimeIntact(t *testing.T) {
	files := map[string]string{
		"alvo.md": "Alvo",
		"ref.md":  "Ref para [[alvo]]",
	}

	svc, _, _, root := createMoveService(t, files)

	alvoPath := filepath.Join(root, "alvo.md")
	refPath := filepath.Join(root, "ref.md")

	infoBeforeAlvo, _ := os.Stat(alvoPath)
	infoBeforeRef, _ := os.Stat(refPath)

	time.Sleep(10 * time.Millisecond)

	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:          "alvo.md",
		To:            "Novo/alvo.md",
		UpdateLinks:   true,
		CreateFolders: true,
		DryRun:        true,
	})
	if err != nil {
		t.Fatalf("MoveNote dry_run: %v", err)
	}

	if !res.DryRun {
		t.Error("esperado DryRun=true no resultado")
	}
	if len(res.Diffs) == 0 {
		t.Error("esperava diffs populados em dry_run")
	}

	infoAfterAlvo, _ := os.Stat(alvoPath)
	infoAfterRef, _ := os.Stat(refPath)

	if !infoBeforeAlvo.ModTime().Equal(infoAfterAlvo.ModTime()) {
		t.Error("mtime de alvo.md foi alterado durante dry_run")
	}
	if !infoBeforeRef.ModTime().Equal(infoAfterRef.ModTime()) {
		t.Error("mtime de ref.md foi alterado durante dry_run")
	}
}

func TestMoveNote_UpdateLinksFalse(t *testing.T) {
	files := map[string]string{
		"alvo.md": "Alvo",
		"ref.md":  "Ref para [[alvo]]",
	}

	svc, _, _, root := createMoveService(t, files)
	refPath := filepath.Join(root, "ref.md")

	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:        "alvo.md",
		To:          "destino.md",
		UpdateLinks: false,
	})
	if err != nil {
		t.Fatalf("MoveNote update_links=false: %v", err)
	}

	if res.LinksUpdated != 0 || len(res.Rewritten) != 0 {
		t.Errorf("esperava 0 links atualizados, obtido %d", res.LinksUpdated)
	}

	refRaw, _ := os.ReadFile(refPath)
	if string(refRaw) != "Ref para [[alvo]]" {
		t.Errorf("ref.md foi alterado apesar de update_links=false: %s", refRaw)
	}
}

func TestMoveNote_CreateFoldersFalseMissingDir(t *testing.T) {
	files := map[string]string{
		"alvo.md": "Alvo",
	}

	svc, _, _, _ := createMoveService(t, files)

	_, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:          "alvo.md",
		To:            "PastaInexistente/alvo.md",
		CreateFolders: false,
	})

	if err == nil {
		t.Fatal("esperava erro ao mover para pasta inexistente com create_folders=false")
	}

	var codeErr *service.Error
	if errors.As(err, &codeErr) {
		if codeErr.Code != service.CodeFolderNotFound {
			t.Errorf("esperava CodeFolderNotFound, obtido %v", codeErr.Code)
		}
	}
}

func TestMoveNote_OutsideVaultAndAlreadyExists(t *testing.T) {
	files := map[string]string{
		"a.md": "Nota A",
		"b.md": "Nota B",
	}

	svc, _, _, _ := createMoveService(t, files)

	// Destino fora do cofre
	_, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From: "a.md",
		To:   "../fora.md",
	})
	if err == nil {
		t.Error("esperava erro para destino fora do cofre")
	}

	// Destino ja existe
	_, err = svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From: "a.md",
		To:   "b.md",
	})
	if err == nil {
		t.Error("esperava erro para destino existente b.md")
	}
}

func TestMoveNote_PreservesAliasAndAnchor(t *testing.T) {
	files := map[string]string{
		"origem.md": "# Origem",
		"ref.md":    "Link [[origem#Secao|Alias]] e [[origem#^bloco]]",
	}

	svc, _, _, root := createMoveService(t, files)

	res, err := svc.MoveNote(context.Background(), service.MoveNoteRequest{
		From:          "origem.md",
		To:            "Pasta/destino.md",
		UpdateLinks:   true,
		CreateFolders: true,
	})
	if err != nil {
		t.Fatalf("MoveNote: %v", err)
	}

	if res.LinksUpdated != 2 {
		t.Errorf("esperava 2 links atualizados, obtido %d", res.LinksUpdated)
	}

	refPath := filepath.Join(root, "ref.md")
	refRaw, _ := os.ReadFile(refPath)
	want := "Link [[destino#Secao|Alias]] e [[destino#^bloco]]"
	if string(refRaw) != want {
		t.Errorf("obtido %q, quer %q", string(refRaw), want)
	}
}
