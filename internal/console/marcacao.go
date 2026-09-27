package console

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// marcar desenha a marcacao que os textos do produto carregam: **negrito**,
// *italico* e `codigo`.
//
// # Por que a marcacao mora no texto
//
// O dono revisa a redacao em docs/TEXTOS.md, e o que ele escreve la em negrito
// ou em italico precisa sair assim na tela. Guardar a enfase fora do texto --
// uma lista de palavras a realcar, por exemplo -- seria uma segunda conta da
// mesma decisao, e a menos consultada divergiria. A constante leva a marcacao,
// e o console e o unico que a desenha.
//
// # Com e sem cor
//
// Com cor, cada trecho marcado vira uma sequencia SGR e o resto do texto fica
// como veio. Sem cor -- saida redirecionada, NO_COLOR, TERM=dumb --, negrito e
// italico somem e sobram as palavras, mas as crases FICAM: elas sao o que ainda
// separa um comando do texto em volta, a mesma regra de o marcador de estado
// continuar legivel sem cor.
//
// # Estilo de fora
//
// externo sao os codigos que envolvem o texto inteiro -- o apagado de uma linha
// de Detail, o realce de um titulo. Um trecho marcado termina com reset, e o
// reset desliga TAMBEM o estilo de fora; sem religa-lo, o resto da linha sairia
// sem o apagado. Por isso cada trecho termina com reset seguido do estilo de
// fora de novo.
//
// # Quando um asterisco NAO e marcacao
//
// Italico abre num asterisco no comeco do texto ou depois de espaco ou
// pontuacao de abertura, seguido de algo que nao e espaco; fecha num asterisco
// precedido de algo que nao e espaco e seguido de fim, espaco ou pontuacao.
// "* para todos" e "3 * 4" nao abrem nada, e um asterisco sem par sai como
// esta. Dentro de crases nada e interpretado: "`*`" e um asterisco literal.
func (s *Stream) marcar(t string, externo ...string) string {
	if !strings.ContainsAny(t, "*`") {
		return t
	}

	var b strings.Builder
	b.Grow(len(t) + 16)

	trecho := func(conteudo string, codes ...string) {
		if !s.color {
			b.WriteString(conteudo)
			return
		}
		b.WriteString(sgr(append(append([]string{}, externo...), codes...)...))
		b.WriteString(conteudo)
		b.WriteString(reset)
		if len(externo) > 0 {
			b.WriteString(sgr(externo...))
		}
	}

	for i := 0; i < len(t); {
		switch {
		case t[i] == '`':
			fim := strings.IndexByte(t[i+1:], '`')
			if fim < 0 {
				b.WriteString(t[i:])
				return b.String()
			}
			conteudo := t[i+1 : i+1+fim]
			if s.color {
				trecho(conteudo, corCodigo...)
			} else {
				b.WriteString("`" + conteudo + "`")
			}
			i += fim + 2

		case strings.HasPrefix(t[i:], "**") && abreEnfase(t, i, 2):
			fim := fechaEnfase(t, i+2, "**")
			if fim < 0 {
				b.WriteString("**")
				i += 2
				continue
			}
			trecho(t[i+2:fim], codeBold)
			i = fim + 2

		case t[i] == '*' && abreEnfase(t, i, 1):
			fim := fechaEnfase(t, i+1, "*")
			if fim < 0 {
				b.WriteByte('*')
				i++
				continue
			}
			trecho(t[i+1:fim], codeItalic)
			i = fim + 1

		default:
			b.WriteByte(t[i])
			i++
		}
	}
	return b.String()
}

// abreEnfase diz se o delimitador de largura n em i abre um trecho: comeco do
// texto ou depois de espaco ou pontuacao de abertura, e seguido de algo que
// nao e espaco nem outro asterisco.
func abreEnfase(t string, i, n int) bool {
	if i+n >= len(t) {
		return false
	}
	depois, _ := utf8.DecodeRuneInString(t[i+n:])
	if unicode.IsSpace(depois) || depois == '*' {
		return false
	}
	if i == 0 {
		return true
	}
	antes, _ := utf8.DecodeLastRuneInString(t[:i])
	return unicode.IsSpace(antes) || strings.ContainsRune("([{\"'‹“", antes)
}

// fechaEnfase procura, a partir de de, o delimitador que fecha o trecho:
// precedido de algo que nao e espaco e seguido de fim, espaco ou pontuacao.
// Devolve -1 quando nao ha.
func fechaEnfase(t string, de int, delim string) int {
	for j := de; j < len(t); j++ {
		if t[j] == '`' {
			// Crase dentro de um trecho e texto: o fechamento nao e procurado
			// dentro dela.
			if k := strings.IndexByte(t[j+1:], '`'); k >= 0 {
				j += k + 1
			}
			continue
		}
		if !strings.HasPrefix(t[j:], delim) || j == de {
			continue
		}
		if delim == "*" && strings.HasPrefix(t[j:], "**") {
			j++
			continue
		}
		antes, _ := utf8.DecodeLastRuneInString(t[:j])
		if unicode.IsSpace(antes) {
			continue
		}
		fim := j + len(delim)
		if fim == len(t) {
			return j
		}
		depois, _ := utf8.DecodeRuneInString(t[fim:])
		if unicode.IsSpace(depois) || unicode.IsPunct(depois) {
			return j
		}
	}
	return -1
}

// SemMarcacao devolve o texto sem negrito nem italico, com as crases mantidas:
// o que um Stream sem cor escreveria.
//
// Existe para o texto que o console NAO imprime -- a descricao que o shell
// mostra ao lado de um comando no autocompletar sai pelo cobra direto, e la
// "**gobsidian**" apareceria com os asteriscos.
func SemMarcacao(t string) string {
	return (&Stream{}).marcar(t)
}
