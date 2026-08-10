//go:build windows

package daemon

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// applyDetachAttrs desprende o processo do daemon do console e do grupo de
// processos desta ponte, no Windows. CREATE_NEW_PROCESS_GROUP evita que um
// CTRL_BREAK endereçado a este grupo (ver o cenario "signal" do gate de
// orfaos, scripts/test_orphans.ps1) alcance o daemon; DETACHED_PROCESS tira
// o daemon do console herdado -- ele nao tem, e nao deve ter, um.
func applyDetachAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
	}
}
