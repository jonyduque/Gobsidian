//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// pidVivo diz se o processo de PID `pid` ainda esta em execucao.
//
// Sinal 0 nao entrega nada; so testa existencia e permissao. EPERM significa
// que o processo EXISTE e pertence a outro usuario — e vida, nao ausencia.
//
// Nao reusa lifecycle.parentIdentity de proposito: aquela funcao responde
// outra pergunta ("o pai continua sendo o MESMO processo") e se apoia em
// os.Getppid(), que nao faz sentido para um PID arbitrario lido de um arquivo.
func pidVivo(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	return err == syscall.EPERM
}
