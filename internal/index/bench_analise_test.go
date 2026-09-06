package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func cofreReal(b *testing.B) (*index.Index, string) {
	b.Helper()
	dir := benchIndexVaultPath(b)
	v, err := vault.New(dir)
	if err != nil {
		b.Fatalf("vault.New: %v", err)
	}
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		b.Fatalf("Build: %v", err)
	}
	return ix, dir
}

// BenchmarkSaveIndexCacheReal mede a gravacao do cache de indice como ela e
// hoje: via vault.ReplaceFile, com fsync do arquivo e do diretorio (Task 172).
func BenchmarkSaveIndexCacheReal(b *testing.B) {
	ix, dir := cofreReal(b)
	cacheDir := b.TempDir()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := index.SaveIndexCache(context.Background(), cacheDir, dir, ix); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSaveIndexCacheRealComFsync soma um SEGUNDO Sync por fora, depois do
// que BenchmarkSaveIndexCacheReal ja faz via vault.ReplaceFile (Task 172).
// Sobrevive so como referencia historica de antes da Task 172.
func BenchmarkSaveIndexCacheRealComFsync(b *testing.B) {
	ix, dir := cofreReal(b)
	cacheDir := b.TempDir()
	final := filepath.Join(cacheDir, "index_cache.gob")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := index.SaveIndexCache(context.Background(), cacheDir, dir, ix); err != nil {
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

// BenchmarkTagsSemPrefixo mede ix.Tags como tag_list a chama: ToLower por
// chave a cada chamada.
func BenchmarkTagsSemPrefixo(b *testing.B) {
	ix, _ := cofreReal(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if len(ix.Tags("", 1)) == 0 {
			b.Fatal("sem tags")
		}
	}
}
