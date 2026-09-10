//go:build darwin

package console

import "golang.org/x/sys/unix"

// Os ioctls de termios no macOS -- nomes diferentes dos do Linux para a mesma
// operacao. Ver bruto_unix.go para o uso.
const (
	ioctlLerTermios      = unix.TIOCGETA
	ioctlEscreverTermios = unix.TIOCSETA
)
