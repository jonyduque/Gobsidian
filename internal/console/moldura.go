package console

import "strings"

// Moldura desenha um bloco com titulo na borda de cima e, opcionalmente, um
// rodape na de baixo.
//
// # Uma conta so
//
// A lista de selecao desenhava a propria moldura, e o resumo da instalacao
// imprimia linhas soltas. Duas formas de mostrar um bloco na mesma tela, e a
// menos usada divergiria -- que e a armadilha registrada no CLAUDE.md sobre
// duas contas da mesma regra. Aqui a moldura e UMA, e quem quer um bloco pede
// um bloco.
//
// Devolve as linhas em vez de imprimir porque quem chama precisa saber QUANTAS
// foram: a lista interativa sobe o cursor por esse numero para se redesenhar.
//
// # Largura
//
// Toda linha e preenchida ate a mesma largura interna, medida do conteudo mais
// largo -- ver larguraVisivel, que ignora as sequencias ANSI. A primeira versao
// somava margens diferentes em cada lugar e as bordas nao fechavam na mesma
// coluna.
func (s *Stream) Moldura(titulo string, corpos []string, rodape string) []string {
	g := GlifosDaSaida()

	const teto = 78
	const margem = 2

	tituloDeco := ""
	if titulo != "" {
		tituloDeco = g.Horizontal + " " + s.style(titulo, corTitulo...) + " "
	}
	rodapeDeco := ""
	if rodape != "" {
		rodapeDeco = g.Horizontal + " " + s.style(rodape, corNota...) + " "
	}

	// Uma linha de conteudo tem de ser UMA linha. Trecho de busca traz quebra
	// de linha do proprio arquivo, e uma quebra no meio do corpo joga o resto
	// do texto para fora da moldura -- foi o que aconteceu com `search` na
	// primeira vez que ele passou por aqui. Achatar mora AQUI e nao em cada
	// chamador: quem monta um bloco nao deveria precisar saber disso.
	for i, c := range corpos {
		corpos[i] = achatar(c)
	}

	interno := larguraVisivel(tituloDeco)
	if l := larguraVisivel(rodapeDeco); l > interno {
		interno = l
	}
	for _, c := range corpos {
		if l := larguraVisivel(c) + margem; l > interno {
			interno = l
		}
	}
	if interno > teto {
		interno = teto
	}

	borda := func(esq, dir, deco string) string {
		return s.style(esq, corBorda...) + ajustarComEstilo(s, deco, interno, g.Horizontal) + s.style(dir, corBorda...)
	}

	linhas := make([]string, 0, len(corpos)+2)
	linhas = append(linhas, borda(g.CantoSupEsq, g.CantoSupDir, tituloDeco))
	for _, c := range corpos {
		linhas = append(linhas,
			s.style(g.Vertical, corBorda...)+ajustar(c, interno, " ")+s.style(g.Vertical, corBorda...))
	}
	linhas = append(linhas, borda(g.CantoInfEsq, g.CantoInfDir, rodapeDeco))
	return linhas
}

// ajustarComEstilo preenche a borda com o traco JA estilizado, para o
// enchimento ter a mesma cor da moldura.
//
// Estilizar o traco um a um produziria uma sequencia ANSI por caractere -- e
// uma borda de 78 colunas viraria 78 escapes. Estiliza o bloco inteiro de uma
// vez.
func ajustarComEstilo(s *Stream, conteudo string, largura int, traco string) string {
	l := larguraVisivel(conteudo)
	if l >= largura {
		return cortar(conteudo, largura)
	}
	return conteudo + s.style(strings.Repeat(traco, largura-l), corBorda...)
}

// Bloco imprime uma moldura pronta.
//
// E o que os comandos usam para mostrar um resumo; a lista interativa nao
// passa por aqui porque ela precisa das linhas para se redesenhar.
func (s *Stream) Bloco(titulo string, corpos []string, rodape string) {
	for _, l := range s.Moldura(titulo, corpos, rodape) {
		s.Line("%s", l)
	}
}

// achatar troca por espaco toda quebra de linha e todo controle C0, menos o
// ESC que abre uma sequencia ANSI.
//
// Preservar o ESC e o ponto: achatar sem cuidado apagaria as cores junto com as
// quebras.
func achatar(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == 0x1b:
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f:
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
