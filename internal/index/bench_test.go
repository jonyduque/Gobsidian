package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// benchIndexVaultPath devolve a raiz do cofre sintetico de 5.000 notas usado
// pelos benchmarks de RNF. Mesma convencao de internal/service/bench_test.go
// e internal/service/rnf5000_test.go: GOBSIDIAN_BENCH_VAULT para o CI, senao
// o caminho fixo em os.TempDir() que scripts/gen_vault.ps1 produz.
func benchIndexVaultPath(b *testing.B) string {
	b.Helper()
	dir := os.Getenv("GOBSIDIAN_BENCH_VAULT")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "vault_5000")
	}
	if _, err := os.Stat(dir); err != nil {
		b.Skipf("cofre de benchmark ausente em %s; gere com "+
			"scripts/gen_vault.ps1 -Notes 5000 -Seed 42 -Out <caminho>", dir)
	}
	return dir
}

// BenchmarkReplaceSingleFile mede o custo de reindexar UM arquivo — o
// caminho que o watcher percorre a cada evento fsnotify, e o que RNF-06
// limita a 20 ms. Ate a Task 86 nao havia benchmark nenhum deste caminho:
// so a medicao ad-hoc registrada em docs/OPERACAO.md (20,35 ms de mediana).
//
// O arquivo escolhido tem links de saida (para exercitar resolveTarget) e e
// citado por outras notas do cofre sintetico (para exercitar o reprocessamento
// dos links que apontam para ele) — sem isso o benchmark mediria so a
// parte barata do caminho.
func BenchmarkReplaceSingleFile(b *testing.B) {
	dir := benchIndexVaultPath(b)
	v, err := vault.New(dir)
	if err != nil {
		b.Fatalf("vault.New: %v", err)
	}

	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		b.Fatalf("Build: %v", err)
	}
	if n := ix.NoteCount(); n < 5000 {
		b.Fatalf("o cofre de benchmark tem %d notas; o baseline foi medido com 5000, "+
			"e comparar escalas diferentes nao mede regressao nenhuma", n)
	}

	// Escolhe, entre as notas do cofre sintetico, uma que tem link de saida
	// E backlink de entrada — sem isso o benchmark mediria so a parte barata
	// do caminho (nem resolveTarget nem o reprocessamento de citantes seriam
	// exercitados). gen_vault.ps1 roda com semente fixa, mas a PASTA de cada
	// nota tambem e sorteada, entao o caminho nao pode ser hardcoded.
	var target vault.CanonicalPath
	for _, p := range ix.NotePaths() {
		n, _ := ix.Get(p)
		if len(n.Links) > 0 && len(ix.Backlinks(p)) > 0 {
			target = p
			break
		}
	}
	if target == "" {
		b.Fatal("nenhuma nota do cofre de benchmark tem link de saida e backlink de entrada")
	}

	ctx := context.Background()
	for b.Loop() {
		if err := ix.Replace(ctx, v, target); err != nil {
			b.Fatalf("Replace: %v", err)
		}
	}
}
