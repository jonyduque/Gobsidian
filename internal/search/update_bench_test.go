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

// BenchmarkInvertedUpdateLote mede a construcao do indice de busca do jeito que
// o servidor a faz: um Inverted.Update por caminho devolvido por
// index.NotePaths, que e exatamente o laco de buildInvertedIndex em
// cmd/gobsidian/serve.go.
//
// Existe para responder uma pergunta que ficou aberta na Task 97: a guarda de
// placeholder de nuvem paga um vault.IsCloudOnly — isto e, um GetFileAttributes
// no Windows — por nota, alem do que vault.Walk ja paga na varredura. Nao havia
// benchmark que cobrisse Update em lote, entao o custo entrou no ledger como
// "nao medido". Este e o instrumento; o custo se mede desligando a guarda e
// comparando com benchstat.
//
// Mede o caminho COMPLETO de propósito — leitura de disco, Analyze e Add —, e
// nao so a guarda: o numero que interessa e quanto a guarda pesa sobre o
// trabalho real, e nao quanto custa uma syscall isolada.
func BenchmarkInvertedUpdateLote(b *testing.B) {
	dir := os.Getenv("GOBSIDIAN_BENCH_VAULT")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "vault_5000")
	}
	if _, err := os.Stat(dir); err != nil {
		b.Skipf("cofre de benchmark ausente em %s; gere com "+
			"scripts/gen_vault.ps1 -Notes 5000 -Seed 42 -Out <caminho>", dir)
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
	caminhos := idx.NotePaths()

	// Guarda de corpus. Sem ela um cofre truncado produz um benchmark rapido e
	// verde, e a comparacao leria a perda de corpus como ganho.
	if len(caminhos) < 5000 {
		b.Fatalf("o cofre tem %d notas, quer >= 5000; comparar escalas diferentes "+
			"nao mede coisa nenhuma", len(caminhos))
	}

	for b.Loop() {
		inv := search.NewInverted()
		for _, p := range caminhos {
			if err := inv.Update(ctx, v, p); err != nil {
				b.Fatalf("Update %s: %v", p, err)
			}
		}
		// Afirma dentro do laco: um Update que passasse a nao indexar nada
		// ficaria dramaticamente mais rapido e o benchmark chamaria isso de
		// melhora.
		if inv.DocCount() < 5000 {
			b.Fatalf("DocCount = %d depois do lote, quer >= 5000", inv.DocCount())
		}
	}
}
