package service_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

func createSearchService(t *testing.T, files map[string]string) (*service.Service, *vault.Vault, *index.Index, *search.Inverted) {
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
	for _, p := range idx.NotePaths() {
		if data, err := v.ReadAll(context.Background(), p); err == nil {
			body, _ := vault.StripBOM(data)
			inv.Add(string(p), search.Analyze(string(body)))
		}
	}

	svc := service.New(v, idx, inv, nil, service.Options{})
	return svc, v, idx, inv
}

// 1. Parameter `query`
func TestVaultSearchQuery(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "# Prescrição\n\nTexto sobre prescrição.\n",
		"b.md": "# Outra\n\nTexto qualquer sem o termo.\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 1 || res.Results[0].Path != "a.md" {
		t.Fatalf("Query filter failed: res = %+v", res)
	}
}

// 2. Parameter `folder`
func TestVaultSearchFolder(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"Civil/a.md": "# Prescrição\n\nTexto civil.\n",
		"Penal/b.md": "# Prescrição\n\nTexto penal.\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao", Folder: "Civil"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 1 || res.Results[0].Path != "Civil/a.md" {
		t.Fatalf("Folder filter failed: res = %+v", res)
	}
}

// 3. Parameter `tags`
func TestVaultSearchTags(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "---\ntags: [urgente]\n---\n# Prescrição\n\nTexto.\n",
		"b.md": "---\ntags: [arquivado]\n---\n# Prescrição\n\nTexto.\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao", Tags: []string{"urgente"}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 1 || res.Results[0].Path != "a.md" {
		t.Fatalf("Tags filter failed: res = %+v", res)
	}
}

// 4. Parameter `frontmatter`
func TestVaultSearchFrontmatter(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "---\nautor: Silva\n---\n# Prescrição\n\nTexto.\n",
		"b.md": "---\nautor: Souza\n---\n# Prescrição\n\nTexto.\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{
		Query:       "prescricao",
		Frontmatter: map[string]any{"autor": "Silva"},
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 1 || res.Results[0].Path != "a.md" {
		t.Fatalf("Frontmatter filter failed: res = %+v", res)
	}
}

// 5. Parameter `modified_after`
func TestVaultSearchModifiedAfter(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "# Prescrição\n\nTexto.\n",
	})

	past := time.Now().Add(-2 * time.Hour)
	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao", ModifiedAfter: &past})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 1 {
		t.Fatalf("ModifiedAfter filter failed: res = %+v", res)
	}

	future := time.Now().Add(2 * time.Hour)
	res2, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao", ModifiedAfter: &future})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res2.Results) != 0 {
		t.Fatalf("ModifiedAfter future filter failed: res = %+v", res2)
	}
}

// 6. Parameter `modified_before`
func TestVaultSearchModifiedBefore(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "# Prescrição\n\nTexto.\n",
	})

	future := time.Now().Add(2 * time.Hour)
	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao", ModifiedBefore: &future})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 1 {
		t.Fatalf("ModifiedBefore filter failed: res = %+v", res)
	}

	past := time.Now().Add(-2 * time.Hour)
	res2, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao", ModifiedBefore: &past})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res2.Results) != 0 {
		t.Fatalf("ModifiedBefore past filter failed: res = %+v", res2)
	}
}

// 7. Parameter `snippet_chars`
func TestVaultSearchSnippetChars(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "# Prescrição\n\n" + strings.Repeat("palavra ", 50) + "prescrição " + strings.Repeat("palavra ", 50),
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao", SnippetChars: 40})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 1 {
		t.Fatalf("len(res) = %d, quer 1", len(res.Results))
	}
	if len(res.Results[0].Snippet) > 40 {
		t.Errorf("SnippetChars = %d, quer <= 40", len(res.Results[0].Snippet))
	}
}

// 8. Parameter `limit`
func TestVaultSearchLimit(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "# Prescrição\n",
		"b.md": "# Prescrição\n",
		"c.md": "# Prescrição\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao", Limit: 2})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 2 || res.Total != 3 || !res.Truncated {
		t.Fatalf("Limit failed: len = %d, total = %d, truncated = %v", len(res.Results), res.Total, res.Truncated)
	}
}

// 9. Parameter `offset`
func TestVaultSearchOffset(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "# Prescrição\n",
		"b.md": "# Prescrição\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao", Offset: 1, Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 1 || res.Total != 2 {
		t.Fatalf("Offset failed: len = %d, total = %d", len(res.Results), res.Total)
	}
}

// Check 6 return fields
func TestVaultSearchReturnFields(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"Civil/a.md": "---\ntitle: Nota Exemplo\n---\n## Seção 1\n\nPrescrição intercorrente.\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 1 {
		t.Fatalf("len(res) = %d, quer 1", len(res.Results))
	}

	hit := res.Results[0]

	// Field 1: path
	if hit.Path != "Civil/a.md" {
		t.Errorf("Field path = %q, quer %q", hit.Path, "Civil/a.md")
	}
	// Field 2: title
	if hit.Title != "Nota Exemplo" {
		t.Errorf("Field title = %q, quer %q", hit.Title, "Nota Exemplo")
	}
	// Field 3: score
	if hit.Score <= 0 {
		t.Errorf("Field score = %f, quer > 0", hit.Score)
	}
	// Field 4: snippet
	if !strings.Contains(hit.Snippet, "intercorrente") {
		t.Errorf("Field snippet = %q, quer conter intercorrente", hit.Snippet)
	}
	// Field 5: matched_headings
	if len(hit.MatchedHeadings) == 0 || hit.MatchedHeadings[0] != "Seção 1" {
		t.Errorf("Field matched_headings = %v, quer [Seção 1]", hit.MatchedHeadings)
	}
	// Field 6: modified
	if hit.Modified == "" {
		t.Errorf("Field modified está vazio")
	}
}

// Query vazia com filtros -> redireciona para metadados (RF-25)
func TestVaultSearchEmptyQueryMetadataOnly(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"Civil/a.md": "# A\n",
		"Penal/b.md": "# B\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "", Folder: "Civil"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 1 || res.Results[0].Path != "Civil/a.md" {
		t.Fatalf("Empty query metadata search failed: %+v", res)
	}
	if res.Results[0].Score != 0.0 {
		t.Errorf("Empty query metadata search score = %f, quer 0.0", res.Results[0].Score)
	}
}

// Busca por frase exata com aspas duplas (RF-24)
func TestVaultSearchPhraseQueryExactSequence(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "A prescricao intercorrente corre rapido.\n",
		"b.md": "A prescricao e intercorrente na lei.\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: `"prescricao intercorrente"`})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 1 || res.Results[0].Path != "a.md" {
		t.Fatalf("Phrase search failed: res = %+v, quer apenas a.md", res)
	}
}

// Consulta sem resultados devolve results: [] e total 0 sem erro
func TestVaultSearchEmptyResultsNoError(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "# A\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "inexistente"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.Results == nil || len(res.Results) != 0 || res.Total != 0 {
		t.Fatalf("Empty results failed: res = %+v", res)
	}
}

// Offset além do fim devolve lista vazia e total correto
func TestVaultSearchOffsetOutOfBounds(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "# Prescrição\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "prescricao", Offset: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Results) != 0 || res.Total != 1 {
		t.Fatalf("Offset out of bounds failed: res = %+v", res)
	}
}
