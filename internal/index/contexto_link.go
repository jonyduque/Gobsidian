package index

import (
	"strings"
	"unicode/utf8"

	"github.com/jonyd/gobsidian/internal/parser"
)

// contextoBytes e quanto de texto entra de cada lado da referencia.
//
// 80 de cada lado da ~160 caracteres — o bastante para o modelo decidir se o
// backlink interessa sem abrir a origem, e curto o bastante para nao inflar o
// cache de metadados de um cofre com muitos links. Nao e numero medido: e uma
// escolha de produto, e esta escrita aqui para ser discutida em um lugar so.
const contextoBytes = 80

// montarLinks constroi os ResolvedLink de uma nota, com o contexto de cada
// referencia recortado do corpo.
//
// E a UNICA funcao que monta essa lista. Ha dois caminhos de construcao de Note
// — o worker do Build e construirNota, no caminho do watcher — e ate 2026-08-26
// os dois faziam `ResolvedLink{Link: l}` a mao, em `index.go:133` e
// `update.go:142`. Duas contas do mesmo fato concordavam por coincidencia; e o
// padrao que produziu o bug [[STJ]].
//
// O corpo e obrigatorio aqui porque o contexto nao pode ser recalculado depois:
// o cache de metadados existe justamente para nao reler o disco no boot, e o
// codec nao persiste Resolved/Via/State (recalculados na carga). O contexto,
// diferente deles, NAO e derivavel do que esta no indice — por isso ele e
// persistido, e por isso esta tarefa exigiu bump de formato.
func montarLinks(body []byte, links []parser.Link) []ResolvedLink {
	if len(links) == 0 {
		return nil
	}
	out := make([]ResolvedLink, 0, len(links))
	for _, l := range links {
		out = append(out, ResolvedLink{
			Link:    l,
			Context: contextoDoLink(body, l),
		})
	}
	return out
}

// contextoDoLink recorta o texto ao redor de uma referencia.
//
// Devolve "" quando os offsets nao servem — link sem span conhecido
// (offsetUnknown), ou offsets que nao cabem no corpo recebido. Um contexto
// vazio e honesto; um contexto recortado de posicao errada seria pior que
// nenhum, porque pareceria informacao.
//
// Sondado em 2026-08-26: os TRES tipos de link carregam offsets reais —
// wikilink, embed e markdown. O comentario de parser/types.go que dizia que
// link Markdown fica em offsetUnknown estava velho (achado B14 da auditoria) e
// foi corrigido junto.
func contextoDoLink(body []byte, l parser.Link) string {
	if l.Start < 0 || l.End <= l.Start {
		return ""
	}
	if int(l.End) > len(body) {
		return ""
	}

	ini := int(l.Start) - contextoBytes
	if ini < 0 {
		ini = 0
	}
	fim := int(l.End) + contextoBytes
	if fim > len(body) {
		fim = len(body)
	}

	// Recorte por byte pode cair no meio de uma runa; alinhar evita gravar
	// U+FFFD no cache e devolve-lo ao host como se fosse o texto da nota.
	for ini < len(body) && !utf8.RuneStart(body[ini]) {
		ini++
	}
	for fim > ini && fim < len(body) && !utf8.RuneStart(body[fim]) {
		fim--
	}

	trecho := strings.TrimSpace(string(body[ini:fim]))
	// Uma referencia costuma estar no meio de um paragrafo; quebras de linha
	// no meio do contexto atrapalham quem le a resposta.
	return strings.Join(strings.Fields(trecho), " ")
}
