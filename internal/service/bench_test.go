package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Os benchmarks abaixo são o que scripts/bench_compare.ps1 compara contra
// docs/bench-baseline.json. Cada um cobre um caminho com alvo de RNF nomeado:
//
//	BenchmarkIndexBuild    RNF-01, indexação a frio
//	BenchmarkInvertedLoad  RNF-02, boot com cache válido
//	BenchmarkSearch*       RNF-04, latência de vault_search por formato
//
// A comparação é POR BENCHMARK, nunca agregada: um agregado esconde a regressão
// de um caminho atrás da melhora de outro.
//
// Eles rodam contra o cofre determinístico de gen_vault.ps1 com semente fixa.
// Sem cofre não há medição: o benchmark PULA, e o comparador trata benchmark
// ausente como erro. Um bench que mede um corpus vazio reporta um número ótimo
// e some com a regressão que existia para pegar.

// benchVaultPath devolve a raiz do cofre sintético de benchmark.
//
// GOBSIDIAN_BENCH_VAULT existe para o CI, que gera o cofre num diretório de
// trabalho próprio. O padrão é o mesmo caminho que rnf5000_test.go usa, para
// que uma geração local sirva aos dois.
func benchVaultPath(b *testing.B) string {
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

// benchServico monta a pilha completa uma vez, fora do cronômetro.
func benchServico(b *testing.B) *service.Service {
	b.Helper()
	dir := benchVaultPath(b)
	v, err := vault.New(dir)
	if err != nil {
		b.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		b.Fatalf("idx.Build: %v", err)
	}
	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		if err := inv.Update(context.Background(), v, p); err != nil {
			b.Fatalf("inv.Update %s: %v", p, err)
		}
	}
	// Guarda de corpus. Sem ele, um cofre truncado produziria benchmarks
	// rápidos e verdes, e o gate de 20% leria a perda de corpus como melhora.
	if n := idx.NoteCount(); n < 5000 {
		b.Fatalf("o cofre de benchmark tem %d notas; o baseline foi medido com 5000, "+
			"e comparar escalas diferentes nao mede regressao nenhuma", n)
	}
	return service.New(v, idx, inv, nil, service.Options{})
}

func BenchmarkIndexBuild(b *testing.B) {
	dir := benchVaultPath(b)
	v, err := vault.New(dir)
	if err != nil {
		b.Fatalf("vault.New: %v", err)
	}
	for b.Loop() {
		idx := index.New()
		if err := idx.Build(context.Background(), v); err != nil {
			b.Fatalf("idx.Build: %v", err)
		}
		if n := idx.NoteCount(); n < 5000 {
			b.Fatalf("NoteCount = %d, quer 5000", n)
		}
	}
}

func BenchmarkInvertedLoad(b *testing.B) {
	dir := benchVaultPath(b)
	v, err := vault.New(dir)
	if err != nil {
		b.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		b.Fatalf("idx.Build: %v", err)
	}
	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		if err := inv.Update(context.Background(), v, p); err != nil {
			b.Fatalf("inv.Update %s: %v", p, err)
		}
	}
	cacheDir := b.TempDir()
	if err := search.SaveInvertedCache(context.Background(), cacheDir, dir, inv); err != nil {
		b.Fatalf("SaveInvertedCache: %v", err)
	}

	for b.Loop() {
		got, _, err := search.LoadInvertedCache(context.Background(), cacheDir, dir)
		if err != nil {
			b.Fatalf("LoadInvertedCache: %v", err)
		}
		if got == nil {
			b.Fatal("LoadInvertedCache devolveu cache nulo")
		}
	}
}

// benchBusca roda um formato de consulta e afirma que ele casou alguma coisa.
//
// A asserção de resultado não é decoração. Uma consulta que deixa de casar
// mede o caminho vazio e fica dramaticamente mais rápida — este projeto já
// registrou uma medição de RNF-04 de 0,58 ms porque chamava a camada errada.
// Sem esta guarda, esse mesmo defeito apareceria como melhora de 90% e o gate
// só emitiria um aviso.
func benchBusca(b *testing.B, svc *service.Service, opts service.SearchOptions, minResultados int) {
	b.Helper()
	ctx := context.Background()
	for b.Loop() {
		res, err := svc.Search(ctx, opts)
		if err != nil {
			b.Fatalf("Search: %v", err)
		}
		if len(res.Results) < minResultados {
			b.Fatalf("a consulta casou %d resultados, esperava ao menos %d; "+
				"benchmark que nao casa nada mede o caminho vazio",
				len(res.Results), minResultados)
		}
	}
}

func BenchmarkSearchTermoAmplo(b *testing.B) {
	benchBusca(b, benchServico(b), service.SearchOptions{Query: "nota"}, 10)
}

func BenchmarkSearchDoisTermos(b *testing.B) {
	benchBusca(b, benchServico(b), service.SearchOptions{Query: "servidor mcp"}, 1)
}

func BenchmarkSearchFraseExata(b *testing.B) {
	benchBusca(b, benchServico(b), service.SearchOptions{Query: `"algoritmo BM25 com pesos"`}, 1)
}

func BenchmarkSearchLimit200(b *testing.B) {
	benchBusca(b, benchServico(b), service.SearchOptions{Query: "nota", Limit: 200}, 200)
}
