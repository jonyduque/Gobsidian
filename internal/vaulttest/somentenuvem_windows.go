//go:build windows

package vaulttest

import (
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jonyd/gobsidian/internal/vault"
)

// MarcarSomenteNuvem poe FILE_ATTRIBUTE_OFFLINE no arquivo, restaura NORMAL no
// fim do teste, e CONFERE que vault.IsCloudOnly passou a responder verdadeiro
// antes de devolver.
//
// E o unico atributo que serve: FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS nao e
// gravavel por SetFileAttributes — e o motivo de TestReadNoteCloudOnlyFails
// estar pulado —, e vault.IsCloudOnly aceita os dois. So Windows porque o
// atributo e do NTFS; fora dele IsCloudOnly devolve false por construcao e nao
// ha condicao para montar.
func MarcarSomenteNuvem(t testing.TB, abs string) {
	t.Helper()
	p, err := windows.UTF16PtrFromString(vault.LongPath(abs))
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Skipf("vaulttest: nao foi possivel marcar FILE_ATTRIBUTE_OFFLINE em %s: %v", abs, err)
	}
	t.Cleanup(func() {
		_ = windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_NORMAL)
	})
	if !vault.IsCloudOnly(abs) {
		t.Fatalf("vaulttest: FILE_ATTRIBUTE_OFFLINE nao fez vault.IsCloudOnly(%s) responder verdadeiro", abs)
	}
}
