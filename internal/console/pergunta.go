package console

import "fmt"

// Pergunta escreve o prompt de uma pergunta, SEM quebrar a linha: o cursor
// fica onde o usuario vai digitar.
//
// # Por que nao Detail
//
// Ate 2026-09-09 toda pergunta saia por Detail, que e a linha de detalhe do
// relatorio do `doctor` -- cinza, indentada, e com quebra de linha no fim. O
// resultado era que a pergunta parecia parte do relatorio anterior e o cursor
// piscava na linha de baixo, longe dela. Uma pergunta e o unico momento em que
// o programa PARA e espera; ela precisa se distinguir de tudo o que rolou
// antes.
//
// O marcador tem cor E glifo proprios, pela mesma razao dos marcadores de
// estado: num terminal sem cor a pergunta continua se distinguindo.
//
// O padrao aparece entre parenteses, apagado, porque ele e a resposta de quem
// so aperta Enter -- e quem so aperta Enter merece saber no que esta
// concordando.
func (s *Stream) Pergunta(pergunta, padrao string) {
	g := GlifosDaSaida()
	linha := s.style(g.Pergunta, corPergunta...) + " " + s.style(pergunta, corTitulo...)
	if padrao != "" {
		linha += " " + s.style("("+padrao+")", corNota...)
	}
	linha += " " + s.style(g.Cursor, corDestaque...) + " "
	_, _ = fmt.Fprint(s.w, linha)
}

// Resposta ecoa o que foi escolhido, fechando a pergunta.
//
// Existe porque o modo nao-interativo (--yes) e o padrao-por-Enter deixam a
// tela sem nenhum registro do que foi decidido: a pergunta some no scroll e o
// usuario nao consegue conferir depois. Uma linha por decisao, apagada, e o
// que torna a instalacao auditavel sem virar log.
func (s *Stream) Resposta(escolha string) {
	g := GlifosDaSaida()
	s.Line("%s %s", s.style(g.Resposta, corMarcada...), s.style(escolha, corNota...))
}

// Titulo abre uma secao da interface interativa.
//
// Diferente de Step: Step marca ETAPA em andamento num relatorio ([...]), e
// Titulo separa BLOCOS de uma conversa com o usuario. Os dois na mesma tela
// confundiam qual estava em curso.
func (s *Stream) Titulo(format string, a ...any) {
	g := GlifosDaSaida()
	s.Line("")
	s.Line("%s %s", s.style(g.Secao, corDestaque...), s.style(fmt.Sprintf(format, a...), corTitulo...))
}

// Passo anuncia uma etapa em andamento dentro de uma sequencia.
//
// Distinto de Step: Step abre uma etapa de RELATORIO ([...]), e Passo marca
// progresso numa sequencia que o usuario esta esperando terminar. Sai
// indentado e apagado, porque o que importa nele e o movimento, nao o conteudo
// -- quem le quer saber que algo esta acontecendo.
func (s *Stream) Passo(format string, a ...any) {
	g := GlifosDaSaida()
	s.Line("  %s %s", s.style(g.Cursor, corDestaque...), s.Dim(fmt.Sprintf(format, a...)))
}
