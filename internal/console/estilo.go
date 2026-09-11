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
// Os marcadores de estado ([OK], [!], [i]) continuam ASCII em qualquer caso.
// Eles aparecem em log, em redirecionamento e em `doctor`, que sao os lugares
// onde a regra vale sem exececao. O que ganha desenho e SO a lista
// interativa, que por definicao so aparece quando ha um terminal de verdade.
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

	Separador: "·", Enter: "⏎", Setas: "↑↓",

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

	Separador: "|", Enter: "enter", Setas: "setas",

	Pergunta: "?", Resposta: ">", Secao: "*",

	Reticencia: "...",
}

// forcarGlifos existe SO para o teste de previa mostrar os dois conjuntos sem
// depender da code page da maquina onde ele roda. Nulo em producao.
var forcarGlifos *Glifos

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
	switch os.Getenv(VarDeEstilo) {
	case "1", "true", "sim":
		return glifosUnicode
	case "0", "false", "nao":
		return glifosASCII
	}
	if suportaUnicode() {
		return glifosUnicode
	}
	return glifosASCII
}
