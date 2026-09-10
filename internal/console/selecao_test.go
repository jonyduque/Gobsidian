package console

import (
	"bytes"
	"strings"
	"testing"
)

// A tabela de teclas e o que decide o comportamento da lista, e ela e testavel
// sem terminal -- que e a razao de interpretar existir separada de Selecionar.
func TestInterpretarTeclas(t *testing.T) {
	const total = 3
	casos := []struct {
		nome   string
		bytes  []byte
		cursor int
		acao   acaoDeTecla
		alvo   int
		porQue string
	}{
		{"seta baixo", []byte{0x1b, '[', 'B'}, 0, mover, 1, "a sequencia ANSI e a mesma nas duas plataformas"},
		{"seta cima", []byte{0x1b, '[', 'A'}, 1, mover, 0, ""},
		{"seta cima no topo da volta", []byte{0x1b, '[', 'A'}, 0, mover, 2, "a lista e circular; parar no topo faz o usuario achar que travou"},
		{"seta baixo no fim da volta", []byte{0x1b, '[', 'B'}, 2, mover, 0, ""},
		{"j desce", []byte{'j'}, 0, mover, 1, "quem vem do vim nao procura a seta"},
		{"k sobe", []byte{'k'}, 1, mover, 0, ""},
		{"espaco marca", []byte{' '}, 1, alternar, 1, ""},
		{"a marca todos", []byte{'a'}, 0, alternarTodos, 0, ""},
		{"enter confirma", []byte{'\r'}, 0, confirmar, 0, ""},
		{"newline tambem confirma", []byte{'\n'}, 0, confirmar, 0, "terminal que traduz CR para LF nao pode travar a lista"},
		{"esc cancela", []byte{0x1b}, 0, cancelar, 0, "esc sozinho; com bytes atras e sequencia"},
		{"q cancela", []byte{'q'}, 0, cancelar, 0, ""},
		{"ctrl-c cancela", []byte{0x03}, 0, cancelar, 0, "em modo bruto o Ctrl-C chega como byte, e nao como sinal"},
		{"tecla desconhecida nao faz nada", []byte{'z'}, 1, nada, 1, ""},
		{"sequencia ANSI desconhecida nao faz nada", []byte{0x1b, '[', 'C'}, 1, nada, 1, "seta para o lado nao move a lista"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			acao, alvo := interpretar(c.bytes, c.cursor, total)
			if acao != c.acao {
				t.Errorf("acao = %v, queria %v. %s", acao, c.acao, c.porQue)
			}
			if acao == mover && alvo != c.alvo {
				t.Errorf("cursor = %d, queria %d. %s", alvo, c.alvo, c.porQue)
			}
		})
	}
}

// O caminho digitado existe porque `install` roda em terminal de IDE, sob pipe
// e em CI. Ele responde a MESMA pergunta, e "nenhum" precisa ser dizivel.
func TestSelecionarDigitando(t *testing.T) {
	opcoes := []Opcao{{Rotulo: "a"}, {Rotulo: "b"}, {Rotulo: "c"}}

	casos := []struct {
		resposta string
		quer     []int
		erro     bool
		porQue   string
	}{
		{"", nil, false, "vazio e NENHUM, e nenhum e uma escolha valida"},
		{"0", nil, false, "zero tambem e nenhum"},
		{"2", []int{1}, false, ""},
		{"1 3", []int{0, 2}, false, "espaco separa"},
		{"1,3", []int{0, 2}, false, "virgula tambem"},
		{"3 1", []int{2, 0}, false, "a ordem digitada e respeitada"},
		{"2 2", []int{1}, false, "repetir nao duplica"},
		{"*", []int{0, 1, 2}, false, "asterisco e todos"},
		{"4", nil, true, "fora da lista reprova em vez de escolher outro"},
		{"x", nil, true, "nao-numero reprova"},
	}

	for _, c := range casos {
		t.Run(c.resposta, func(t *testing.T) {
			got, err := SelecionarDigitando(c.resposta, opcoes)
			if c.erro {
				if err == nil {
					t.Fatalf("SelecionarDigitando(%q) = %v, queria erro. %s", c.resposta, got, c.porQue)
				}
				return
			}
			if err != nil {
				t.Fatalf("SelecionarDigitando(%q): %v", c.resposta, err)
			}
			if len(got) != len(c.quer) {
				t.Fatalf("SelecionarDigitando(%q) = %v, queria %v. %s", c.resposta, got, c.quer, c.porQue)
			}
			for i := range got {
				if got[i] != c.quer[i] {
					t.Fatalf("SelecionarDigitando(%q) = %v, queria %v. %s", c.resposta, got, c.quer, c.porQue)
				}
			}
		})
	}
}

// TestMolduraFechaEmTodasAsLinhas e o teste que faltava quando o desenho tinha
// margens ad-hoc: a borda de baixo saia um caractere mais longa que as outras,
// e no conjunto Unicode a barra da direita colava no texto.
//
// A invariante e uma so e vale para os dois conjuntos de glifos: TODA linha da
// moldura ocupa a MESMA largura visivel.
//
// Prova de mutacao, rodada em 2026-09-09: tirar o `ajustar` das linhas de corpo
// (`g.Vertical + c + g.Vertical`) reprova nos dois conjuntos, nomeando as duas
// larguras.
//
// A PRIMEIRA prova que escrevi aqui estava errada e ficou registrada por isso:
// eu disse que trocar `+ margem` por `+ 0` reprovaria, e ela NAO reprova --
// `ajustar` preenche tudo ate `interno` de qualquer jeito, entao a margem so
// tira o respiro da direita e a moldura continua fechando. Prova de mutacao que
// nao foi rodada e palpite com cara de prova.
//
// O que este caso NAO cobre: o teto de 78 celulas. Nenhum rotulo aqui chega
// perto, e tirar o `cortar` de `ajustar` passa incolume. Cobrir o teto pediria
// um rotulo gigante, e o corte tem consequencia visual, nao estrutural.
func TestMolduraFechaEmTodasAsLinhas(t *testing.T) {
	opcoes := []Opcao{
		{Rotulo: "curto"},
		{Rotulo: "um rotulo consideravelmente mais longo que o outro", Nota: "(ja configurado)"},
		{Rotulo: "com acento: Ação Direta", Nota: "(aberto agora)"},
	}
	marcadas := []bool{true, false, true}

	for _, caso := range []struct {
		nome   string
		glifos Glifos
	}{
		{"unicode", glifosUnicode},
		{"ascii", glifosASCII},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			forcarGlifos = &caso.glifos
			defer func() { forcarGlifos = nil }()

			for _, cursor := range []int{-1, 0, 2} {
				var buf bytes.Buffer
				s := NewPlain(&buf)
				desenhar(s, opcoes, marcadas, cursor, "Quais cofres configurar?", false)

				linhas := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
				if len(linhas) != len(opcoes)+3 {
					t.Fatalf("cursor %d: %d linhas, queria %d (topo + %d opcoes + branca + rodape)",
						cursor, len(linhas), len(opcoes)+3, len(opcoes))
				}
				quer := larguraVisivel(linhas[0])
				for i, l := range linhas {
					if got := larguraVisivel(l); got != quer {
						t.Errorf("cursor %d, linha %d tem largura %d e a primeira tem %d; a moldura nao fecha:\n%s",
							cursor, i, got, quer, buf.String())
					}
				}
			}
		})
	}
}

// larguraVisivel e o que sustenta o alinhamento: uma nota esmaecida tem mais
// BYTES que celulas, e alinhar por len() empurraria a borda para dentro do
// texto.
func TestLarguraVisivelIgnoraSequenciaANSI(t *testing.T) {
	casos := map[string]int{
		"abc":                    3,
		"\x1b[2mabc\x1b[0m":      3,
		"Ação":                   4,
		"\x1b[1m\x1b[2mx\x1b[0m": 1,
		"":                       0,
	}
	for entrada, quer := range casos {
		if got := larguraVisivel(entrada); got != quer {
			t.Errorf("larguraVisivel(%q) = %d, queria %d", entrada, got, quer)
		}
	}
}

// TestMolduraAchataConteudoMultilinha e o defeito que `search` mostrou na
// primeira vez que passou pela moldura: o trecho de um resultado traz a quebra
// de linha do PROPRIO arquivo, e ela jogava metade do texto para fora da caixa.
//
// Prova de mutacao: tirar o laco que chama achatar em Moldura faz o caso
// reprovar, porque a saida ganha mais linhas que o bloco tem.
func TestMolduraAchataConteudoMultilinha(t *testing.T) {
	var buf bytes.Buffer
	s := NewPlain(&buf)
	forcarGlifos = &glifosASCII
	defer func() { forcarGlifos = nil }()

	s.Bloco("Resultados", []string{
		"  nota.md",
		"    ... ---\naliases: [P3]\ntags: [civil]\n---\n# Nota ...",
	}, "")

	linhas := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(linhas) != 4 {
		t.Fatalf("bloco de 2 corpos saiu com %d linhas, queria 4 (topo + 2 + rodape):\n%s",
			len(linhas), buf.String())
	}
	quer := larguraVisivel(linhas[0])
	for i, l := range linhas {
		if got := larguraVisivel(l); got != quer {
			t.Errorf("linha %d tem largura %d e a primeira tem %d:\n%s", i, got, quer, buf.String())
		}
	}
}

// A cor tem de sobreviver ao achatamento: achatar sem cuidado apagaria o ESC
// junto com as quebras, e o bloco sairia sem realce nenhum.
func TestAchatarPreservaSequenciaANSI(t *testing.T) {
	got := achatar("a\x1b[2mb\x1b[0m\nc")
	quer := "a\x1b[2mb\x1b[0m c"
	if got != quer {
		t.Errorf("achatar = %q, queria %q", got, quer)
	}
}
