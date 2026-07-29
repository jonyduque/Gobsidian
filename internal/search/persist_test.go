package search_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

func TestSaveAndLoadInvertedCache(t *testing.T) {
	cacheDir := t.TempDir()
	vaultPath := t.TempDir()

	inv := search.NewInverted()
	inv.Add("a.md", search.Analyze("# Prescrição\n\nTexto sobre prescrição.\n"))
	inv.Add("b.md", search.Analyze("# Outra\n\nTexto sem o termo.\n"))

	ctx := context.Background()
	if err := search.SaveInvertedCache(ctx, cacheDir, vaultPath, inv); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}

	loaded, header, err := search.LoadInvertedCache(ctx, cacheDir, vaultPath)
	if err != nil {
		t.Fatalf("LoadInvertedCache: %v", err)
	}
	if header.NoteCount != 2 {
		t.Errorf("header.NoteCount = %d, want 2", header.NoteCount)
	}

	postings := loaded.Postings("prescricao")
	if len(postings) != 1 || postings[0].Path != "a.md" {
		t.Errorf("Get(prescricao) = %+v, want a.md", postings)
	}
}

// Prova de mutação: desligar a checagem de analyzer_version faz este teste reprovar.
func TestCacheAnalyzerVersionMismatchDiscardsCache(t *testing.T) {
	cacheDir := t.TempDir()
	vaultPath := t.TempDir()

	inv := search.NewInverted()
	inv.Add("a.md", search.Analyze("termo"))

	ctx := context.Background()
	if err := search.SaveInvertedCache(ctx, cacheDir, vaultPath, inv); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}

	cachePath := filepath.Join(cacheDir, "inverted_cache.gob")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	badData := search.InvertedCacheData{
		Header: search.CacheHeader{
			FormatVersion:   search.CacheFormatVersion,
			ParserVersion:   search.CacheParserVersion,
			AnalyzerVersion: 999, // Versão mutada
			VaultPath:       vaultPath,
			NoteCount:       1,
		},
		Terms: inv.ExportTermsForCache(),
	}

	f, err := os.Create(cachePath)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	enc := search.NewEncoderForTest(f)
	if err := enc.Encode(badData); err != nil {
		_ = f.Close()
		t.Fatalf("Encode: %v", err)
	}
	_ = f.Close()

	_, _, err = search.LoadInvertedCache(ctx, cacheDir, vaultPath)
	if err == nil {
		t.Fatal("LoadInvertedCache deveria falhar com versão de analisador diferente (999)")
	}
	if !errors.Is(err, search.ErrCacheVersionMismatch) {
		t.Fatalf("err = %v, want ErrCacheVersionMismatch", err)
	}

	_ = data
}

func TestTruncatedCacheRefused(t *testing.T) {
	cacheDir := t.TempDir()
	vaultPath := t.TempDir()

	cachePath := filepath.Join(cacheDir, "inverted_cache.gob")
	if err := os.WriteFile(cachePath, []byte("tru"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, err := search.LoadInvertedCache(context.Background(), cacheDir, vaultPath)
	if err == nil {
		t.Fatal("LoadInvertedCache deveria falhar com arquivo truncado")
	}
	if !errors.Is(err, search.ErrCacheCorrupted) {
		t.Fatalf("err = %v, want ErrCacheCorrupted", err)
	}
}

func TestEmptyVaultCacheDistinguishableFromMissing(t *testing.T) {
	cacheDir := t.TempDir()
	vaultPath := t.TempDir()

	// 1. Cache inexistente
	_, _, err := search.LoadInvertedCache(context.Background(), cacheDir, vaultPath)
	if !errors.Is(err, search.ErrCacheNotFound) {
		t.Fatalf("Cache inexistente = %v, want ErrCacheNotFound", err)
	}

	// 2. Cache de cofre vazio (0 notas)
	inv := search.NewInverted()
	if err := search.SaveInvertedCache(context.Background(), cacheDir, vaultPath, inv); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}

	loaded, header, err := search.LoadInvertedCache(context.Background(), cacheDir, vaultPath)
	if err != nil {
		t.Fatalf("Cache de cofre vazio falhou: %v", err)
	}
	if header.NoteCount != 0 {
		t.Errorf("header.NoteCount = %d, want 0", header.NoteCount)
	}
	if loaded.DocCount() != 0 {
		t.Errorf("loaded.DocCount = %d, want 0", loaded.DocCount())
	}
}

func TestCacheOutsideVault(t *testing.T) {
	vaultPath := t.TempDir()
	cacheDir := t.TempDir()

	cClean := filepath.Clean(cacheDir)
	vClean := filepath.Clean(vaultPath)
	if strings.HasPrefix(cClean, vClean) {
		t.Fatalf("cacheDir %q está dentro de vaultPath %q", cacheDir, vaultPath)
	}
}

// geraCorpus cria N notas com caminhos DISTINTOS e conteúdo distinto.
func geraCorpus(t *testing.T, n int) (*vault.Vault, *index.Index, *search.Inverted, string) {
	t.Helper()
	root := t.TempDir()
	for i := 0; i < n; i++ {
		dir := filepath.Join(root, fmt.Sprintf("pasta%02d", i%10))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		corpo := fmt.Sprintf("---\ntags: [t%d]\n---\n\n# Nota %d\n\nprescricao intercorrente termo%d civil direito processo\n", i%7, i, i)
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("nota%04d.md", i)), []byte(corpo), 0644); err != nil {
			t.Fatal(err)
		}
	}
	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		data, err := v.ReadAll(context.Background(), p)
		if err != nil {
			t.Fatalf("corpus ilegivel em %s: %v", p, err)
		}
		body, _ := vault.StripBOM(data)
		inv.Add(string(p), search.Analyze(string(body)))
	}

	if got := idx.NoteCount(); got != n {
		t.Fatalf("corpus tem %d notas, quer %d", got, n)
	}
	return v, idx, inv, root
}

func TestQ3PerformanceMeasurement(t *testing.T) {
	const n = 500
	v, idx, inv, _ := geraCorpus(t, n)
	cacheDir := t.TempDir()
	vaultPath := v.Root()
	ctx := context.Background()

	if err := search.SaveInvertedCache(ctx, cacheDir, vaultPath, inv); err != nil {
		t.Fatalf("SaveInvertedCache: %v", err)
	}

	startLoad := time.Now()
	loadedInv, header, err := search.LoadInvertedCache(ctx, cacheDir, vaultPath)
	if err != nil {
		t.Fatalf("LoadInvertedCache: %v", err)
	}
	loadDur := time.Since(startLoad)

	if header.NoteCount != n {
		t.Fatalf("header.NoteCount = %d, quer %d", header.NoteCount, n)
	}
	if loadedInv.DocCount() != n {
		t.Fatalf("loadedInv.DocCount() = %d, quer %d", loadedInv.DocCount(), n)
	}

	startRebuild := time.Now()
	rebuiltInv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		data, err := v.ReadAll(ctx, p)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		body, _ := vault.StripBOM(data)
		rebuiltInv.Add(string(p), search.Analyze(string(body)))
	}
	rebuildDur := time.Since(startRebuild)

	if rebuiltInv.DocCount() != n {
		t.Fatalf("rebuiltInv.DocCount() = %d, quer %d", rebuiltInv.DocCount(), n)
	}

	t.Logf("Q3 Medição em %d notas distintas:", n)
	t.Logf("  (a) LoadInvertedCache (disco): %v", loadDur)
	t.Logf("  (b) Reconstruir Invertido (metadados): %v", rebuildDur)
}

func TestRNF04SearchLatencyPercentile(t *testing.T) {
	const corpusSize = 500
	const numQueries = 200
	_, idx, inv, _ := geraCorpus(t, corpusSize)

	durations := make([]time.Duration, numQueries)
	for i := 0; i < numQueries; i++ {
		termo := fmt.Sprintf("termo%d", i%100)
		tokens := search.Analyze(termo)
		start := time.Now()
		_ = search.CalculateBM25(tokens, inv, idx)
		durations[i] = time.Since(start)
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	minDur := durations[0]
	medianDur := durations[numQueries/2]
	p95Dur := durations[int(float64(numQueries)*0.95)]

	t.Logf("RNF-04 Medição de Latência de Busca (%d consultas em %d notas):", numQueries, corpusSize)
	t.Logf("  Mínimo:  %v", minDur)
	t.Logf("  Mediana: %v", medianDur)
	t.Logf("  p95:     %v", p95Dur)

	if p95Dur > 100*time.Millisecond {
		t.Errorf("p95 %v excede o limite de RNF-04 (100ms)", p95Dur)
	}
}
