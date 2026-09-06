//go:build !windows

package index_test

import (
	"os"
	"testing"
)

// lockFileForTest torna o arquivo ilegivel do jeito que o POSIX permite. Nao ha
// share mode aqui — a permissao e a unica alavanca —, e por isso este lado nao
// vem de internal/vaulttest, cujo helper e de handle exclusivo do Windows.
func lockFileForTest(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0644) })

	// A mesma prova simetrica que vaulttest.TravarExclusivo faz no Windows: a
	// condicao e conferida antes de devolver. root ignora a permissao, e ai
	// toda assercao de "arquivo ilegivel" deste teste seria vazia.
	if _, err := os.ReadFile(path); err == nil {
		t.Fatalf("chmod 0000 nao barrou a leitura de %s (rodando como root?); a prova de 'arquivo ilegivel' seria vazia", path)
	}
}
