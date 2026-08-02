package service_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

func getVault5000Path(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "vault_5000")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("cofre de 5000 notas nao encontrado em %s; gere com gen_vault.ps1", dir)
	}
	return dir
}

func TestScale5000_RNF01_RNF02_RNF07_RNF04(t *testing.T) {
	vaultDir := getVault5000Path(t)

	// 1. RNF-01: Indexação a frio (5 execuções)
	var rnf01Times []time.Duration
	for i := 0; i < 5; i++ {
		v, err := vault.New(vaultDir)
		if err != nil {
			t.Fatalf("vault.New: %v", err)
		}
		idx := index.New()
		start := time.Now()
		if err := idx.Build(context.Background(), v); err != nil {
			t.Fatalf("idx.Build: %v", err)
		}
		dur := time.Since(start)
		if idx.NoteCount() != 5000 {
			t.Fatalf("NoteCount = %d, quer 5000", idx.NoteCount())
		}
		rnf01Times = append(rnf01Times, dur)
	}

	sort.Slice(rnf01Times, func(i, j int) bool { return rnf01Times[i] < rnf01Times[j] })
	t.Logf("=== RNF-01 (Indexacao a Frio 5.000 notas) ===")
	for i, d := range rnf01Times {
		t.Logf("  Rodada %d: %v", i+1, d)
	}
	t.Logf("  Min: %v, Mediana: %v, Max: %v", rnf01Times[0], rnf01Times[2], rnf01Times[4])

	// 2. RNF-02: Boot com cache válido (5 execuções)
	v, _ := vault.New(vaultDir)
	idx := index.New()
	_ = idx.Build(context.Background(), v)
	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		_ = inv.Update(context.Background(), v, p)
	}
	cacheDir := t.TempDir()
	if err := search.SaveInvertedCache(context.Background(), cacheDir, vaultDir, inv); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}

	var rnf02Times []time.Duration
	for i := 0; i < 5; i++ {
		start := time.Now()
		_, _, err := search.LoadInvertedCache(context.Background(), cacheDir, vaultDir)
		if err != nil {
			t.Fatalf("LoadInvertedCache: %v", err)
		}
		dur := time.Since(start)
		rnf02Times = append(rnf02Times, dur)
	}
	sort.Slice(rnf02Times, func(i, j int) bool { return rnf02Times[i] < rnf02Times[j] })
	t.Logf("=== RNF-02 (Boot com Cache Valido 5.000 notas) ===")
	for i, d := range rnf02Times {
		t.Logf("  Rodada %d: %v", i+1, d)
	}
	t.Logf("  Min: %v, Mediana: %v, Max: %v", rnf02Times[0], rnf02Times[2], rnf02Times[4])

	// 3. RNF-07: RSS em repouso
	var mem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&mem)
	t.Logf("=== RNF-07 (RSS em repouso 5.000 notas) ===")
	t.Logf("  Alloc: %.2f MB, Sys: %.2f MB", float64(mem.Alloc)/1024/1024, float64(mem.Sys)/1024/1024)

	// 4. RNF-04: Latência de vault_search p95 a 5.000 notas
	svc := service.New(v, idx, inv, nil, service.Options{})
	t.Logf("=== RNF-04 (Latencia vault_search p95 5.000 notas) ===")

	queries := []struct {
		nome string
		opts service.SearchOptions
	}{
		{"termo amplo, limit default", service.SearchOptions{Query: "nota"}},
		{"dois termos", service.SearchOptions{Query: "servidor mcp"}},
		{"termo seletivo", service.SearchOptions{Query: "Acentuada"}},
		{"filtro de pasta", service.SearchOptions{Query: "nota", Folder: "Projetos"}},
		{"filtro de tag", service.SearchOptions{Query: "nota", Tags: []string{"golang"}}},
		{"frase exata", service.SearchOptions{Query: `"algoritmo BM25 com pesos"`}},
		{"trecho maximo", service.SearchOptions{Query: "comportamento do watcher", SnippetChars: 1000}},
		{"limit maximo do schema", service.SearchOptions{Query: "nota", Limit: 200}},
	}

	for _, q := range queries {
		var lats []time.Duration
		for k := 0; k < 30; k++ {
			st := time.Now()
			_, err := svc.Search(context.Background(), q.opts)
			if err != nil {
				t.Fatalf("Search err: %v", err)
			}
			lats = append(lats, time.Since(st))
		}
		sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })
		mediana := lats[15]
		p95 := lats[int(float64(len(lats))*0.95)-1]
		t.Logf("  %-30s mediana %-10v p95 %-10v", q.nome, mediana, p95)
	}
}
