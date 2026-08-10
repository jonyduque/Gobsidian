//go:build !windows

package daemon

import (
	"os/exec"
	"syscall"
)

// applyDetachAttrs desprende o processo do daemon desta ponte, em Unix, via
// Setsid: o daemon vira lider de uma sessao nova, sem terminal de controle
// herdado -- a ponte sai logo depois de lanca-lo, e um filho preso a
// sessao dela receberia SIGHUP quando ela morrer.
func applyDetachAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
