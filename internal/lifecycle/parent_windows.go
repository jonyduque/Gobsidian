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

	return identity{pid: pid, created: creation}, nil
}

func sameProcess(a, b identity) bool {
	return a.pid == b.pid &&
		a.created.HighDateTime == b.created.HighDateTime &&
		a.created.LowDateTime == b.created.LowDateTime
}
