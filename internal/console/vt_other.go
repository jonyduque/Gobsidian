//go:build !windows

package console

import "os"

// ativarVT nao faz nada fora do Windows: um terminal Unix que se declara
// terminal ja interpreta ANSI, e nao existe modo a ligar. Devolve true para
// que a decisao de cor dependa so das checagens portateis de SupportsColor
// (NO_COLOR, TERM=dumb, e f ser mesmo um dispositivo de caractere).
//
// O parametro se chama _ de proposito. A regra do projeto e que parametro que
// nenhum corpo usa recebe esse nome -- um parametro nomeado que ninguem le
// ensina o revisor a ignorar parametros.
func ativarVT(_ *os.File) bool { return true }
