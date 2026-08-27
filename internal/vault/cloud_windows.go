//go:build windows

package vault

import (
	"io/fs"
	"syscall"

	"golang.org/x/sys/windows"
)

// Atributos que indicam arquivo nao hidratado pelo sincronizador de nuvem.
// Ler um arquivo assim dispara download sincrono, que pode levar segundos ou
// falhar sem conexao — e uma indexacao ingenua forcaria o download do cofre
// inteiro no boot.
const (
	attrRecallOnDataAccess = 0x00400000
	attrRecallOnOpen       = 0x00040000
)

// nuvemPorAtributos e a UNICA conta da regra "isto e placeholder de nuvem".
//
// IsCloudOnly e IsCloudOnlyInfo chegam ao mesmo bit de formas diferentes — uma
// por syscall, a outra pelo que o walk ja tinha em maos — e as duas terminam
// aqui. Duplicar a mascara nas duas faria a conta menos usada divergir na
// primeira vez que um atributo novo entrasse na lista, e o sintoma seria um
// cofre em que o boot respeita o placeholder e a reindexacao o abre.
func nuvemPorAtributos(attrs uint32) bool {
	const offline = windows.FILE_ATTRIBUTE_OFFLINE
	return attrs&(attrRecallOnDataAccess|attrRecallOnOpen|offline) != 0
}

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
	return nuvemPorAtributos(attrs)
}

// IsCloudOnlyInfo responde a mesma pergunta a partir de um fs.FileInfo que o
// chamador JA tem, sem gastar um segundo syscall (achado P14).
//
// No Windows o fs.DirEntry entregue por WalkDir vem de ReadDir, que ja traz os
// atributos completos: d.Info() nao vai ao disco. A varredura chamava
// IsCloudOnly(abs) logo depois de d.Info(), pagando um GetFileAttributes por
// entrada para reler o que estava na mao — num cofre de 5.686 arquivos em
// OneDrive, 5.686 syscalls evitaveis por varredura.
//
// Cai de volta em false quando Sys() nao traz os dados do Windows, que e o
// comportamento honesto: nao saber nao e o mesmo que saber que nao e.
func IsCloudOnlyInfo(info fs.FileInfo) bool {
	if info == nil {
		return false
	}
	dados, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok || dados == nil {
		return false
	}
	return nuvemPorAtributos(dados.FileAttributes)
}
