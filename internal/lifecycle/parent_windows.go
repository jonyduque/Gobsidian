//go:build windows

package lifecycle

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// identity e PID mais creation time. So o par identifica um processo de forma
// estavel: o PID sozinho e reciclado, e o creation time sozinho colide entre
// processos iniciados no mesmo tick.
type identity struct {
	pid     int
	created windows.Filetime
	// exited marca que o processo JA TERMINOU, ainda que continue
	// consultavel. Ver parentIdentity.
	exited bool
}

func parentIdentity(pid int) (identity, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return identity{}, fmt.Errorf("abrindo processo %d: %w", pid, err)
	}
	defer func() { _ = windows.CloseHandle(h) }()

	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return identity{}, fmt.Errorf("lendo tempos do processo %d: %w", pid, err)
	}

	// O Windows nao libera o PID enquanto houver um handle aberto para o
	// processo morto, e o proprio ato de consultar mantem um. Na pratica, um
	// processo terminado continua respondendo a OpenProcess e devolvendo o
	// mesmo creation time por um bom tempo depois de morrer — entao comparar
	// so (pid, created) NUNCA percebe a morte do pai.
	//
	// exitTime nao-zero e a resposta: ele so e preenchido quando o processo
	// termina. Este e o campo que faz a vigilia funcionar, e ignora-lo foi o
	// defeito que deixou 5 de 5 orfaos no primeiro teste de ponta a ponta.
	exited := exit.HighDateTime != 0 || exit.LowDateTime != 0

	return identity{pid: pid, created: creation, exited: exited}, nil
}

func sameProcess(a, b identity) bool {
	// Um processo que ja terminou nunca e "o mesmo processo", mesmo que PID e
	// creation time ainda batam. Devolver false aqui leva watchParent ao ramo
	// de divergencia de identidade, que dispara na hora — e o correto: morte
	// confirmada nao e condicao transitoria e nao merece o limiar de falhas.
	if b.exited {
		return false
	}
	return a.pid == b.pid &&
		a.created.HighDateTime == b.created.HighDateTime &&
		a.created.LowDateTime == b.created.LowDateTime
}
