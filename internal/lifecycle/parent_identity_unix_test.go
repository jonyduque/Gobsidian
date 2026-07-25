//go:build !windows

package lifecycle

import "testing"

// O ppid observado no momento da captura e o que torna a comparacao segura
// contra reparentamento; nao ha creation time no Unix, entao esse e o unico
// sinal disponivel. A comparacao em si e pura, entao e verificavel
// diretamente, sem precisar provocar reparentamento de verdade.
func TestSameProcessRejectsDifferentPpid(t *testing.T) {
	base := identity{pid: 4242, ppid: 1000}

	same := identity{pid: 4242, ppid: 1000}
	if !sameProcess(base, same) {
		t.Error("mesma identidade reportada como processo diferente")
	}

	reparented := identity{pid: 4242, ppid: 2000}
	if sameProcess(base, reparented) {
		t.Error("ppid divergente aceito como mesmo processo — reparentamento nao seria detectado")
	}

	other := identity{pid: 9999, ppid: base.ppid}
	if sameProcess(base, other) {
		t.Error("PID diferente aceito como mesmo processo")
	}
}
