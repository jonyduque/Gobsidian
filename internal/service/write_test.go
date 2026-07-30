package service_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

func setupTestService(t *testing.T) (*service.Service, *vault.Vault, *index.Index) {
	t.Helper()
	dir := t.TempDir()
	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	inv := search.NewInverted()
	svc := service.New(v, idx, inv, nil, service.Options{})
	return svc, v, idx
}

// === note_create ===

func TestCreateNote_Basic(t *testing.T) {
	svc, v, _ := setupTestService(t)
	ctx := context.Background()

	res, err := svc.CreateNote(ctx, service.CreateNoteRequest{
		Path:    "NovaNota.md",
		Content: "Conteudo da nova nota",
	})
	if err != nil {
		t.Fatalf("CreateNote falhou: %v", err)
	}
	if !res.Created {
		t.Errorf("Created = false, quer true")
	}

	raw, err := os.ReadFile(v.Abs(vault.CanonicalPath("NovaNota.md")))
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	if string(raw) != "Conteudo da nova nota" {
		t.Errorf("conteudo no disco = %q, quer 'Conteudo da nova nota'", string(raw))
	}
}

func TestCreateNote_AlreadyExists(t *testing.T) {
	svc, v, _ := setupTestService(t)
	ctx := context.Background()
	_ = os.WriteFile(v.Abs("Existente.md"), []byte("ja existe"), 0644)

	_, err := svc.CreateNote(ctx, service.CreateNoteRequest{
		Path:    "Existente.md",
		Content: "tentativa de sobrescrita",
	})
	if err == nil {
		t.Fatal("esperava erro ao tentar criar nota existente")
	}
	if service.CodeOf(err) != service.CodeNoteExists {
		t.Errorf("Code = %v, quer %v", service.CodeOf(err), service.CodeNoteExists)
	}
}

func TestCreateNote_WithFrontmatter(t *testing.T) {
	svc, v, _ := setupTestService(t)
	ctx := context.Background()

	fm := map[string]any{"autor": "Jony", "tags": []string{"teste"}}
	_, err := svc.CreateNote(ctx, service.CreateNoteRequest{
		Path:        "ComFM.md",
		Content:     "Texto principal",
		Frontmatter: fm,
	})
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	raw, _ := os.ReadFile(v.Abs("ComFM.md"))
	s := string(raw)
	if !strings.Contains(s, "autor: Jony") || !strings.Contains(s, "Texto principal") {
		t.Errorf("frontmatter ou conteudo ausente:\n%s", s)
	}
}

func TestCreateNote_CreateFoldersTrueAndFalse(t *testing.T) {
	svc, _, _ := setupTestService(t)
	ctx := context.Background()

	// 1. CreateFolders = false deve falhar se a pasta nao existir
	_, err := svc.CreateNote(ctx, service.CreateNoteRequest{
		Path:          "SubPasta/Nota.md",
		Content:       "Texto",
		CreateFolders: false,
	})
	if err == nil || service.CodeOf(err) != service.CodeFolderNotFound {
		t.Errorf("esperava CodeFolderNotFound, obteve: %v", err)
	}

	// 2. CreateFolders = true cria a pasta intermediaria
	res, err := svc.CreateNote(ctx, service.CreateNoteRequest{
		Path:          "SubPasta/Nota.md",
		Content:       "Texto",
		CreateFolders: true,
	})
	if err != nil || !res.Created {
		t.Fatalf("CreateNote com CreateFolders=true falhou: %v", err)
	}
}

func TestCreateNote_DryRunPreservesMTimeAndDoesNotWrite(t *testing.T) {
	svc, v, _ := setupTestService(t)
	ctx := context.Background()

	res, err := svc.CreateNote(ctx, service.CreateNoteRequest{
		Path:    "DryRunNota.md",
		Content: "Conteudo simulado",
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("CreateNote dry_run: %v", err)
	}
	if res.Created {
		t.Error("Created deve ser false em dry_run")
	}
	if res.Diff == "" {
		t.Error("Diff deve vir preenchido em dry_run")
	}

	if _, err := os.Stat(v.Abs("DryRunNota.md")); !os.IsNotExist(err) {
		t.Error("arquivo foi criado no disco durante dry_run")
	}
}

// === note_append ===

func TestAppendNote_Basic(t *testing.T) {
	svc, v, idx := setupTestService(t)
	ctx := context.Background()
	_ = os.WriteFile(v.Abs("Base.md"), []byte("conteudo inicial\r\n"), 0644)
	if err := idx.Build(ctx, v); err != nil {
		t.Fatal(err)
	}

	res, err := svc.AppendNote(ctx, service.AppendNoteRequest{
		Path:    "Base.md",
		Content: "conteudo anexado",
	})
	if err != nil || !res.Appended {
		t.Fatalf("AppendNote falhou: %v", err)
	}

	raw, _ := os.ReadFile(v.Abs("Base.md"))
	if !strings.Contains(string(raw), "conteudo inicial\r\nconteudo anexado") {
		t.Errorf("conteudo no disco = %q", string(raw))
	}
}

func TestAppendNote_ToHeading(t *testing.T) {
	svc, v, idx := setupTestService(t)
	ctx := context.Background()
	_ = os.WriteFile(v.Abs("Secoes.md"), []byte("# Topo\n\n## Cap 1\n\ntexto 1\n\n## Cap 2\n\ntexto 2\n"), 0644)
	if err := idx.Build(ctx, v); err != nil {
		t.Fatal(err)
	}

	_, err := svc.AppendNote(ctx, service.AppendNoteRequest{
		Path:    "Secoes.md",
		Content: "anexo no cap 1",
		Heading: "Cap 1",
	})
	if err != nil {
		t.Fatalf("AppendNote no heading falhou: %v", err)
	}

	raw, _ := os.ReadFile(v.Abs("Secoes.md"))
	s := string(raw)
	pos1 := strings.Index(s, "texto 1")
	posAnexo := strings.Index(s, "anexo no cap 1")
	posCap2 := strings.Index(s, "## Cap 2")

	if pos1 == -1 || posAnexo == -1 || posCap2 == -1 {
		t.Fatalf("trecho ausente no resultado:\n%s", s)
	}
	if pos1 >= posAnexo || posAnexo >= posCap2 {
		t.Errorf("ordem incorreta dos trechos: pos1=%d, posAnexo=%d, posCap2=%d:\n%s", pos1, posAnexo, posCap2, s)
	}
}

func TestAppendNote_CreateIfMissingTrueAndFalse(t *testing.T) {
	svc, v, idx := setupTestService(t)
	ctx := context.Background()
	_ = os.WriteFile(v.Abs("Secoes2.md"), []byte("# Topo\n"), 0644)
	if err := idx.Build(ctx, v); err != nil {
		t.Fatal(err)
	}

	// CreateIfMissing = false
	_, err := svc.AppendNote(ctx, service.AppendNoteRequest{
		Path:            "Secoes2.md",
		Content:         "texto",
		Heading:         "Cap Inexistente",
		CreateIfMissing: false,
	})
	if err == nil || service.CodeOf(err) != service.CodeHeadingNotFound {
		t.Errorf("esperava CodeHeadingNotFound, obteve: %v", err)
	}

	// CreateIfMissing = true
	_, err = svc.AppendNote(ctx, service.AppendNoteRequest{
		Path:            "Secoes2.md",
		Content:         "conteudo novo",
		Heading:         "Cap Inexistente",
		HeadingLevel:    2,
		CreateIfMissing: true,
	})
	if err != nil {
		t.Fatalf("AppendNote com CreateIfMissing=true falhou: %v", err)
	}

	raw, _ := os.ReadFile(v.Abs("Secoes2.md"))
	s := string(raw)
	if !strings.Contains(s, "## Cap Inexistente\nconteudo novo") {
		t.Errorf("heading novo nao foi criado:\n%s", s)
	}
}

func TestAppendNote_ExpectedHashMatchAndMismatch(t *testing.T) {
	svc, v, idx := setupTestService(t)
	ctx := context.Background()
	_ = os.WriteFile(v.Abs("HashNote.md"), []byte("conteudo"), 0644)
	_ = idx.Build(ctx, v)

	note, _ := idx.Get("HashNote.md")
	correctHash := fmt.Sprintf("%016x", note.Hash)

	// Hash mismatch
	_, err := svc.AppendNote(ctx, service.AppendNoteRequest{
		Path:         "HashNote.md",
		Content:      "texto",
		ExpectedHash: "hash_errado",
	})
	if err == nil || service.CodeOf(err) != service.CodeHashMismatch {
		t.Errorf("esperava CodeHashMismatch, obteve: %v", err)
	}

	// Hash correto
	_, err = svc.AppendNote(ctx, service.AppendNoteRequest{
		Path:         "HashNote.md",
		Content:      "texto novo",
		ExpectedHash: correctHash,
	})
	if err != nil {
		t.Fatalf("AppendNote com hash correto falhou: %v", err)
	}
}

func TestAppendNote_DryRunPreservesMTime(t *testing.T) {
	svc, v, idx := setupTestService(t)
	ctx := context.Background()
	p := v.Abs("DryAppend.md")
	_ = os.WriteFile(p, []byte("original\n"), 0644)
	_ = idx.Build(ctx, v)

	fiBefore, _ := os.Stat(p)
	time.Sleep(50 * time.Millisecond)

	res, err := svc.AppendNote(ctx, service.AppendNoteRequest{
		Path:    "DryAppend.md",
		Content: "anexo dry run",
		DryRun:  true,
	})
	if err != nil || res.Appended {
		t.Fatalf("AppendNote dry_run: %v", err)
	}

	fiAfter, _ := os.Stat(p)
	if !fiBefore.ModTime().Equal(fiAfter.ModTime()) {
		t.Error("mtime do arquivo foi alterado durante dry_run")
	}
}

// === note_patch ===

func TestPatchNote_ReplaceSection(t *testing.T) {
	svc, v, idx := setupTestService(t)
	ctx := context.Background()
	_ = os.WriteFile(v.Abs("PatchNote.md"), []byte("# Topo\n\n## Secao\n\nconteudo antigo\n"), 0644)
	_ = idx.Build(ctx, v)

	res, err := svc.PatchNote(ctx, service.PatchNoteRequest{
		Path:    "PatchNote.md",
		Heading: "Secao",
		Content: "conteudo novo",
		Mode:    "replace_section",
	})
	if err != nil || !res.Patched {
		t.Fatalf("PatchNote falhou: %v", err)
	}

	raw, _ := os.ReadFile(v.Abs("PatchNote.md"))
	s := string(raw)
	if !strings.Contains(s, "conteudo novo") || strings.Contains(s, "conteudo antigo") {
		t.Errorf("patch de secao falhou:\n%s", s)
	}
}

func TestPatchNote_ReplaceHeadingAndSectionMode(t *testing.T) {
	svc, v, idx := setupTestService(t)
	ctx := context.Background()
	_ = os.WriteFile(v.Abs("PatchHeading.md"), []byte("# Topo\n\n## Titulo Antigo\n\nconteudo\n"), 0644)
	_ = idx.Build(ctx, v)

	_, err := svc.PatchNote(ctx, service.PatchNoteRequest{
		Path:    "PatchHeading.md",
		Heading: "Titulo Antigo",
		Content: "## Titulo Novo\n\nconteudo",
		Mode:    "replace_heading_and_section",
	})
	if err != nil {
		t.Fatalf("PatchNote replace_heading_and_section falhou: %v", err)
	}

	raw, _ := os.ReadFile(v.Abs("PatchHeading.md"))
	s := string(raw)
	if strings.Contains(s, "Titulo Antigo") || !strings.Contains(s, "## Titulo Novo") {
		t.Errorf("heading antigo nao foi substituido:\n%s", s)
	}
}

func TestPatchNote_ReplaceBlock(t *testing.T) {
	svc, v, idx := setupTestService(t)
	ctx := context.Background()
	_ = os.WriteFile(v.Abs("PatchBlock.md"), []byte("Paragrafo velho ^bloco1\n"), 0644)
	_ = idx.Build(ctx, v)

	_, err := svc.PatchNote(ctx, service.PatchNoteRequest{
		Path:    "PatchBlock.md",
		BlockID: "bloco1",
		Content: "Paragrafo novo",
		Mode:    "replace_block",
	})
	if err != nil {
		t.Fatalf("PatchNote replace_block falhou: %v", err)
	}

	raw, _ := os.ReadFile(v.Abs("PatchBlock.md"))
	s := string(raw)
	if !strings.Contains(s, "Paragrafo novo ^bloco1") {
		t.Errorf("bloco nao foi substituido:\n%s", s)
	}
}

func TestPatchNote_ExpectedHashMatchAndMismatch(t *testing.T) {
	svc, v, idx := setupTestService(t)
	ctx := context.Background()
	_ = os.WriteFile(v.Abs("PatchHash.md"), []byte("## S\n\ntexto\n"), 0644)
	_ = idx.Build(ctx, v)

	note, _ := idx.Get("PatchHash.md")

	_, err := svc.PatchNote(ctx, service.PatchNoteRequest{
		Path:         "PatchHash.md",
		Heading:      "S",
		Content:      "novo",
		ExpectedHash: "invalido",
	})
	if err == nil || service.CodeOf(err) != service.CodeHashMismatch {
		t.Errorf("esperava CodeHashMismatch, obteve: %v", err)
	}

	_, err = svc.PatchNote(ctx, service.PatchNoteRequest{
		Path:         "PatchHash.md",
		Heading:      "S",
		Content:      "novo",
		ExpectedHash: fmt.Sprintf("%016x", note.Hash),
	})
	if err != nil {
		t.Fatalf("PatchNote com hash correto falhou: %v", err)
	}
}

func TestPatchNote_DryRunPreservesMTime(t *testing.T) {
	svc, v, idx := setupTestService(t)
	ctx := context.Background()
	p := v.Abs("DryPatch.md")
	_ = os.WriteFile(p, []byte("## S\n\ntexto\n"), 0644)
	_ = idx.Build(ctx, v)

	fiBefore, _ := os.Stat(p)
	time.Sleep(50 * time.Millisecond)

	res, err := svc.PatchNote(ctx, service.PatchNoteRequest{
		Path:    "DryPatch.md",
		Heading: "S",
		Content: "novo conteudo",
		DryRun:  true,
	})
	if err != nil || res.Patched {
		t.Fatalf("PatchNote dry_run falhou: %v", err)
	}

	fiAfter, _ := os.Stat(p)
	if !fiBefore.ModTime().Equal(fiAfter.ModTime()) {
		t.Error("mtime do arquivo foi alterado durante dry_run em note_patch")
	}
}
