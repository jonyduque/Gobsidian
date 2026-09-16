package console

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// Confirmar faz uma pergunta de sim ou nao com dois BOTOES, na mesma moldura
// das listas de selecao.
//
// # Por que botoes, e nao "S/n"
//
// A pergunta digitada exigia ler o sufixo para saber o que o Enter faz, e o
// que aparecia na tela depois era uma letra solta. Com botoes, a resposta
// corrente esta SEMPRE visivel -- e o Enter confirma o que se esta vendo, em
// vez do que se leu tres linhas atras.
//
// Os itens que justificam a pergunta (os processos a encerrar, por exemplo)
// entram na mesma moldura: quem decide precisa ver sobre o que decide, e uma
// lista impressa antes da pergunta rola para fora da tela.
//
// # Teclas
//
//	seta esquerda/direita, h/l   troca de botao
//	tab                          troca de botao
//	s, n (e y)                   responde direto
//	enter                        confirma o botao em foco
//	esc, q, Ctrl-C               cancela
//
// # Sem terminal
//
// Devolve ErrSemTerminal quando a entrada nao e um console -- pipe, IDE, CI.
// Quem chama cai na pergunta digitada, como nas listas. Esse caminho nao e
// excecao rara: o bootstrap roda por cano, e foi assim que o instalador leu o
// proprio script como resposta em 2026-09-11.
func Confirmar(s *Stream, entrada *os.File, pergunta string, itens []string, padrao bool) (bool, error) {
	restaurar, err := entrarNoModoBruto(entrada)
	if err != nil {
		return padrao, ErrSemTerminal
	}
	defer restaurar()

	sim := padrao
	desenharBotoes(s, pergunta, itens, sim, false)

	buf := make([]byte, 8)
	for {
		n, err := entrada.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return false, ErrCancelado
			}
			return false, err
		}
		if n == 0 {
			continue
		}

		acao, valor := interpretarBotao(buf[:n], sim)
		switch acao {
		case mover:
			sim = valor
		case confirmar:
			sim = valor
			desenharBotoes(s, pergunta, itens, sim, true)
			return sim, nil
		case cancelar:
			desenharBotoes(s, pergunta, itens, false, true)
			return false, ErrCancelado
		default:
			continue
		}
		desenharBotoes(s, pergunta, itens, sim, true)
	}
}

// interpretarBotao traduz os bytes lidos numa acao e no valor que ela carrega:
// para mover, o botao que passa a ter foco; para confirmar, a RESPOSTA.
//
// Separada de Confirmar para ter teste sem terminal, pelo mesmo motivo de
// interpretar: o que decide o comportamento e esta tabela.
func interpretarBotao(b []byte, sim bool) (acaoDeTecla, bool) {
	// Setas: ESC [ C (direita) e ESC [ D (esquerda). O botao "Sim" fica a
	// esquerda, entao a direcao decide o valor -- e nao alterna, porque numa
	// escolha de dois alternar e mover, e repetir a mesma seta ficaria
	// piscando entre os dois.
	if len(b) >= 3 && b[0] == 0x1b && b[1] == '[' {
		switch b[2] {
		case 'D':
			return mover, true
		case 'C':
			return mover, false
		}
		return nada, sim
	}
	if len(b) == 1 && b[0] == 0x1b {
		return cancelar, sim
	}

	switch b[0] {
	case 'h', 'H':
		return mover, true
	case 'l', 'L':
		return mover, false
	case '\t', ' ':
		return mover, !sim
	case 's', 'S', 'y', 'Y':
		return confirmar, true
	case 'n', 'N':
		return confirmar, false
	case '\r', '\n':
		return confirmar, sim
	case 'q', 'Q', 0x03: // 0x03 = Ctrl-C
		return cancelar, sim
	}
	return nada, sim
}

// desenharBotoes escreve a pergunta, os itens e a linha de botoes dentro de uma
// moldura, redesenhando no lugar como a lista de selecao.
func desenharBotoes(s *Stream, pergunta string, itens []string, sim, redesenhar bool) {
	g := GlifosDaSaida()

	corpos := make([]string, 0, len(itens)+2)
	for _, i := range itens {
		corpos = append(corpos, "  "+s.Dim(i))
	}
	if len(itens) > 0 {
		corpos = append(corpos, "")
	}

	// Os dois botoes tem a MESMA largura em foco e fora dele: o glifo de foco
	// ocupa a coluna que, no outro, e espaco. Sem isso a linha muda de
	// comprimento a cada seta e a moldura pisca.
	botao := func(rotulo string, focado bool) string {
		if focado {
			return s.style("[ "+g.Cursor+" "+rotulo+" ]", corDestaque...)
		}
		return s.style("[   "+rotulo+" ]", corNota...)
	}
	corpos = append(corpos, "  "+botao("Sim", sim)+"   "+botao("Nao", !sim))

	rodape := g.Setas + " mover " + g.Separador + " s/n responder " + g.Separador + " " + g.Enter + " confirmar"

	linhas := s.Moldura(pergunta, corpos, rodape)
	if redesenhar && s.Colored() {
		_, _ = fmt.Fprintf(s.w, "\x1b[%dA", len(linhas))
	}
	for _, l := range linhas {
		if s.Colored() {
			l += "\x1b[K"
		}
		_, _ = fmt.Fprintln(s.w, l)
	}
}
