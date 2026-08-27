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

// cofreComHub monta o caso que o achado P8 descreve: N notas citando o MESMO
// alvo. É onde a lista de citantes cresce, e onde a varredura por link virava
// soma quadrática.
func cofreComHub(b *testing.B, n int) *vault.Vault {
	b.Helper()
	root := b.TempDir()
	esc := func(nome, corpo string) {
		if err := os.WriteFile(filepath.Join(root, nome), []byte(corpo), 0644); err != nil {
			b.Fatalf("escrevendo %s: %v", nome, err)
		}
	}
	esc("Hub.md", "# Hub\n")
	for i := range n {
		esc(fmt.Sprintf("N%04d.md", i), fmt.Sprintf("# N%d\n\ncita [[Hub]] aqui.\n", i))
	}
	v, err := vault.New(root)
	if err != nil {
		b.Fatalf("vault.New: %v", err)
	}
	return v
}

// BenchmarkBuildComHub mede o Build de um cofre em que todas as notas citam o
// mesmo alvo — o formato em que o P8 aparece.
func BenchmarkBuildComHub(b *testing.B) {
	v := cofreComHub(b, 2000)
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		ix := index.New()
		if err := ix.Build(ctx, v); err != nil {
			b.Fatalf("Build: %v", err)
		}
	}
}

// BenchmarkTotalSizeRepetido mede o achado P6: `vault_stats` chama TotalSize, e
// ele varria os dois mapas inteiros sob lock a cada chamada.
func BenchmarkTotalSizeRepetido(b *testing.B) {
	v := cofreComHub(b, 2000)
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		b.Fatalf("Build: %v", err)
	}
	b.ResetTimer()
	for b.Loop() {
		if ix.TotalSize() <= 0 {
			b.Fatal("TotalSize zerou")
		}
	}
}

// BenchmarkBuildComHubGrande existe porque o hub de 2.000 notas nao distinguiu
// nada: o termo quadratico do P8 e ~2 milhoes de comparacoes contra um Build de
// ~460 ms, ou seja, ruido. A 8.000 ele e ~32 milhoes — 16x maior — e o RNF-09
// promete linearidade ate 20.000 notas, que e exatamente o que uma soma
// quadratica quebra.
func BenchmarkBuildComHubGrande(b *testing.B) {
	v := cofreComHub(b, 8000)
	ctx := context.Background()
	b.ResetTimer()
	for b.Loop() {
		ix := index.New()
		if err := ix.Build(ctx, v); err != nil {
			b.Fatalf("Build: %v", err)
		}
	}
}
