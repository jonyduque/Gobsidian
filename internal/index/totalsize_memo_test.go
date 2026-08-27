package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestTotalSizeMemorizaEInvalida é o achado P6, e o risco da correção é maior
// que o do defeito: memorizar sem invalidar devolve um número obsoleto para
// sempre, e `vault_stats` passaria a mentir em silêncio.
//
// Por isso o teste cobre as DUAS direções — que o valor é reutilizado enquanto
// nada muda, e que ele muda quando o índice muda.
func TestTotalSizeMemorizaEInvalida(t *testing.T) {
	root := t.TempDir()
	escrever := func(nome, corpo string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, nome), []byte(corpo), 0644); err != nil {
			t.Fatalf("escrevendo %s: %v", nome, err)
		}
	}
	escrever("A.md", "# A\n\ntexto curto\n")
	escrever("B.md", "# B\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	primeiro := ix.TotalSize()
	if primeiro <= 0 {
		t.Fatalf("TotalSize = %d: o cenario nao exercita nada", primeiro)
	}
	if segundo := ix.TotalSize(); segundo != primeiro {
		t.Errorf("duas chamadas seguidas sem mudanca deram %d e %d", primeiro, segundo)
	}

	// Cresce a nota e reindexa: o total TEM de acompanhar. Se a memorização não
	// invalidar, este é o assert que pega — e o sintoma real seria vault_stats
	// devolvendo o tamanho do boot para sempre.
	escrever("A.md", "# A\n\n"+string(make([]byte, 5000))+"\n")
	if err := ix.Replace(context.Background(), v, "A.md"); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	depois := ix.TotalSize()
	if depois <= primeiro {
		t.Errorf("TotalSize = %d depois de crescer a nota, era %d antes\n"+
			"a memorizacao nao invalidou: o numero congelou no valor do boot", depois, primeiro)
	}
}
