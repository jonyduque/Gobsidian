//go:build !windows

package vaulttest

import "testing"

// MarcarSomenteNuvem nao tem equivalente fora do Windows: FILE_ATTRIBUTE_OFFLINE
// e atributo NTFS, e vault.IsCloudOnly devolve false por construcao nos outros
// sistemas — nao ha condicao para montar.
func MarcarSomenteNuvem(t testing.TB, _ string) {
	t.Helper()
	t.Skip("FILE_ATTRIBUTE_OFFLINE e atributo NTFS")
}
