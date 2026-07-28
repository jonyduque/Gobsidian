package service

import (
	"context"
	"testing"
)

func TestService_LinkGraph(t *testing.T) {
	svc := setupGraphTest(t)

	t.Run("basic graph traversal", func(t *testing.T) {
		res, err := svc.LinkGraph(context.Background(), GraphRequest{
			Path:  "a.md",
			Depth: 2,
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Nodes) == 0 {
			t.Errorf("expected nodes in graph, got 0")
		}
	})
}

func TestService_TagList(t *testing.T) {
	svc := setupGraphTest(t)

	t.Run("list tags", func(t *testing.T) {
		res, err := svc.TagList(context.Background(), TagRequest{
			Prefix:   "",
			MinCount: 1,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Tags) == 0 {
			t.Errorf("expected tags in vault, got 0")
		}
	})
}

func TestService_ListNotes(t *testing.T) {
	svc := setupGraphTest(t)

	t.Run("list all notes", func(t *testing.T) {
		res, err := svc.ListNotes(context.Background(), ListRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Total == 0 {
			t.Errorf("expected notes, got 0")
		}
	})
}

func TestService_NoteMetadata(t *testing.T) {
	svc := setupGraphTest(t)

	t.Run("get note metadata", func(t *testing.T) {
		res, err := svc.NoteMetadata(context.Background(), MetadataRequest{
			Path: "a.md",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Path != "a.md" {
			t.Errorf("expected a.md, got %s", res.Path)
		}
	})
}

func TestService_VaultStats(t *testing.T) {
	svc := setupGraphTest(t)

	t.Run("full stats", func(t *testing.T) {
		res, err := svc.VaultStats(context.Background(), StatsRequest{
			IncludeRuntime: true,
			IncludeHealth:  true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Notes == 0 {
			t.Errorf("expected notes to be counted")
		}
		if res.Runtime == nil {
			t.Errorf("expected runtime stats")
		}
	})
}

func setupGraphTest(t *testing.T) *Service {
	root := t.TempDir()
	writeFile(t, root, "a.md", "---\ntitle: a\ntags: [x]\n---\n[[b]]\n")
	writeFile(t, root, "b.md", "---\ntitle: b\n---\n[[a]]\n")

	return newTestService(t, root)
}

// O parametro fields era declarado no schema de note_list e ignorado: o modelo
// pedia tres campos, recebia o frontmatter inteiro dentro do *index.Note
// completo, e nao tinha como saber que o pedido nao fizera nada.
func TestListNotesProjectsRequestedFields(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md",
		"---\ntitulo: Ponto 3\nautor: Fulano\nano: 2026\nsigilo: interno\n---\n# A\n\ntexto\n")
	svc := newTestService(t, root)

	res, err := svc.ListNotes(context.Background(), ListRequest{
		Fields: []string{"autor", "ano", "inexistente"},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(res.Notes) != 1 {
		t.Fatalf("notas = %d, quer 1", len(res.Notes))
	}

	item := res.Notes[0]
	if item.Fields["autor"] != "Fulano" {
		t.Errorf("autor = %v, quer Fulano", item.Fields["autor"])
	}
	if item.Fields["ano"] != 2026 {
		t.Errorf("ano = %v (%T), quer 2026", item.Fields["ano"], item.Fields["ano"])
	}
	// Campo nao pedido nao vaza. Sem esta asercao, devolver o frontmatter
	// inteiro passaria no teste.
	if _, vazou := item.Fields["sigilo"]; vazou {
		t.Error("campo nao pedido vazou no retorno")
	}
	// Campo pedido e inexistente simplesmente nao aparece.
	if _, inventou := item.Fields["inexistente"]; inventou {
		t.Error("campo inexistente foi inventado no retorno")
	}
}

// Sem fields, note_list nao devolve frontmatter nenhum. E a tool barata: quem
// quer os metadados completos de uma nota chama note_metadata.
func TestListNotesOmitsFrontmatterByDefault(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "---\nautor: Fulano\n---\n# A\n")
	svc := newTestService(t, root)

	res, err := svc.ListNotes(context.Background(), ListRequest{})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(res.Notes) != 1 {
		t.Fatalf("notas = %d, quer 1", len(res.Notes))
	}
	if len(res.Notes[0].Fields) != 0 {
		t.Errorf("Fields = %v, quer vazio sem pedido explicito", res.Notes[0].Fields)
	}
	if res.Notes[0].Path != "A.md" || res.Notes[0].Hash == "" {
		t.Errorf("projecao incompleta: %+v", res.Notes[0])
	}
}
