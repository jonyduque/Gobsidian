//go:build !windows

package lifecycle

import (
	"fmt"
	"os"
	"syscall"
)

// No Unix nao ha equivalente barato e portavel de creation time, e nao ha
// necessidade: quando o pai morre, o filho e reparentado, e os.Getppid()
// muda. Essa mudanca e o sinal, e ela nao sofre de reutilizacao de PID.
//
// O que a identidade guarda e o ppid observado no momento da captura, nao a
// constante 1. Comparar contra 1 parece equivalente e nao e: em containers e
// sob supervisores modernos o reaper nao e o init. Docker com tini, systemd
// com um servico de usuario e s6 todos reparentam para um PID que nao e 1, e
// uma verificacao de `Getppid() == 1` nunca dispara nesses ambientes — o pai
// morre e o processo sobrevive indefinidamente, que e exatamente a falha que
// este pacote existe para eliminar.
//
// Comparar o ppid atual com o inicial funciona em qualquer um deles: se
// mudou, fomos reparentados, e so ha um motivo para isso.
type identity struct {
	pid  int
	ppid int
}

func parentIdentity(pid int) (identity, error) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return identity{}, fmt.Errorf("localizando processo %d: %w", pid, err)
	}
	// Sinal 0 nao entrega nada; so testa existencia e permissao.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return identity{}, fmt.Errorf("processo %d inacessivel: %w", pid, err)
	}

	return identity{pid: pid, ppid: os.Getppid()}, nil
}

func sameProcess(a, b identity) bool {
	return a.pid == b.pid && a.ppid == b.ppid
}
