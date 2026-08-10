// Package console formata a saida dos comandos de CLI: os marcadores de
// estado, os realces do texto de ajuda, e a decisao de usar ou nao cor.
//
// O pacote se chama console, e nao utils, porque a preocupacao dele tem nome:
// e a saida para um terminal humano. Um pacote chamado utils atrai qualquer
// coisa que nao coube em outro lugar, e deixa de ser possivel dizer o que ele
// faz sem abri-lo.
//
// # Os marcadores continuam em ASCII
//
// [OK], [!], [i], [*] e [...] sao os mesmos de antes. A cor SOMA a eles, nunca
// os substitui: um console PowerShell em CP-850 renderiza qualquer coisa fora
// do ASCII como lixo, e `doctor` e justamente o comando que alguem roda quando
// ja esta confuso. Se a cor for descartada -- por redirecionamento, por
// NO_COLOR, por terminal que nao a suporta -- a saida continua legivel e
// diferenciavel, porque a informacao esta no marcador e a cor so a reforca.
//
// # stdout de `serve` nao passa por aqui
//
// stdout pertence ao JSON-RPC. Uma sequencia ANSI escrita nele corrompe a
// sessao exatamente como um fmt.Println corromperia, e o sintoma e o mesmo:
// o servidor some do host sem erro nenhum. Este pacote e para `doctor`,
// `version`, `index`, `search` e `inspect`, que sao comandos de CLI, e para o
// stderr. TestNadaColoreOStdoutDoServe cobre isso.
package console

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// Codigos SGR usados. Ficam como constantes nomeadas porque "\033[1;33m"
// espalhado pelo codigo nao diz a ninguem que aquilo e amarelo em negrito.
const (
	reset = "\033[0m"

	codeBold   = "1"
	codeDim    = "2"
	codeItalic = "3"

	codeRed    = "31"
	codeGreen  = "32"
	codeYellow = "33"
	codeBlue   = "34"
	codeCyan   = "36"
)

// sgr monta uma sequencia ANSI unica com todos os codigos, em vez de
// concatenar sequencias. "\033[1;31m" e uma escrita; "\033[1m\033[31m" sao
// duas, e um terminal que interrompe entre elas mostra o texto meio formatado.
func sgr(codes ...string) string {
	return "\033[" + strings.Join(codes, ";") + "m"
}

// Stream e uma saida de console com a decisao de cor JA TOMADA.
//
// A decisao e tomada uma vez, na construcao, e nao a cada linha impressa. Duas
// razoes, e a segunda importa mais: um os.Stat por linha e desperdicio, mas
// uma decisao reavaliada a cada linha pode MUDAR no meio de uma saida -- basta
// alguem mexer em NO_COLOR entre duas chamadas -- e metade do relatorio sai
// colorida e a outra metade nao.
type Stream struct {
	w     io.Writer
	color bool
}

// New devolve um Stream para w, decidindo sobre cor a partir do PROPRIO w.
//
// Decidir a partir de w, e nao de os.Stdout sempre, e o que faz
// `gobsidian doctor > relatorio.txt` gravar um arquivo limpo e AINDA ASSIM
// colorir os erros que vao para o stderr do terminal. Quem olha so o stdout
// para decidir os dois erra sempre um dos casos.
func New(w io.Writer) *Stream {
	return &Stream{w: w, color: SupportsColor(w)}
}

// NewPlain devolve um Stream que nunca colore, qualquer que seja o destino.
// Serve para teste e para saida destinada a ser consumida por outro programa.
func NewPlain(w io.Writer) *Stream {
	return &Stream{w: w, color: false}
}

// Colored informa se este Stream esta emitindo sequencias ANSI.
func (s *Stream) Colored() bool { return s.color }

// Writer devolve o destino, para quem precisa escrever algo que este pacote
// nao formata (um bloco JSON, por exemplo).
func (s *Stream) Writer() io.Writer { return s.w }

func (s *Stream) style(text string, codes ...string) string {
	if !s.color || len(codes) == 0 {
		return text
	}
	return sgr(codes...) + text + reset
}

// Bold realca um trecho DENTRO de uma linha.
func (s *Stream) Bold(text string) string { return s.style(text, codeBold) }

// Dim apaga um trecho DENTRO de uma linha, para informacao secundaria.
func (s *Stream) Dim(text string) string { return s.style(text, codeDim) }

// Italic aplica italico a um trecho DENTRO de uma linha.
//
// E o menos portavel dos tres realces: o console legado do Windows (conhost)
// o ignora, enquanto o Windows Terminal o aplica. Ignorar e aceitavel -- o
// texto aparece sem enfase, nao quebrado -- entao ele fica, mas nao carregue
// informacao apenas nele.
func (s *Stream) Italic(text string) string { return s.style(text, codeItalic) }

// Os metodos abaixo imprimem uma linha com marcador. O marcador vai colorido,
// o texto vai como veio: colorir a mensagem inteira de vermelho torna
// ilegivel um caminho longo, e o que precisa saltar aos olhos e o estado.
func (s *Stream) printf(marker string, codes []string, format string, a ...any) {
	_, _ = fmt.Fprintf(s.w, "%s %s\n", s.style(marker, codes...), fmt.Sprintf(format, a...))
}

// OK marca sucesso.
func (s *Stream) OK(format string, a ...any) {
	s.printf("[OK]", []string{codeBold, codeGreen}, format, a...)
}

// Warn marca aviso e falha. Os dois usam [!], como antes deste pacote: o
// marcador nao distingue os dois casos, a cor distingue, e um terminal sem
// cor volta ao comportamento que o projeto sempre teve.
func (s *Stream) Warn(format string, a ...any) {
	s.printf("[!]", []string{codeBold, codeYellow}, format, a...)
}

// Err marca falha.
func (s *Stream) Err(format string, a ...any) {
	s.printf("[!]", []string{codeBold, codeRed}, format, a...)
}

// Info marca informacao secundaria.
func (s *Stream) Info(format string, a ...any) {
	s.printf("[i]", []string{codeDim}, format, a...)
}

// Item marca um item de listagem ou um numero medido.
func (s *Stream) Item(format string, a ...any) {
	s.printf("[*]", []string{codeCyan}, format, a...)
}

// Step marca etapa em andamento.
func (s *Stream) Step(format string, a ...any) {
	s.printf("[...]", []string{codeBlue}, format, a...)
}

// Line imprime sem marcador nenhum, respeitando o destino do Stream.
func (s *Stream) Line(format string, a ...any) {
	_, _ = fmt.Fprintf(s.w, format+"\n", a...)
}

// Detail imprime uma linha de detalhe indentada sob o item anterior, em tom
// apagado. E o segundo nivel do relatorio do doctor.
func (s *Stream) Detail(format string, a ...any) {
	_, _ = fmt.Fprintf(s.w, "     %s\n", s.Dim(fmt.Sprintf(format, a...)))
}

// SupportsColor decide se w aceita sequencias ANSI.
//
// Ordem deliberada: NO_COLOR primeiro, porque e uma escolha explicita de quem
// esta rodando o comando e nao deve ser sobreposta por deteccao nenhuma.
// Depois TERM=dumb. So entao a pergunta tecnica de w ser um terminal.
//
// Um w que nao seja *os.File nunca recebe cor: e um buffer, um pipe interno ou
// um teste, e em nenhum desses casos ha alguem lendo escape de terminal.
func SupportsColor(w io.Writer) bool {
	if ambienteProibeCor() {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	if !ehTerminal(info) {
		return false
	}
	return enableVirtualTerminal(f)
}

// ambienteProibeCor responde a parte da decisao que depende SO do ambiente,
// separada do resto de proposito.
//
// Ela existe separada porque a alternativa nao e testavel: um teste que
// exercitasse NO_COLOR atraves de SupportsColor precisaria de um terminal de
// verdade, e sem ele escreveria contra um arquivo comum -- que ja nao recebe
// cor por NAO SER dispositivo de caractere. O teste passaria com a checagem de
// NO_COLOR apagada, reportando cobertura que nao existe. Provado por mutacao:
// ver TestAmbienteProibeCor.
func ambienteProibeCor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	return os.Getenv("TERM") == "dumb"
}

// ehTerminal responde se o destino e um dispositivo de caractere, e nao um
// arquivo comum ou um pipe. E o que separa `doctor` no terminal de
// `doctor > relatorio.txt`.
//
// Recebe um os.FileInfo, e nao um *os.File, para poder ser verificada: com um
// arquivo de verdade, esta regra e INDISTINGUIVEL da checagem seguinte --
// ativarVT tambem reprova um handle de arquivo, entao apagar esta aqui nao
// mudava o resultado e o teste passava com a regra removida. Medido, nao
// suposto: a mutacao de "== 0" para "!= 0" saiu verde antes desta extracao.
// Com FileInfo, um teste pode apresentar um modo de dispositivo de caractere
// sem precisar de um terminal de verdade.
func ehTerminal(info os.FileInfo) bool {
	return info.Mode()&os.ModeCharDevice != 0
}

// habilitado guarda o resultado de enableVirtualTerminal por descritor, para
// nao repetir a syscall a cada Stream construido. O mapa e minusculo -- na
// pratica duas entradas, stdout e stderr.
var (
	habilitadoMu sync.Mutex
	habilitado   = map[uintptr]bool{}
)

// enableVirtualTerminal prepara f para receber ANSI e informa se deu certo.
// A implementacao real esta em vt_windows.go; nos demais sistemas o terminal
// ja interpreta ANSI e vt_other.go devolve true sem fazer nada.
func enableVirtualTerminal(f *os.File) bool {
	fd := f.Fd()

	habilitadoMu.Lock()
	defer habilitadoMu.Unlock()
	if v, ok := habilitado[fd]; ok {
		return v
	}
	v := ativarVT(f)
	habilitado[fd] = v
	return v
}
