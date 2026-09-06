package search_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

func invertidoReal(b *testing.B) (*search.Inverted, string) {
	b.Helper()
	dir := os.Getenv("GOBSIDIAN_BENCH_VAULT")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "vault_5000")
	}
	if _, err := os.Stat(dir); err != nil {
		b.Skipf("cofre de benchmark ausente em %s", dir)
	}
	ctx := context.Background()
	v, err := vault.New(dir)
	if err != nil {
		b.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(ctx, v); err != nil {
		b.Fatalf("idx.Build: %v", err)
	}
	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		if err := inv.Update(ctx, v, p); err != nil {
			b.Fatalf("inv.Update %s: %v", p, err)
		}
	}
	return inv, dir
}

// BenchmarkSaveInvertedCacheReal mede a gravacao do cache invertido como ela e
// hoje: via vault.ReplaceFile, com fsync do arquivo e do diretorio (Task 172).
func BenchmarkSaveInvertedCacheReal(b *testing.B) {
	inv, dir := invertidoReal(b)
	cacheDir := b.TempDir()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := search.SaveInvertedCache(context.Background(), cacheDir, dir, inv); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSaveInvertedCacheRealComFsync soma um SEGUNDO Sync por fora, depois
// do que BenchmarkSaveInvertedCacheReal ja faz via vault.ReplaceFile (Task 172).
// Sobrevive so como referencia historica de antes da Task 172.
func BenchmarkSaveInvertedCacheRealComFsync(b *testing.B) {
	inv, dir := invertidoReal(b)
	cacheDir := b.TempDir()
	final := filepath.Join(cacheDir, "inverted_cache.gob")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := search.SaveInvertedCache(context.Background(), cacheDir, dir, inv); err != nil {
			b.Fatal(err)
		}
		f, err := os.OpenFile(final, os.O_RDWR, 0)
		if err != nil {
			b.Fatal(err)
		}
		if err := f.Sync(); err != nil {
			b.Fatal(err)
		}
		_ = f.Close()
	}
}
