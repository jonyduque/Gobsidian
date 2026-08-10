package search_test

import (
	"context"
	"os"
	"testing"

	"github.com/jonyd/gobsidian/internal/search"
)

// BenchmarkLoadInvertedCacheReal mede o carregamento do cache sobre um cofre de
// verdade, e não sobre um corpus gerado.
//
// Um corpus sintético mente aqui em duas frentes que decidem justamente o que
// este código otimiza: os caminhos ficam curtos e uniformes (o cofre de
// referência tem 101 caracteres em média, e a repetição deles era metade do
// arquivo antigo), e as posições ficam equiespaçadas, o que faz o delta em
// varint parecer melhor do que é.
//
// Fica atrás de variável de ambiente porque cofre real não entra no repositório
// e nem existe no CI. Sem ela, pula — um benchmark que mede um cofre vazio
// reportaria um número excelente e sem significado nenhum.
func BenchmarkLoadInvertedCacheReal(b *testing.B) {
	cacheDir := os.Getenv("GOBSIDIAN_BENCH_CACHE_DIR")
	if cacheDir == "" {
		b.Skip("GOBSIDIAN_BENCH_CACHE_DIR nao definido")
	}

	ctx := context.Background()

	// Confere fora do laço: erro dentro do laço medido vira "carregamento
	// instantaneo" em vez de falha.
	inv, hdr, err := search.LoadInvertedCache(ctx, cacheDir, "")
	if err != nil {
		b.Fatalf("LoadInvertedCache: %v", err)
	}
	if hdr.NoteCount == 0 || inv.TermCount() == 0 {
		b.Fatalf("cache vazio: %d notas, %d termos", hdr.NoteCount, inv.TermCount())
	}
	b.Logf("cofre de referencia: %d notas, %d termos", hdr.NoteCount, inv.TermCount())
	_ = inv.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		got, _, err := search.LoadInvertedCache(ctx, cacheDir, "")
		if err != nil {
			b.Fatal(err)
		}
		// Impede que o otimizador descarte o resultado inteiro.
		if got.TermCount() == 0 {
			b.Fatal("indice vazio")
		}
		// Task 89: fecha a arena mapeada a cada volta — sem isto o cofre real
		// fica mapeado uma vez por iteracao do benchmark, ate o processo
		// terminar.
		_ = got.Close()
	}
}
