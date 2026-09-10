//go:build linux || darwin

package console

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// entrarNoModoBruto poe o terminal em modo tecla-a-tecla e devolve a funcao que
// restaura o estado anterior.
//
// O irmao Unix de bruto_windows.go. Aqui o termios ja entrega seta como
// sequencia ANSI, entao so e preciso desligar o modo de linha (ICANON) e o eco
// (ECHO) -- os dois mesmos bits que o Windows desliga com outro nome.
//
// VMIN=1/VTIME=0: cada Read devolve assim que houver UMA tecla, sem esperar
// linha nem estourar prazo.
//
// Os numeros de ioctl NAO sao os mesmos nos dois: o Linux usa TCGETS/TCSETS e o
// macOS usa TIOCGETA/TIOCSETA. Eles ficam em bruto_linux.go e bruto_darwin.go,
// e nao num `if runtime.GOOS ==` aqui dentro.
//
// Restaurar NAO e opcional: terminal que fica sem ECHO depois que o processo
// sai deixa o usuario digitando as cegas.
func entrarNoModoBruto(f *os.File) (restaurar func(), err error) {
	fd := int(f.Fd())

	antes, err := unix.IoctlGetTermios(fd, ioctlLerTermios)
	if err != nil {
		// Nao e terminal: pipe, redirecionamento, CI. Quem chama cai no modo
		// digitado.
		return nil, errors.New("a entrada nao e um terminal")
	}

	depois := *antes
	depois.Lflag &^= unix.ICANON | unix.ECHO
	depois.Cc[unix.VMIN] = 1
	depois.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, ioctlEscreverTermios, &depois); err != nil {
		return nil, err
	}

	anterior := *antes
	return func() { _ = unix.IoctlSetTermios(fd, ioctlEscreverTermios, &anterior) }, nil
}
