//go:build windows

package lifecycle

import (
	"testing"

	"golang.org/x/sys/windows"
)

// Reciclagem de PID e o cenario que este pacote existe para cobrir e que
// nenhum teste de integracao consegue provocar. A comparacao em si e pura,
// entao e verificavel diretamente.
func TestSameProcessRejectsRecycledPID(t *testing.T) {
	base := identity{pid: 4242, created: windows.Filetime{HighDateTime: 31, LowDateTime: 1000}}

	same := identity{pid: 4242, created: windows.Filetime{HighDateTime: 31, LowDateTime: 1000}}
	if !sameProcess(base, same) {
		t.Error("mesma identidade reportada como processo diferente")
	}

	recycled := identity{pid: 4242, created: windows.Filetime{HighDateTime: 31, LowDateTime: 2000}}
	if sameProcess(base, recycled) {
		t.Error("PID reciclado aceito como mesmo processo — a vigilia nunca dispararia")
	}

	// Filetime tem duas metades de 32 bits, e sameProcess compara as duas.
	// Sem este caso, uma implementacao que largasse HighDateTime passaria no
	// teste — e HighDateTime e justamente a metade que muda quando os dois
	// processos nascem com mais de sete minutos de diferenca.
	recycledHigh := identity{pid: 4242, created: windows.Filetime{HighDateTime: 32, LowDateTime: 1000}}
	if sameProcess(base, recycledHigh) {
		t.Error("divergencia apenas em HighDateTime aceita como mesmo processo")
	}

	other := identity{pid: 9999, created: base.created}
	if sameProcess(base, other) {
		t.Error("PID diferente aceito como mesmo processo")
	}
}
