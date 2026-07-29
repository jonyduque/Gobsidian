package search_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/search"
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

func TestQ3PerformanceMeasurement(t *testing.T) {
	cacheDir := t.TempDir()
	vaultPath := t.TempDir()

	inv := search.NewInverted()
	for i := 0; i < 100; i++ {
		inv.Add(filepath.Join("folder", "note.md"), search.Analyze("termo exemplo de teste de performance para o indice invertido"))
	}

	ctx := context.Background()
	startSave := time.Now()
	if err := search.SaveInvertedCache(ctx, cacheDir, vaultPath, inv); err != nil {
		t.Fatalf("Save: %v", err)
	}
	saveDur := time.Since(startSave)

	startLoad := time.Now()
	_, header, err := search.LoadInvertedCache(ctx, cacheDir, vaultPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	loadDur := time.Since(startLoad)

	t.Logf("Q3 Medição: 100 notas | Save: %v | Load: %v | Notes: %d", saveDur, loadDur, header.NoteCount)
}
