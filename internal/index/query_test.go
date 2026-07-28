package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func writeFile2(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func setupQueryIndex(t *testing.T) (*index.Index, *vault.Vault) {
	root := t.TempDir()
	// Create some notes
	writeFile2(t, root, "A.md", "---\ntags: [tag1, tag2/sub]\nautor: joão\nidade: 30\nlista: [a, b, c]\nnested:\n  key: val\n---\n# A\n")
	writeFile2(t, root, "B.md", "---\ntags: [tag1]\nautor: joao\nidade: 25\nlista: [a]\n---\n# B\n")
	writeFile2(t, root, "Folder/C.md", "---\ntags: [tag3]\nautor: MARIA\ndata: 2023-01-01\n---\n# C\n")
	writeFile2(t, root, "Folder/Sub/D.md", "---\nfield_null: null\n---\n# D\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}
	return idx, v
}

func TestQuery(t *testing.T) {
	idx, _ := setupQueryIndex(t)

	t.Run("Empty Query Returns All", func(t *testing.T) {
		res, total := idx.List(index.Query{})
		if total != 4 {
			t.Errorf("Expected 4 total notes, got %d", total)
		}
		if len(res) != 4 {
			t.Errorf("Expected 4 results, got %d", len(res))
		}
	})

	t.Run("Scalar vs Scalar Accent/Case Insensitive", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Frontmatter: map[string]any{"autor": "Joao"},
		})
		if len(res) != 2 {
			t.Errorf("Expected 2 results (joão, joao), got %d", len(res))
		}
	})

	t.Run("Scalar vs List", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Frontmatter: map[string]any{"lista": "b"},
		})
		if len(res) != 1 || res[0].Title != "A" {
			t.Errorf("Expected note A, got %v", res)
		}
	})

	t.Run("List vs List all required", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Frontmatter: map[string]any{"lista": []any{"a", "c"}},
		})
		if len(res) != 1 || res[0].Title != "A" {
			t.Errorf("Expected note A, got %v", res)
		}

		res2, _ := idx.List(index.Query{
			Frontmatter: map[string]any{"lista": []any{"a", "d"}},
		})
		if len(res2) != 0 {
			t.Errorf("Expected 0 notes, got %v", res2)
		}
	})

	t.Run("Null for key presence", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Frontmatter: map[string]any{"field_null": nil},
		})
		if len(res) != 1 || res[0].Title != "D" {
			t.Errorf("Expected D, got %v", res)
		}

		res2, _ := idx.List(index.Query{
			Frontmatter: map[string]any{"autor": nil},
		})
		if len(res2) != 3 {
			t.Errorf("Expected 3 notes with autor, got %d", len(res2))
		}
	})

	t.Run("Missing field", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Frontmatter: map[string]any{"missing": nil},
		})
		if len(res) != 0 {
			t.Errorf("Expected 0 notes, got %d", len(res))
		}
	})

	t.Run("Dotted key navigation", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Frontmatter: map[string]any{"nested.key": "val"},
		})
		if len(res) != 1 || res[0].Title != "A" {
			t.Errorf("Expected note A, got %v", res)
		}
	})

	t.Run("Folder non-recursive", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Folder: "Folder",
		})
		if len(res) != 1 || res[0].Title != "C" {
			t.Errorf("Expected C, got %d results", len(res))
		}
	})

	t.Run("Folder recursive", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Folder:    "Folder",
			Recursive: true,
		})
		if len(res) != 2 {
			t.Errorf("Expected C and D, got %d results", len(res))
		}
	})

	t.Run("Glob *", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Glob: "Folder/*.md",
		})
		if len(res) != 1 || res[0].Title != "C" {
			t.Errorf("Expected C, got %v", res)
		}
	})

	t.Run("Tags hierarchical", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Tags: []string{"tag2"}, // should match tag2/sub
		})
		if len(res) != 1 || res[0].Title != "A" {
			t.Errorf("Expected A, got %v", res)
		}
	})

	t.Run("Tags any", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Tags:    []string{"tag3", "tag2"},
			TagMode: "any",
		})
		if len(res) != 2 {
			t.Errorf("Expected 2 notes, got %d", len(res))
		}
	})

	t.Run("Tags all", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Tags:    []string{"tag1", "tag2"},
			TagMode: "all",
		})
		if len(res) != 1 || res[0].Title != "A" {
			t.Errorf("Expected 1 note, got %v", res)
		}
	})

	t.Run("Pagination total before limit", func(t *testing.T) {
		res, total := idx.List(index.Query{
			Limit:  1,
			Offset: 1,
		})
		if total != 4 {
			t.Errorf("Expected total 4, got %d", total)
		}
		if len(res) != 1 {
			t.Errorf("Expected len 1, got %d", len(res))
		}
	})

	t.Run("Sorting asc", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Sort:  "title",
			Order: "asc",
		})
		if len(res) != 4 || res[0].Title != "A" || res[3].Title != "D" {
			t.Errorf("Expected A ... D, got %s ... %s", res[0].Title, res[3].Title)
		}
	})

	t.Run("Sorting desc", func(t *testing.T) {
		res, _ := idx.List(index.Query{
			Sort:  "title",
			Order: "desc",
		})
		if len(res) != 4 || res[0].Title != "D" || res[3].Title != "A" {
			t.Errorf("Expected D ... A, got %s ... %s", res[0].Title, res[3].Title)
		}
	})
}
