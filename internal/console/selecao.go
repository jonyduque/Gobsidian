package console

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"golang.org/x/text/width"

	"github.com/jonyduque/Gobsidian/internal/textos"
)

// Opcao e um item de uma lista de selecao.
type Opcao struct {
	// Rotulo e o texto principal.
	Rotulo string
	// Nota e um sufixo esmaecido -- o caminho de um cofre, a chave de um host.
	Nota string
	// Marcada e o estado INICIAL da caixa.
	Marcada bool
}

// ErrSemTerminal diz que a selecao interativa nao e possivel neste ambiente:
// a entrada e um pipe, um redirecionamento ou um console sem modo bruto.
//
// E um erro esperado, e nao uma falha: quem chama cai numa pergunta digitada.
var ErrSemTerminal = errors.New("a entrada não é um terminal interativo")

// ErrCancelado diz que o usuario saiu da lista com Esc, q ou Ctrl-C.
//
// Distinto de "nao marcou nada": nao marcar nada e uma ESCOLHA, e cancelar e
// desistir. Quem chama trata as duas diferente -- a primeira configura zero
// cofres, a segunda aborta a instalacao.
var ErrCancelado = errors.New("seleção cancelada")

// Selecionar mostra uma lista de caixas e devolve os indices marcados.
//
// # Por que caixas, e nao um numero digitado
//
// A lista de cofres pedia um numero e aceitava um so. Escolher tres cofres era
// rodar o comando tres vezes, e escolher NENHUM nao era possivel -- a pergunta
// pressupunha a resposta. Caixas resolvem os dois: 0..N itens, sem digitar.
//
// # Teclas
//
//	seta cima/baixo, k/j   move
//	espaco                 marca ou desmarca
//	a                      marca ou desmarca todos
//	enter                  confirma
//	esc, q, Ctrl-C         cancela
//
// As setas chegam como sequencia ANSI nas duas plataformas -- no Windows
// porque bruto_windows.go liga ENABLE_VIRTUAL_TERMINAL_INPUT de proposito, para
// nao existirem duas leituras de tecla.
//
// # Desenho
//
// Uma moldura com o titulo na borda de cima e as teclas na de baixo. Os glifos
// vem de GlifosDaSaida: arredondados onde a codificacao do console aguenta,
// ASCII onde nao -- ver o comentario de estilo.go para por que a escolha e
// medida e nao suposta.
//
// A lista e redesenhada no lugar, subindo o cursor com ESC[<n>A; quando o
// Stream nao tem cor (saida redirecionada), ela sai uma vez e sem redesenho.
func Selecionar(s *Stream, entrada *os.File, titulo string, opcoes []Opcao) ([]int, error) {
	if len(opcoes) == 0 {
		return nil, nil
	}

	restaurar, err := entrarNoModoBruto(entrada)
	if err != nil {
		return nil, ErrSemTerminal
	}
	defer restaurar()

	marcadas := make([]bool, len(opcoes))
	for i, o := range opcoes {
		marcadas[i] = o.Marcada
	}

	cursor := 0
	desenhar(s, opcoes, marcadas, cursor, titulo, false)

	buf := make([]byte, 8)
	for {
		n, err := entrada.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, ErrCancelado
			}
			return nil, err
		}
		if n == 0 {
			continue
		}

		acao, alvo := interpretar(buf[:n], cursor, len(opcoes))
		switch acao {
		case mover:
			cursor = alvo
		case alternar:
			marcadas[cursor] = !marcadas[cursor]
		case alternarTodos:
			todos := true
			for _, m := range marcadas {
				if !m {
					todos = false
					break
				}
			}
			for i := range marcadas {
				marcadas[i] = !todos
			}
		case confirmar:
			desenhar(s, opcoes, marcadas, -1, titulo, true)
			var saida []int
			for i, m := range marcadas {
				if m {
					saida = append(saida, i)
				}
			}
			return saida, nil
		case cancelar:
			desenhar(s, opcoes, marcadas, -1, titulo, true)
			return nil, ErrCancelado
		default:
			continue
		}
		desenhar(s, opcoes, marcadas, cursor, titulo, true)
	}
}

type acaoDeTecla int

const (
	nada acaoDeTecla = iota
	mover
	alternar
	alternarTodos
	confirmar
	cancelar
)

// interpretar traduz os bytes lidos numa acao. Devolve tambem o novo indice do
// cursor quando a acao e mover.
//
// Fica separada de Selecionar para ter teste sem terminal: o que decide o
// comportamento da lista e esta tabela, e ela e testavel sozinha.
func interpretar(b []byte, cursor, total int) (acaoDeTecla, int) {
	// Sequencia de seta: ESC [ A (cima) ou ESC [ B (baixo).
	if len(b) >= 3 && b[0] == 0x1b && b[1] == '[' {
		switch b[2] {
		case 'A':
			return mover, (cursor - 1 + total) % total
		case 'B':
			return mover, (cursor + 1) % total
		}
		return nada, cursor
	}
	// ESC sozinho cancela. Com mais bytes atras dele e uma sequencia que esta
	// lista nao usa.
	if len(b) == 1 && b[0] == 0x1b {
		return cancelar, cursor
	}

	switch b[0] {
	case 'k', 'K':
		return mover, (cursor - 1 + total) % total
	case 'j', 'J':
		return mover, (cursor + 1) % total
	case ' ':
		return alternar, cursor
	case 'a', 'A':
		return alternarTodos, cursor
	case '\r', '\n':
		return confirmar, cursor
	case 'q', 'Q', 0x03: // 0x03 = Ctrl-C
		return cancelar, cursor
	}
	return nada, cursor
}

// larguraVisivel conta as CELULAS que uma string ocupa, ignorando as
// sequencias ANSI.
//
// Sem isso a moldura nao fecha: `s.Dim("(ja configurado)")` tem 16 caracteres
// visiveis e ~24 bytes, e alinhar pelo len() empurraria a barra da direita
// para dentro do texto. Conta por runa, e nao por byte, pelo mesmo motivo -- e
// quanto cada runa ocupa e larguraDaRuna que diz.
func larguraVisivel(s string) int {
	largura, dentroDeEscape := 0, false
	for _, r := range s {
		switch {
		case dentroDeEscape:
			// A sequencia SGR termina em 'm'; as de cursor em letra tambem.
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				dentroDeEscape = false
			}
		case r == 0x1b:
			dentroDeEscape = true
		default:
			largura += larguraDaRuna(r)
		}
	}
	return largura
}

// larguraDaRuna e quantas colunas uma runa visivel ocupa no terminal. Uma
// conta so para larguraVisivel e cortar: ate 2026-10-01 eram duas, e cortar
// contava a marca combinante como coluna -- "Ação" em NFD cortado em duas
// colunas perdia a cedilha.
//
//   - Marca combinante nao ocupa coluna: "ç" em NFD e "c" mais U+0327.
//     Medido em 2026-09-14: um caminho em NFD saia 4 colunas mais curto que a
//     borda da moldura.
//   - Ideograma, kana e forma de largura total ocupam duas (East Asian Width W
//     e F): um cofre com nome em japones desalinhava a borda.
//
// O seletor de variacao de emoji (U+FE0F) e marca, e conta zero: um terminal
// que desenhe "⚠️" em duas colunas desalinha a borda em uma. Nao medido em
// terminal real.
func larguraDaRuna(r rune) int {
	if unicode.In(r, unicode.Mn, unicode.Me) {
		return 0
	}
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	}
	return 1
}

// desenhar escreve a lista dentro de uma moldura. Com redesenhar, sobe o
// cursor o numero de linhas ja escritas antes de escrever por cima.
//
// O desenho e uma caixa porque a lista e uma pergunta, e uma pergunta com
// moldura separa o que se responde do que ja rolou na tela. Os glifos vem de
// GlifosDaSaida -- arredondados onde a codificacao aguenta, ASCII onde nao.
func desenhar(s *Stream, opcoes []Opcao, marcadas []bool, cursor int, titulo string, redesenhar bool) int {
	g := GlifosDaSaida()

	corpos := make([]string, 0, len(opcoes)+1)
	for i, o := range opcoes {
		// A cor carrega a informacao, e o GLIFO tambem: a caixa marcada muda de
		// simbolo E de cor, e a linha sob o cursor muda de ponta E de cor.
		// Terminal sem cor -- NO_COLOR, redirecionamento, conhost antigo --
		// continua distinguindo os dois estados, que e a mesma regra dos
		// marcadores [OK]/[!].
		caixa := s.style(g.Desmarcada, corNota...)
		if marcadas[i] {
			caixa = s.style(g.Marcada, corMarcada...)
		}
		ponta := " "
		rotulo := o.Rotulo
		if i == cursor {
			ponta = s.style(g.Cursor, corDestaque...)
			rotulo = s.style(rotulo, corDestaque...)
			// Em foco, a caixa vira o glifo de foco, e a COR continua dizendo
			// se a linha esta marcada. O conjunto ASCII nao tem glifo de foco
			// e mostra o estado.
			if g.Foco != "" {
				cor := corDestaque
				if marcadas[i] {
					cor = corMarcada
				}
				caixa = s.style(g.Foco, cor...)
			}
		}
		corpo := " " + ponta + " " + caixa + " " + rotulo
		if o.Nota != "" {
			corpo += "  " + s.style(o.Nota, corNota...)
		}
		corpos = append(corpos, corpo)
	}
	corpos = append(corpos, "")

	rodape := rodapeDeTeclas(g,
		[2]string{g.Setas, textos.TeclaMover},
		[2]string{"[ " + textos.TeclaEspaco + " ]", textos.TeclaMarcar},
		[2]string{textos.TeclaTodosLetra, textos.TeclaTodos},
		[2]string{g.Enter, textos.TeclaConfirmar})

	linhas := s.Moldura(titulo, corpos, rodape)

	if redesenhar && s.Colored() {
		_, _ = fmt.Fprintf(s.w, "\x1b[%dA", len(linhas))
	}
	for _, l := range linhas {
		if s.Colored() {
			l += "\x1b[K"
		}
		_, _ = fmt.Fprintln(s.w, l)
	}
	return len(linhas)
}

// ajustar preenche `conteudo` com `enchimento` ate a largura pedida.
//
// Conteudo mais largo que o teto e cortado, e cortar aqui e melhor que
// estourar: uma linha que passa da largura do terminal quebra sozinha e
// destroi a moldura inteira, inclusive o redesenho, que conta LINHAS.
func ajustar(conteudo string, largura int, enchimento string) string {
	l := larguraVisivel(conteudo)
	if l > largura {
		// A reticencia diz que o texto CONTINUA. Sem ela um trecho de busca
		// cortado no teto parece um trecho que simplesmente acaba ali, e quem
		// le nao sabe se falta conteudo ou se a nota e assim.
		fim := GlifosDaSaida().Reticencia
		cortado := cortar(conteudo, largura-larguraVisivel(fim)) + fim
		// Um ideograma que nao cabe inteiro fica de fora, e a coluna que
		// sobra e preenchida: sem isto a linha cortada sai mais curta que a
		// borda.
		return cortado + repetir(enchimento, largura-larguraVisivel(cortado))
	}
	return conteudo + repetir(enchimento, largura-l)
}

// cortar devolve as primeiras `largura` celulas visiveis, preservando as
// sequencias ANSI que aparecerem no caminho -- cortar no meio de um escape
// deixaria o resto da linha colorido.
func cortar(conteudo string, largura int) string {
	var b strings.Builder
	visiveis, dentroDeEscape := 0, false
	for _, r := range conteudo {
		switch {
		case dentroDeEscape:
			b.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				dentroDeEscape = false
			}
			continue
		case r == 0x1b:
			b.WriteRune(r)
			dentroDeEscape = true
			continue
		}
		w := larguraDaRuna(r)
		if visiveis+w > largura {
			break
		}
		b.WriteRune(r)
		visiveis += w
	}
	return b.String()
}

// rodapeDeTeclas monta o rodape de uma pergunta: cada tecla em negrito e o que
// ela faz em italico, separados pelo glifo de separacao. Sai com marcacao, e
// quem a desenha e a Moldura -- a mesma conta de todo texto do produto.
func rodapeDeTeclas(g Glifos, teclas ...[2]string) string {
	partes := make([]string, 0, len(teclas))
	for _, t := range teclas {
		partes = append(partes, "**"+t[0]+"** *"+t[1]+"*")
	}
	return strings.Join(partes, " "+g.Separador+" ")
}

func repetir(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, n)
}

// SelecionarDigitando e o caminho de quando nao ha terminal: a mesma pergunta,
// respondida por numeros separados por espaco ou virgula.
//
// Existe porque `install` roda em terminal de IDE, sob pipe e em CI, e uma
// instalacao que so funciona com terminal de verdade nao serve. Nao recebe
// Stream porque nao imprime nada: quem mostra a lista e quem chama. "" ou "0"
// significam nenhum; "*" significa todos.
func SelecionarDigitando(resposta string, opcoes []Opcao) ([]int, error) {
	resposta = strings.TrimSpace(resposta)
	if resposta == "" || resposta == "0" {
		return nil, nil
	}
	if resposta == "*" {
		todos := make([]int, len(opcoes))
		for i := range opcoes {
			todos[i] = i
		}
		return todos, nil
	}

	campos := strings.FieldsFunc(resposta, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})
	vistos := map[int]bool{}
	var saida []int
	for _, c := range campos {
		var n int
		if _, err := fmt.Sscanf(c, "%d", &n); err != nil {
			return nil, fmt.Errorf(textos.ErroEscolhaInvalida, c)
		}
		if n < 1 || n > len(opcoes) {
			return nil, fmt.Errorf(textos.ErroEscolhaForaDaLista, n)
		}
		if !vistos[n-1] {
			vistos[n-1] = true
			saida = append(saida, n-1)
		}
	}
	return saida, nil
}
