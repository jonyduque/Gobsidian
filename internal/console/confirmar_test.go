package console

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

// TestInterpretarBotao trava a tabela de teclas do modal. Ela e o
// comportamento inteiro da pergunta; o resto de Confirmar e desenho e leitura.
func TestInterpretarBotao(t *testing.T) {
	casos := []struct {
		nome       string
		bytes      []byte
		simAntes   bool
		querAcao   acaoDeTecla
		querValor  bool
		querMoveu  bool
		ignorarVal bool
	}{
		{nome: "seta esquerda foca Sim", bytes: []byte{0x1b, '[', 'D'}, simAntes: false, querAcao: mover, querValor: true},
		{nome: "seta direita foca Nao", bytes: []byte{0x1b, '[', 'C'}, simAntes: true, querAcao: mover, querValor: false},
		{nome: "tab alterna", bytes: []byte{'\t'}, simAntes: true, querAcao: mover, querValor: false},
		{nome: "espaco alterna", bytes: []byte{' '}, simAntes: false, querAcao: mover, querValor: true},
		{nome: "s responde sim", bytes: []byte{'s'}, simAntes: false, querAcao: confirmar, querValor: true},
		{nome: "n responde nao", bytes: []byte{'n'}, simAntes: true, querAcao: confirmar, querValor: false},
		{nome: "enter confirma o foco", bytes: []byte{'\r'}, simAntes: true, querAcao: confirmar, querValor: true},
		{nome: "enter confirma o foco em Nao", bytes: []byte{'\n'}, simAntes: false, querAcao: confirmar, querValor: false},
		{nome: "esc cancela", bytes: []byte{0x1b}, simAntes: true, querAcao: cancelar, ignorarVal: true},
		{nome: "ctrl-c cancela", bytes: []byte{0x03}, simAntes: true, querAcao: cancelar, ignorarVal: true},
		{nome: "tecla desconhecida nao faz nada", bytes: []byte{'z'}, simAntes: true, querAcao: nada, ignorarVal: true},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			acao, valor := interpretarBotao(c.bytes, c.simAntes)
			if acao != c.querAcao {
				t.Fatalf("acao = %v, queria %v", acao, c.querAcao)
			}
			if !c.ignorarVal && valor != c.querValor {
				t.Errorf("valor = %v, queria %v", valor, c.querValor)
			}
		})
	}
}

// TestBotoesMostramOsDoisEstados: a resposta corrente precisa estar visivel --
// e a razao de existir o modal. Num terminal sem cor ela se distingue pelo
// GLIFO, a mesma regra das caixas de selecao.
func TestBotoesMostramOsDoisEstados(t *testing.T) {
	forcarGlifos = &glifosASCII
	defer func() { forcarGlifos = nil }()

	desenho := func(sim bool) string {
		var buf bytes.Buffer
		desenharBotoes(NewPlain(&buf), "Encerrar estes processos?", []string{"pid 42  daemon  C:\\Cofre"}, sim, false)
		return buf.String()
	}

	comSim, comNao := desenho(true), desenho(false)
	if !strings.Contains(comSim, "[ > Sim ]") || !strings.Contains(comSim, "[   Nao ]") {
		t.Errorf("com foco em Sim, a linha de botoes saiu:\n%s", comSim)
	}
	if !strings.Contains(comNao, "[ > Nao ]") || !strings.Contains(comNao, "[   Sim ]") {
		t.Errorf("com foco em Nao, a linha de botoes saiu:\n%s", comNao)
	}
	if !strings.Contains(comSim, "pid 42") {
		t.Errorf("os itens da decisao nao entraram na moldura:\n%s", comSim)
	}

	// As duas telas tem a MESMA largura: o glifo de foco ocupa a coluna que no
	// outro botao e espaco. Sem isso a moldura pisca a cada seta.
	if l1, l2 := len(strings.Split(comSim, "\n")), len(strings.Split(comNao, "\n")); l1 != l2 {
		t.Errorf("as duas telas tem alturas diferentes: %d e %d", l1, l2)
	}
	for i, l := range strings.Split(strings.TrimRight(comSim, "\n"), "\n") {
		outra := strings.Split(strings.TrimRight(comNao, "\n"), "\n")[i]
		if colunasNaTela(l) != colunasNaTela(outra) {
			t.Errorf("linha %d muda de largura entre os dois focos:\n%q\n%q", i, l, outra)
		}
	}
}

// TestRodapeDoModalEnsinaAsTeclasCertas: a primeira redação mostrava o par
// vertical de setas, que é o da lista de seleção -- o modal anda para os lados.
// Um rodapé que ensina a tecla errada é pior que rodapé nenhum: quem aperta
// para cima acha que a pergunta travou.
func TestRodapeDoModalEnsinaAsTeclasCertas(t *testing.T) {
	t.Setenv(VarDeEstilo, "1")
	forcarGlifos = &glifosUnicode
	defer func() { forcarGlifos = nil }()

	var buf bytes.Buffer
	desenharBotoes(NewPlain(&buf), "Encerrar?", nil, true, false)
	saida := buf.String()

	if !strings.Contains(saida, glifosUnicode.SetasLado) {
		t.Errorf("o rodapé não mostra as setas de lado (%q):\n%s", glifosUnicode.SetasLado, saida)
	}
	if strings.Contains(saida, glifosUnicode.Setas) {
		t.Errorf("o rodapé mostra o par vertical (%q), que é o da lista de seleção:\n%s", glifosUnicode.Setas, saida)
	}
	// O rótulo do botão é português: ele passa por adaptarTexto como o resto, e
	// só perde o acento onde o console não aguenta.
	if !strings.Contains(saida, "Não") {
		t.Errorf("o botão perdeu o acento num console que aguenta UTF-8:\n%s", saida)
	}
}

// TestConfirmarSemTerminalDevolveErrSemTerminal: com a entrada vinda de um
// cano ou de um arquivo, o modal nao existe e quem chama cai na pergunta
// digitada. O padrao volta junto para o chamador nao ter de inventar um.
func TestConfirmarSemTerminalDevolveErrSemTerminal(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	var buf bytes.Buffer
	got, err := Confirmar(NewPlain(&buf), f, "Encerrar?", nil, true)
	if !errors.Is(err, ErrSemTerminal) {
		t.Fatalf("Confirmar() com entrada que nao e terminal = (%v, %v), queria ErrSemTerminal", got, err)
	}
	if !got {
		t.Error("Confirmar() nao devolveu o padrao junto com ErrSemTerminal")
	}
	if buf.Len() != 0 {
		t.Errorf("Confirmar() desenhou sem terminal:\n%s", buf.String())
	}
}
