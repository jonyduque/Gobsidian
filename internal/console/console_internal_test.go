package console

import (
	"bytes"
	"os"
	"testing"
)

// TestAmbienteProibeCor exercita a decisao de ambiente SOZINHA.
//
// A versao anterior deste teste passava por SupportsColor com um arquivo
// temporario, e nao podia falhar: um arquivo comum ja nao recebe cor por nao
// ser dispositivo de caractere, entao apagar a checagem de NO_COLOR nao mudava
// o resultado. O teste reportava cobertura que nao existia -- exatamente o
// caso que o CLAUDE.md descreve como pior que teste ausente.
//
// Aqui nao ha essa saida: a funcao le ambiente e mais nada, entao a unica
// coisa que pode fazer o resultado mudar e a regra que se quer verificar.
func TestAmbienteProibeCor(t *testing.T) {
	casos := []struct {
		nome     string
		noColor  string
		term     string
		proibido bool
	}{
		{nome: "sem nada definido", proibido: false},
		{nome: "NO_COLOR=1", noColor: "1", proibido: true},
		// A convencao no-color.org e explicita: o que conta e a variavel
		// ESTAR definida, nao o valor dela. NO_COLOR=0 desliga a cor.
		{nome: "NO_COLOR=0 tambem desliga", noColor: "0", proibido: true},
		{nome: "TERM=dumb", term: "dumb", proibido: true},
		{nome: "TERM comum nao desliga", term: "xterm-256color", proibido: false},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			t.Setenv("NO_COLOR", c.noColor)
			t.Setenv("TERM", c.term)

			if got := ambienteProibeCor(); got != c.proibido {
				t.Errorf("ambienteProibeCor() = %v, quer %v (NO_COLOR=%q TERM=%q)",
					got, c.proibido, c.noColor, c.term)
			}
		})
	}
}

// modoFalso implementa os.FileInfo com o modo que o teste quiser. Existe
// porque a alternativa -- abrir um dispositivo de caractere de verdade -- nao
// e portavel entre os tres sistemas, e sem ele o caso "e terminal" nao teria
// como ser exercitado.
type modoFalso struct {
	os.FileInfo
	modo os.FileMode
}

func (m modoFalso) Mode() os.FileMode { return m.modo }

// TestEhTerminal cobre os dois lados da regra. O lado FALSO ja era coberto por
// TestArquivoComumNaoRecebeCor; o lado VERDADEIRO nao era por ninguem, e era
// justamente ele que faltava para a regra poder reprovar sob mutacao.
func TestEhTerminal(t *testing.T) {
	casos := []struct {
		nome string
		modo os.FileMode
		quer bool
	}{
		{nome: "arquivo comum", modo: 0o644, quer: false},
		{nome: "diretorio", modo: os.ModeDir | 0o755, quer: false},
		{nome: "pipe nomeado", modo: os.ModeNamedPipe, quer: false},
		{nome: "dispositivo de caractere", modo: os.ModeCharDevice | os.ModeDevice, quer: true},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := ehTerminal(modoFalso{modo: c.modo}); got != c.quer {
				t.Errorf("ehTerminal(%v) = %v, quer %v", c.modo, got, c.quer)
			}
		})
	}
}

// TestSequenciasEmitidas fixa os BYTES que saem quando ha cor.
//
// Sem ele, todo o resto da suite so prova que a saida sem cor esta certa --
// e um pacote de formatacao cuja unica cobertura e "nao formatou" reporta
// confianca que nao tem. Um terminal de verdade nao esta disponivel em teste,
// entao o Stream e construido com a cor ja ligada, que e exatamente o estado
// em que SupportsColor o deixaria.
func TestSequenciasEmitidas(t *testing.T) {
	var buf bytes.Buffer
	s := &Stream{w: &buf, color: true}

	casos := []struct {
		nome string
		fn   func()
		quer string
	}{
		{"OK", func() { s.OK("pronto") }, "\x1b[1;32m[OK]\x1b[0m pronto\n"},
		{"Warn", func() { s.Warn("cuidado") }, "\x1b[1;33m[!]\x1b[0m cuidado\n"},
		{"Err", func() { s.Err("falhou") }, "\x1b[1;31m[!]\x1b[0m falhou\n"},
		{"Info", func() { s.Info("nota") }, "\x1b[2m[i]\x1b[0m nota\n"},
		{"Item", func() { s.Item("linha") }, "\x1b[36m[*]\x1b[0m linha\n"},
		{"Step", func() { s.Step("indo") }, "\x1b[34m[...]\x1b[0m indo\n"},
		{"Detail", func() { s.Detail("sob") }, "     \x1b[2msob\x1b[0m\n"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			buf.Reset()
			c.fn()
			if got := buf.String(); got != c.quer {
				t.Errorf("%s emitiu %q, quer %q", c.nome, got, c.quer)
			}
		})
	}

	// Um unico SGR combinado, nao dois encadeados: "\x1b[1;31m" e uma escrita,
	// "\x1b[1m\x1b[31m" sao duas, e um terminal que interrompe entre elas
	// mostra o texto meio formatado.
	if got := s.style("x", codeBold, codeRed); got != "\x1b[1;31mx\x1b[0m" {
		t.Errorf("style combinado = %q, quer sequencia unica \x1b[1;31m", got)
	}
}
