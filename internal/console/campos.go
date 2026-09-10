package console

import (
	"fmt"
	"strings"
)

// Campo e um par rotulo/valor de um relatorio.
type Campo struct {
	Chave string
	Valor string
	// Nota e um sufixo apagado, para a unidade ou a ressalva do valor.
	Nota string
}

// Campos imprime um bloco de pares alinhados.
//
// # Por que existe
//
// `index`, `inspect`, `search`, `version` e `doctor` imprimiam cada um do seu
// jeito: alguns com `con.Item("Notas: %d")`, outros com `con.Detail` indentado,
// e o alinhamento saia do acaso do tamanho de cada rotulo. Sao cinco formas de
// mostrar a MESMA coisa -- um relatorio de pares --, e a menos usada e a que
// diverge. Aqui e uma.
//
// A chave sai apagada e o valor no tom normal, e essa escolha nao e estetica:
// num relatorio a pergunta ja e conhecida e a RESPOSTA e o que se procura.
// Realcar o rotulo faria o olho bater no que ele ja sabe.
//
// O alinhamento e por celula visivel, e nao por byte -- um rotulo com acento
// desalinharia a coluna. Ver larguraVisivel.
func (s *Stream) Campos(titulo string, campos []Campo) {
	if len(campos) == 0 {
		return
	}

	maior := 0
	for _, c := range campos {
		if l := larguraVisivel(c.Chave); l > maior {
			maior = l
		}
	}

	corpos := make([]string, 0, len(campos))
	for _, c := range campos {
		enchimento := strings.Repeat(" ", maior-larguraVisivel(c.Chave))
		linha := "  " + s.style(c.Chave, corNota...) + enchimento + "  " + c.Valor
		if c.Nota != "" {
			linha += "  " + s.style(c.Nota, corNota...)
		}
		corpos = append(corpos, linha)
	}
	s.Bloco(titulo, corpos, "")
}

// Campof monta um par com valor formatado, para o chamador nao repetir
// fmt.Sprintf em cada linha.
func Campof(chave, format string, a ...any) Campo {
	return Campo{Chave: chave, Valor: fmt.Sprintf(format, a...)}
}
