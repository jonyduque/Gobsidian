//go:build windows

package vault_test

import (
	"errors"
	"syscall"
)

// ERROR_PRIVILEGE_NOT_HELD. E o que o Windows devolve quando o processo tenta
// criar um link simbolico sem privilegio elevado e sem o Modo de Desenvolvedor
// ligado. os.IsPermission NAO o reconhece — ele cobre ERROR_ACCESS_DENIED —,
// entao sem esta checagem o teste falharia numa maquina de desenvolvimento
// comum em vez de pular, que e o oposto do combinado.
const errPrivilegeNotHeld syscall.Errno = 1314

func errosDePrivilegioDeSymlink(err error) bool {
	return errors.Is(err, errPrivilegeNotHeld)
}
