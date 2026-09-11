//go:build windows

package console

import (
	"os"

	"golang.org/x/sys/windows"
)

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
	// O Windows Terminal renderiza UTF-8 sempre, independente da code page
	// LEGADA que o console reporta. Medido em 2026-09-11 na maquina do dono:
	// GetConsoleOutputCP devolvia 850 e os acentos dos caminhos de cofre
	// apareciam CERTOS na tela -- a code page sozinha respondia nao onde a
	// resposta era sim, e a moldura saia em ASCII sem motivo.
	//
	// WT_SESSION so existe dentro do Windows Terminal, e quem a define e ele
	// proprio. E sinal de PRESENCA, nao de configuracao, entao nao envelhece
	// junto com a code page.
	if os.Getenv("WT_SESSION") != "" {
		return true
	}
	cp, err := windows.GetConsoleOutputCP()
	if err != nil {
		return false
	}
	return cp == cpUTF8
}
