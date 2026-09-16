package console

import "os"

// Glifos e o conjunto de desenho de uma lista de selecao.
//
// # Por que ha dois conjuntos, e nao um
//
// A regra do projeto e saida em ASCII puro, e a razao dela esta no comentario
// de pacote: um console PowerShell em CP-850 renderiza qualquer coisa fora do
// ASCII como LIXO, e quem roda `install` numa maquina recem-formatada esta
// exatamente nesse console. Uma caixa bonita que sai como "ÔòÂÔöÇ" e pior que
// uma caixa feia.
//
// Mas a regra existe por causa da CODIFICACAO, e nao por gosto. Onde a
// codificacao e UTF-8 -- Windows Terminal, VS Code, qualquer terminal de Unix
// com locale UTF-8 -- o motivo nao se aplica, e o desenho pode ser o bom.
// Entao a escolha e MEDIDA (ver suportaUnicode), e nao adivinhada: consulta a
// code page real do console no Windows e o locale no resto.
//
// Os marcadores de estado seguem a MESMA medicao desde 2026-09-16 (ver
// Marcadores, mais abaixo): emoji onde o console aguenta, escritos onde nao
// aguenta. A regra antiga -- ASCII em qualquer caso -- valia porque a medicao
// nao existia ainda.
type Glifos struct {
	CantoSupEsq string
	CantoSupDir string
	CantoInfEsq string
	CantoInfDir string
	Horizontal  string
	Vertical    string

	Cursor     string
	Marcada    string
	Desmarcada string

	Separador string
	Enter     string
	Setas     string
	// SetasLado e o par horizontal, para a pergunta de dois botoes. A lista de
	// selecao anda para cima e para baixo e o modal anda para os lados: mostrar
	// o par errado ensina a tecla errada, que foi o que a previa de 2026-09-16
	// mostrou.
	SetasLado string

	// Os glifos da conversa: pergunta em aberto, resposta ja dada, e o
	// marcador de secao.
	Pergunta string
	Resposta string
	Secao    string

	// Reticencia marca conteudo cortado no teto da moldura.
	Reticencia string
}

// glifosUnicode e o desenho bom: cantos arredondados, como as caixas do
// lipgloss.
var glifosUnicode = Glifos{
	CantoSupEsq: "╭", CantoSupDir: "╮",
	CantoInfEsq: "╰", CantoInfDir: "╯",
	Horizontal: "─", Vertical: "│",

	Cursor: "▸", Marcada: "◉", Desmarcada: "○",

	Separador: "·", Enter: "⏎", Setas: "↑↓", SetasLado: "←→",

	Pergunta: "◆", Resposta: "✓", Secao: "▪",

	Reticencia: "…",
}

// glifosASCII e o mesmo desenho onde so ASCII sobrevive. Nao e uma versao
// degradada por descuido: cada peca tem um equivalente escolhido para a caixa
// continuar fechando e as colunas continuarem alinhadas.
var glifosASCII = Glifos{
	CantoSupEsq: "+", CantoSupDir: "+",
	CantoInfEsq: "+", CantoInfDir: "+",
	Horizontal: "-", Vertical: "|",

	Cursor: ">", Marcada: "x", Desmarcada: " ",

	Separador: "|", Enter: "enter", Setas: "setas", SetasLado: "setas",

	Pergunta: "?", Resposta: ">", Secao: "*",

	Reticencia: "...",
}

// forcarGlifos existe SO para o teste de previa mostrar os dois conjuntos sem
// depender da code page da maquina onde ele roda. Nulo em producao.
var forcarGlifos *Glifos

// Marcadores e o conjunto de simbolos de estado que abrem cada linha.
//
// # Por que ha dois conjuntos
//
// A mesma razao dos glifos da moldura: onde o console aguenta UTF-8, um
// emoji diz o estado antes de a palavra ser lida; onde nao aguenta, ele sai
// como lixo e o marcador escrito e o que resta. A escolha e a MESMA medicao
// (modoUnicode), e nao uma segunda.
//
// A cor SOMA ao marcador nos dois conjuntos, e nunca o substitui: um terminal
// sem cor -- NO_COLOR, redirecionamento -- continua distinguindo OK de aviso
// pelo simbolo.
//
// # Largura
//
// Um emoji ocupa DUAS colunas, e "[OK] " ocupa cinco. Os marcadores de emoji
// carregam o enchimento que falta para o texto comecar na mesma coluna nos
// dois modos -- e o que mantem alinhada a linha de Detail, que indenta cinco
// espacos, e as continuacoes que o proprio detalhe traz.
type Marcadores struct {
	OK    string
	Aviso string
	Erro  string
	Info  string
	Item  string
	Passo string
}

// marcadoresTexto e o conjunto de sempre, o que sobrevive a qualquer code page.
var marcadoresTexto = Marcadores{
	OK:    "[OK]",
	Aviso: "[!]",
	Erro:  "[!]",
	Info:  "[i]",
	Item:  "[*]",
	Passo: "[...]",
}

// marcadoresEmoji distingue AVISO de ERRO, que o conjunto de texto nao
// distingue -- os dois sao "[!]" la, e so a cor os separa.
var marcadoresEmoji = Marcadores{
	OK:    "✅  ",
	Aviso: "⚠️  ",
	Erro:  "❌  ",
	Info:  "ℹ️  ",
	Item:  "🔹  ",
	Passo: "⏳  ",
}

// forcarMarcadores existe SO para teste, como forcarGlifos.
var forcarMarcadores *Marcadores

// MarcadoresDaSaida escolhe o conjunto pela mesma medicao dos glifos.
func MarcadoresDaSaida() Marcadores {
	if forcarMarcadores != nil {
		return *forcarMarcadores
	}
	if modoUnicode() {
		return marcadoresEmoji
	}
	return marcadoresTexto
}

// VarDeEstilo permite decidir na mao quando a deteccao erra.
//
// GOBSIDIAN_UNICODE=1 forca o desenho bom, =0 forca o ASCII. Existe porque a
// deteccao e um PALPITE INFORMADO sobre o terminal, e palpite erra: em
// 2026-09-11 o dono viu a moldura arredondada no PowerShell e a ASCII no
// nushell, na MESMA maquina e no mesmo Windows Terminal. Quem esta na frente da
// tela sabe mais que a heuristica, e precisa poder dizer isso.
const VarDeEstilo = "GOBSIDIAN_UNICODE"

// GlifosDaSaida escolhe o conjunto a partir do que o terminal aguenta.
func GlifosDaSaida() Glifos {
	if forcarGlifos != nil {
		return *forcarGlifos
	}
	if modoUnicode() {
		return glifosUnicode
	}
	return glifosASCII
}

// modoUnicode e a decisao unica sobre o que esta saida aguenta fora do ASCII.
//
// Os glifos da moldura e o ACENTO do texto respondem a mesma pergunta, e antes
// de 2026-09-16 so os glifos a faziam: o desenho era medido e o texto era
// escrito sem acento sempre, ate no Windows Terminal. Duas respostas para a
// mesma pergunta divergem; esta e a conta unica. Ver adaptarTexto.
func modoUnicode() bool {
	switch os.Getenv(VarDeEstilo) {
	case "1", "true", "sim":
		return true
	case "0", "false", "nao":
		return false
	}
	return suportaUnicode()
}
