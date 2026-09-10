//go:build windows

package console

import "golang.org/x/sys/windows"

// cpUTF8 e a code page 65001. O console so renderiza fora do ASCII sem virar
// lixo quando esta nela.
const cpUTF8 = 65001

// suportaUnicode pergunta ao console qual code page ele usa para SAIDA.
//
// Perguntar, e nao supor: o Windows Terminal moderno ja abre em 65001, e o
// conhost legado de uma instalacao em portugues abre em CP-850. Os dois rodam
// o mesmo binario, e o instalador roda nos dois -- `iex (irm ...)` num
// PowerShell 5.1 recem-aberto e exatamente o segundo caso.
//
// GetConsoleOutputCP devolve 0 quando a saida nao e um console (pipe,
// redirecionamento), e ai a resposta certa e ASCII: quem vai ler aquilo e um
// arquivo ou outro programa.
func suportaUnicode() bool {
	cp, err := windows.GetConsoleOutputCP()
	if err != nil {
		return false
	}
	return cp == cpUTF8
}
