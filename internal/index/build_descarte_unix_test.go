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
}
