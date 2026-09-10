//go:build windows

package vaulttest

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jonyduque/Gobsidian/internal/vault"
)

// TravarExclusivo segura um handle exclusivo sobre o arquivo ate o fim do
// teste e CONFERE que ele barra os.ReadFile antes de devolver.
//
// Leitura E escrita: um handle exclusivo que pede so GENERIC_READ nao barra
// o os.ReadFile (medido; ver docs/ARMADILHAS.md).
func TravarExclusivo(t testing.TB, abs string) {
	t.Helper()
	h := abrirExclusivo(t, abs, 0)
	t.Cleanup(func() { _ = windows.CloseHandle(h) })
	if _, err := os.ReadFile(abs); err == nil {
		t.Fatalf("vaulttest: o handle exclusivo nao barrou a leitura de %s; a prova de 'nao abriu' seria vazia", abs)
	}
}

// TravarDiretorioExclusivo e o equivalente para diretorio: com share mode 0 o
// os.ReadDir falha, que e a condicao que vault.Walk e o watcher precisam
// distinguir de "diretorio vazio".
func TravarDiretorioExclusivo(t testing.TB, abs string) {
	t.Helper()
	h := abrirExclusivo(t, abs, windows.FILE_FLAG_BACKUP_SEMANTICS)
	t.Cleanup(func() { _ = windows.CloseHandle(h) })
	if _, err := os.ReadDir(abs); err == nil {
		t.Fatalf("vaulttest: o handle exclusivo nao barrou a listagem de %s", abs)
	}
}

func abrirExclusivo(t testing.TB, abs string, flags uint32) windows.Handle {
	t.Helper()
	p, err := windows.UTF16PtrFromString(vault.LongPath(abs))
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL|flags, 0)
	if err != nil {
		t.Skipf("vaulttest: nao foi possivel abrir %s em modo exclusivo: %v", abs, err)
	}
	return h
}
