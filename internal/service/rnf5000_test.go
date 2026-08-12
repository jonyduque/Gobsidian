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
		loaded, _, err := search.LoadInvertedCache(context.Background(), cacheDir, vaultDir)
		if err != nil {
			t.Fatalf("LoadInvertedCache: %v", err)
		}
		dur := time.Since(start)
		rnf02Times = append(rnf02Times, dur)
		// Task 89: fecha a arena mapeada (se houver) ANTES da proxima volta —
		// medido o tempo de carga, o mapeamento nao serve mais para nada, e
		// cinco deles abertos ao mesmo tempo sobre o mesmo arquivo fariam o
		// t.TempDir() do fim do teste recusar apagar o diretorio no Windows.
		_ = loaded.Close()
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
	//
	// O serviço é montado com o índice VINDO DO CACHE, não com inv.
	//
	// Inverted.Postings tem dois ramos. Índice construído do zero (inv):
	// base == nil, tudo vive no delta em mapas, e a função ORDENA. Índice
	// carregado do cache: base != nil e delta vazio, e ela devolve a fatia do
	// base sem ordenar. O servidor em produção sempre carrega do cache, e
	// medido no mesmo cofre e na mesma consulta os dois ramos diferem por 4,4x
	// (BenchmarkSearchLimit200 174.791.983 ns/op contra
	// BenchmarkSearchLimit200Cache 39.565.533 ns/op). Até 2026-08-12 este bloco
	// media inv, isto é, o ramo que o servidor não executa.
	//
	// Instância própria, viva até o fim do teste: a do laço de RNF-02 é
	// fechada a cada volta de propósito (ver o comentário lá).
	doCache, _, err := search.LoadInvertedCache(context.Background(), cacheDir, vaultDir)
	if err != nil {
		t.Fatalf("LoadInvertedCache para o RNF-04: %v", err)
	}
	defer func() { _ = doCache.Close() }()

	// Guarda de ramo, no espírito da de bench_cache_test.go. Um cache recusado
	// em silêncio — troca de formato, versão de analisador, caminho de cofre
	// diferente — faria este bloco medir exatamente o ramo que ele existe para
	// NÃO medir, e ninguém notaria.
	//
	// DocCount é o que a distingue de fora do pacote: as duas construções de
	// LoadInvertedCache passam por newInvertedFromSoA, e a única maneira de sair
	// dali com base == nil é a base ter vindo vazia — o que dá DocCount zero.
	// Índice com 5.000 documentos vindo daqui tem base.
	if doCache == nil {
		t.Fatal("LoadInvertedCache devolveu cache nulo; sem base o RNF-04 mediria o ramo do delta")
	}
	if doCache.DocCount() < 5000 {
		t.Fatalf("o indice vindo do cache tem %d documentos, quer >= 5000; "+
			"o cache foi recusado e o RNF-04 mediria o ramo do delta", doCache.DocCount())
	}

	// Cache de trecho DESLIGADO: o laço abaixo repete cada consulta 30 vezes, e
	// com o cache ligado 29 dessas 30 acertariam. O p95 cairia para o valor da
	// consulta repetida, que nenhum usuário vê na primeira busca — seria um RNF
	// declarado atingido por medir outra coisa.
	svc := service.New(v, idx, doCache, nil, service.Options{SnippetCacheEntries: &semCacheDeTrecho})
	t.Logf("=== RNF-04 (Latencia vault_search p95 5.000 notas, indice vindo do CACHE) ===")

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
