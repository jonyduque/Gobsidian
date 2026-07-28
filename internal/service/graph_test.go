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
