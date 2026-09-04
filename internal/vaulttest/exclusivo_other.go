//go:build !windows

package vaulttest

import "testing"

// TravarExclusivo nao existe fora do Windows: POSIX nao tem share mode. O
// teste que depende dele pula, e o nome do pulo diz por que.
func TravarExclusivo(t testing.TB, _ string) {
	t.Helper()
	t.Skip("handle exclusivo e semantica do Windows")
}

// TravarDiretorioExclusivo pula pelo mesmo motivo: em POSIX um diretorio
// aberto por alguem nao deixa de ser legivel por mais ninguem.
func TravarDiretorioExclusivo(t testing.TB, _ string) {
	t.Helper()
	t.Skip("handle exclusivo e semantica do Windows")
}
