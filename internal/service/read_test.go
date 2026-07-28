package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(root, name), []byte(content), 0644)
	if err != nil {
		t.Fatalf("escrevendo %s: %v", name, err)
	}
}

func newTestService(t *testing.T, root string) *Service {
	t.Helper()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New(%q): %v", root, err)
	}

	// How to build an index for tests? We should probably just use index.Build
	ix := index.New()
	err = ix.Build(context.Background(), v)
	if err != nil {
		t.Fatalf("index.Build: %v", err)
	}

	// I need to use the interface Index if we didn't add the methods to Index yet.
	// We'll update the Index interface in service.go.
	return New(v, ix, Options{})
}

func TestReadNoteSection(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "---\ntitle: A\n---\n# Titulo\n\n## Cap 1\n\ntexto um\n\n## Cap 2\n\ntexto dois\n")

	svc := newTestService(t, root)

	res, err := svc.ReadNote(context.Background(), ReadRequest{
		Path:    "A.md",
		Heading: "Cap 1",
	})
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}

	if !strings.Contains(string(res.Content), "texto um") {
		t.Errorf("conteudo nao contem a secao: %q", res.Content)
	}
	if strings.Contains(string(res.Content), "texto dois") {
		t.Errorf("conteudo vazou para a secao seguinte: %q", res.Content)
	}
	if res.Hash == "" {
		t.Error("Hash vazio — expected_hash depende dele")
	}
	if res.Section == nil || res.Section.Level != 2 {
		t.Errorf("Section = %+v, quer nivel 2", res.Section)
	}
}

func TestReadNoteHeadingNotFoundListsAlternatives(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# T\n\n## Cap 115\n\n## Cap 116\n\n## Cap 117\n")

	svc := newTestService(t, root)

	_, err := svc.ReadNote(context.Background(), ReadRequest{Path: "A.md", Heading: "Cap 118"})
	if err == nil {
		t.Fatal("heading inexistente deveria falhar")
	}
	if CodeOf(err) != CodeHeadingNotFound {
		t.Errorf("codigo = %v, quer HEADING_NOT_FOUND", CodeOf(err))
	}
	for _, want := range []string{"Cap 115", "Cap 116", "Cap 117"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("mensagem nao lista %q: %s", want, err.Error())
		}
	}
}

func TestReadNoteAcceptsAccentInsensitiveHeading(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# T\n\n## Capítulo 118\n\ntexto\n")

	svc := newTestService(t, root)

	for _, query := range []string{"Capítulo 118", "capitulo 118", "CAPITULO 118"} {
		res, err := svc.ReadNote(context.Background(), ReadRequest{Path: "A.md", Heading: query})
		if err != nil {
			t.Errorf("ReadNote(%q): %v", query, err)
			continue
		}
		if !strings.Contains(string(res.Content), "texto") {
			t.Errorf("ReadNote(%q) nao trouxe a secao", query)
		}
	}
}

func TestReadNoteRejectsTraversal(t *testing.T) {
	svc := newTestService(t, t.TempDir())

	_, err := svc.ReadNote(context.Background(), ReadRequest{Path: "../fora.md"})
	if CodeOf(err) != CodePathOutsideVault {
		t.Errorf("codigo = %v, quer PATH_OUTSIDE_VAULT", CodeOf(err))
	}
}

func TestReadNoteCloudOnlyFails(t *testing.T) {
	t.Skip("requer arquivo com FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS")
}
