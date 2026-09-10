//go:build windows

package console

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// Bits do modo de console de ENTRADA que precisam sair para ler tecla a tecla.
//
// ENABLE_LINE_INPUT faz o console so entregar bytes depois do Enter, e
// ENABLE_ECHO_INPUT ecoa o que o usuario digita -- os dois transformam uma
// lista de caixas numa lista de caixas com lixo digitado no meio.
//
// ENABLE_VIRTUAL_TERMINAL_INPUT entra no lugar deles: com ele as setas chegam
// como sequencia ANSI (ESC [ A), a MESMA que chega no Unix, e a leitura de
// tecla fica sendo um codigo so para as duas plataformas. Sem ele o Windows
// entrega KEY_EVENT_RECORD por outra API, e seria uma segunda conta da mesma
// regra.
const (
	entradaASair   = windows.ENABLE_LINE_INPUT | windows.ENABLE_ECHO_INPUT | windows.ENABLE_PROCESSED_INPUT
	entradaAEntrar = windows.ENABLE_VIRTUAL_TERMINAL_INPUT
)

// entrarNoModoBruto poe o console de entrada em modo tecla-a-tecla e devolve a
// funcao que restaura o modo anterior.
//
// Restaurar NAO e opcional: um console que fica sem ECHO depois que o processo
// sai deixa o terminal do usuario mudo, e ele so descobre digitando.
func entrarNoModoBruto(f *os.File) (restaurar func(), err error) {
	h := windows.Handle(f.Fd())

	var antes uint32
	if err := windows.GetConsoleMode(h, &antes); err != nil {
		// Nao e console: pipe, redirecionamento, terminal de IDE sem console
		// real. Quem chama cai no modo digitado.
		return nil, errors.New("a entrada nao e um console")
	}

	depois := (antes &^ entradaASair) | entradaAEntrar
	if err := windows.SetConsoleMode(h, depois); err != nil {
		return nil, err
	}
	return func() { _ = windows.SetConsoleMode(h, antes) }, nil
}
