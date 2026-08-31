package index_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func cofreParaList(b *testing.B, n int) *index.Index {
	b.Helper()
	root := b.TempDir()
	for i := range n {
		corpo := fmt.Sprintf("---\ntitle: Nota %04d\ntags: [t%d, comum]\n---\n\n%s\n",
			i, i%17, string(make([]byte, i%500+10)))
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("n%05d.md", i)), []byte(corpo), 0644); err != nil {
			b.Fatalf("escrevendo: %v", err)
		}
	}
	v, err := vault.New(root)
	if err != nil {
		b.Fatalf("vault.New: %v", err)
	}
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		b.Fatalf("Build: %v", err)
	}
	return ix
}

// BenchmarkListOrdenada mede o achado P4 no formato em que ele custa: o
// comparador roda O(n log n) vezes, e `strings.ToLower(q.Sort)` rodava DENTRO
// dele — a mesma string, lowercase de novo, a cada comparação.
func BenchmarkListOrdenada(b *testing.B) {
	ix := cofreParaList(b, 3000)
	q := index.Query{Sort: "size", Order: "desc", Limit: 100}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		if notes, _ := ix.List(q); len(notes) != 100 {
			b.Fatalf("notes = %d", len(notes))
		}
	}
}

// BenchmarkListPorTag exercita o outro lado do P4: `ToLower` por tag, por
// chamada, sobre o mapa de tags inteiro.
func BenchmarkListPorTag(b *testing.B) {
	ix := cofreParaList(b, 3000)
	q := index.Query{Tags: []string{"comum"}, TagMode: "any", Sort: "title", Limit: 100}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		if notes, _ := ix.List(q); len(notes) == 0 {
			b.Fatal("nenhuma nota")
		}
	}
}
