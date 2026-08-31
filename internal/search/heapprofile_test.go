package search_test

import (
	"context"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
)

// TestPerfilDeHeapServindo existe para responder UMA pergunta com número: onde
// mora o heap vivo do estado `servindo` do RNF-07.
//
// Não é gate. Roda só com COFRE e CACHE apontados, e existe porque recomendar
// onde cortar memória sem saber onde ela está seria opinião.
func TestPerfilDeHeapServindo(t *testing.T) {
	vault := os.Getenv("COFRE")
	cache := os.Getenv("CACHE")
	if vault == "" || cache == "" {
		t.Skip("sem COFRE/CACHE")
	}
	ctx := context.Background()

	idx, _, err := index.LoadIndexCache(ctx, cache, vault)
	if err != nil {
		t.Fatalf("LoadIndexCache: %v", err)
	}
	inv, _, err := search.LoadInvertedCache(ctx, cache, vault)
	if err != nil {
		t.Fatalf("LoadInvertedCache: %v", err)
	}

	// Estado `servindo`: uma busca já aconteceu.
	res := search.CalculateBM25(search.Analyze("processo"), inv, idx)
	if len(res) == 0 {
		t.Log("busca sem resultado; o perfil ainda vale para o estado carregado")
	}

	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	t.Logf("MEDIDA heap_vivo=%.1f MB sys=%.1f MB notas=%d docs=%d termos=%d",
		float64(m.HeapAlloc)/(1<<20), float64(m.Sys)/(1<<20),
		idx.NoteCount(), inv.DocCount(), inv.TermCount())

	// O perfil e gravado AQUI, com os dados vivos. O -memprofile do go test
	// escreve no fim do processo, quando o indice ja foi coletado — a primeira
	// tentativa devolveu 2 MB de ruido de runtime por isso.
	if saida := os.Getenv("HEAPOUT"); saida != "" {
		f, err := os.Create(saida)
		if err != nil {
			t.Fatalf("criando perfil: %v", err)
		}
		if err := pprof.WriteHeapProfile(f); err != nil {
			t.Fatalf("gravando perfil: %v", err)
		}
		_ = f.Close()
		t.Logf("perfil em %s", saida)
	}

	// Mantém tudo vivo até aqui, senão o GC coleta antes do perfil.
	runtime.KeepAlive(idx)
	runtime.KeepAlive(inv)
	runtime.KeepAlive(res)
}
