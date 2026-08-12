package service_test

import (
	"context"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

// benchServicoDeCache monta a pilha com o indice invertido CARREGADO DO CACHE.
//
// Existe porque benchServico constroi o indice do zero, e um indice construido
// do zero tem base == nil: toda leitura cai no delta, e Inverted.Postings
// materializa o mapa e ORDENA. O servidor em producao carrega o cache, entao
// base != nil e o delta esta vazio — Postings devolve a fatia do base sem
// ordenar. Sao dois ramos diferentes da mesma funcao, e otimizar medindo so um
// deles mira o ramo que o servidor nao usa.
// entradasTrecho é o teto do cache de trechos. O padrão dos benchmarks é o
// caminho FRIO (cache desligado), que é o que o RNF-04 cobra; ver
// semCacheDeTrecho em bench_test.go.
func benchServicoDeCache(b *testing.B) *service.Service {
	return benchServicoDeCacheCom(b, &semCacheDeTrecho)
}

func benchServicoDeCacheCom(b *testing.B, entradasTrecho *int) *service.Service {
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
	if n := idx.NoteCount(); n < 5000 {
		b.Fatalf("o cofre de benchmark tem %d notas; o baseline foi medido com 5000, "+
			"e comparar escalas diferentes nao mede regressao nenhuma", n)
	}

	cacheDir := b.TempDir()
	if err := search.SaveInvertedCache(context.Background(), cacheDir, dir, inv); err != nil {
		b.Fatalf("SaveInvertedCache: %v", err)
	}
	doCache, _, err := search.LoadInvertedCache(context.Background(), cacheDir, dir)
	if err != nil {
		b.Fatalf("LoadInvertedCache: %v", err)
	}
	if doCache == nil {
		b.Fatal("LoadInvertedCache devolveu cache nulo; sem base o benchmark mede o mesmo ramo que benchServico")
	}
	b.Cleanup(func() { _ = doCache.Close() })

	// Guarda de ramo. Sem ela, um cache recusado em silencio faria este
	// benchmark medir exatamente o ramo que ele existe para NAO medir.
	if doCache.DocCount() < 5000 {
		b.Fatalf("o indice vindo do cache tem %d documentos, quer >= 5000", doCache.DocCount())
	}

	return service.New(v, idx, doCache, nil, service.Options{SnippetCacheEntries: entradasTrecho})
}

func BenchmarkSearchLimit200Cache(b *testing.B) {
	benchBusca(b, benchServicoDeCache(b), service.SearchOptions{Query: "nota", Limit: 200}, 200)
}

func BenchmarkSearchTermoAmploCache(b *testing.B) {
	benchBusca(b, benchServicoDeCache(b), service.SearchOptions{Query: "nota"}, 10)
}
