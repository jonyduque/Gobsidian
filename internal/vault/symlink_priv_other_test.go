//go:build !windows

package vault_test

// Em Linux e macOS criar link simbolico nao pede privilegio nenhum, e
// os.IsPermission ja cobre os casos de diretorio sem permissao de escrita.
// Nao ha erro adicional a reconhecer — e devolver true aqui por precaucao
// transformaria uma falha real em skip, que e como cobertura fantasma nasce.
func errosDePrivilegioDeSymlink(error) bool { return false }
