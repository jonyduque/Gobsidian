//go:build windows

package console

import (
	"os"

	"golang.org/x/sys/windows"
)

// ativarVT liga ENABLE_VIRTUAL_TERMINAL_PROCESSING no console anexado a f.
//
// Sem isto, o console legado do Windows (conhost, que ainda e o padrao em
// muitas maquinas) IMPRIME as sequencias como texto -- a saida vira
// "<-[1;32m[OK] ..." em vez de um [OK] verde, que e pior que nao ter cor
// nenhuma. O Windows Terminal ja vem com o modo ligado, mas ligar de novo
// nao custa e nao quebra.
//
// Devolver false quando a chamada falha e o que faz o chamador desistir da
// cor em vez de emitir sequencias que ninguem vai interpretar. Falha aqui
// nao e erro do programa: e um console que nao suporta o modo, e a resposta
// certa e saida sem cor.
//
// O modo NAO e restaurado no fim. Restaurar exigiria um gancho de saida que
// nao roda quando o processo e morto por sinal ou por morte do pai -- os dois
// caminhos que este projeto exercita 100 vezes por cenario no gate de orfaos.
// Deixar VT ligado num console e inofensivo: e o padrao de todo terminal
// moderno, e o proprio Windows o reseta quando o console fecha.
func ativarVT(f *os.File) bool {
	handle := windows.Handle(f.Fd())

	var modo uint32
	if err := windows.GetConsoleMode(handle, &modo); err != nil {
		return false
	}
	if modo&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true
	}
	if err := windows.SetConsoleMode(handle, modo|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return false
	}
	return true
}
