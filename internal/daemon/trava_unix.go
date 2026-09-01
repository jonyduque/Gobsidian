//go:build !windows

package daemon

import (
	"errors"
	"syscall"
)

// travarArquivo pede a trava exclusiva do kernel, sem bloquear.
//
// LOCK_NB e obrigatorio: sem ele quem perde a disputa DORME ate o dono soltar,
// e a decisao deste projeto e que quem perde sai, nao espera.
func travarArquivo(fd uintptr) (bool, error) {
	err := syscall.Flock(int(fd), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
		return false, nil
	}
	return false, err
}

// destravarArquivo solta a trava. Fechar o descritor tambem soltaria, mas
// soltar explicitamente mantem a simetria com o Windows, onde nao solta.
func destravarArquivo(fd uintptr) error {
	return syscall.Flock(int(fd), syscall.LOCK_UN)
}
