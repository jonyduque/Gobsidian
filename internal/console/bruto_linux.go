//go:build linux

package console

import "golang.org/x/sys/unix"

// Os ioctls de termios no Linux. Ver bruto_unix.go para o uso.
const (
	ioctlLerTermios      = unix.TCGETS
	ioctlEscreverTermios = unix.TCSETS
)
