//go:build windows

package vault

import "golang.org/x/sys/windows"

// Atributos que indicam arquivo nao hidratado pelo sincronizador de nuvem.
// Ler um arquivo assim dispara download sincrono, que pode levar segundos ou
// falhar sem conexao — e uma indexacao ingenua forcaria o download do cofre
// inteiro no boot.
const (
	attrRecallOnDataAccess = 0x00400000
	attrRecallOnOpen       = 0x00040000
)

// IsCloudOnly consulta os atributos sem abrir o arquivo, o que e o ponto:
// abrir e exatamente o que dispara a hidratacao.
func IsCloudOnly(abs string) bool {
	p, err := windows.UTF16PtrFromString(LongPath(abs))
	if err != nil {
		return false
	}
	attrs, err := windows.GetFileAttributes(p)
	if err != nil {
		return false
	}
	const offline = windows.FILE_ATTRIBUTE_OFFLINE
	return attrs&(attrRecallOnDataAccess|attrRecallOnOpen|offline) != 0
}
