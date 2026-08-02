//go:build windows

package service_test

import (
	"os"
	"path/filepath"
	"testing"
)

// tornaInescrivel impede a escrita de `arquivo` dentro de `dir`.
//
// No Windows quem manda e o atributo do ARQUIVO: com ele somente-leitura, tanto
// a escrita direta quanto o rename por cima falham. O diretorio nao serve —
// os.Chmod num diretorio do Windows so mexe no atributo de somente-leitura, que
// nao impede criar arquivo dentro dele.
//
// Ver inescrivel_unix_test.go: la a regra e a oposta, e essa inversao deixou o
// CI vermelho de 2026-07-28 a 2026-08-01.
func tornaInescrivel(t *testing.T, dir, arquivo string) {
	t.Helper()
	alvo := filepath.Join(dir, arquivo)
	if err := os.Chmod(alvo, 0o400); err != nil {
		t.Fatalf("Chmod %s: %v", alvo, err)
	}
	t.Cleanup(func() { _ = os.Chmod(alvo, 0o644) })
}
