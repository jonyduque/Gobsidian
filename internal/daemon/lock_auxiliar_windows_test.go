//go:build windows

package daemon

import "os/exec"

// comandoTrivial devolve um processo que termina sozinho, para pidMorto obter
// um PID cuja morte foi OBSERVADA em vez de presumida.
func comandoTrivial() *exec.Cmd {
	return exec.Command("cmd", "/c", "exit")
}
