package parser

import (
	"sort"
	"strings"

	gast "github.com/yuin/goldmark/ast"
)

// collect percorre a AST uma unica vez e distribui cada no de interesse para
// o campo correspondente de ParsedNote.
func collect(doc gast.Node, body []byte, bodyOffset int64, note *ParsedNote) {
	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *WikilinkNode:
			note.Links = append(note.Links, Link{
				Raw:    node.Raw,
				Target: node.Target,
				Alias:  node.Alias,
				Anchor: node.Anchor,
				Kind:   node.LinkKind,
				Start:  bodyOffset + node.Start,
				End:    bodyOffset + node.End,
			})

		case *gast.Image:
			start, end := findMarkdownLinkSpan(node, true, body, node.Destination)
			if start != offsetUnknown {
				start += bodyOffset
				end += bodyOffset
			}
			note.Links = append(note.Links, Link{
				Raw:    string(node.Destination),
				Target: PercentDecode(string(node.Destination)),
				Alias:  inlineText(node, body),
				Kind:   LinkEmbed,
				Start:  start,
				End:    end,
			})

		case *gast.Link:
			start, end := findMarkdownLinkSpan(node, false, body, node.Destination)
			if start != offsetUnknown {
				start += bodyOffset
				end += bodyOffset
			}
			note.Links = append(note.Links, Link{
				Raw:    string(node.Destination),
				Target: PercentDecode(string(node.Destination)),
				Alias:  inlineText(node, body),
				Kind:   LinkMarkdown,
				Start:  start,
				End:    end,
			})

		case *BlockIDNode:
			note.Blocks = append(note.Blocks, Block{
				ID:    node.ID,
				Start: bodyOffset + node.Start,
				End:   bodyOffset + node.End,
			})

		case *TagNode:
			note.Tags = append(note.Tags, node.Name)

		case *InlineFieldNode:
			if note.Inline == nil {
				note.Inline = map[string][]string{}
			}
			note.Inline[node.Key] = append(note.Inline[node.Key], node.Value)
		}

		return gast.WalkContinue, nil
	})
}

// tagsFromFrontmatter le as chaves "tags" e "tag" do frontmatter. Cada uma
// pode ser uma string unica, uma string com virgulas, ou uma lista; cada
// valor entra sem o '#' inicial.
func tagsFromFrontmatter(fm map[string]any) []string {
	var out []string
	for _, key := range []string{"tags", "tag"} {
		for _, v := range stringsFromFrontmatterValue(fm[key]) {
			out = append(out, strings.TrimPrefix(v, "#"))
		}
	}
	return out
}

// aliasesFromFrontmatter le as chaves "aliases" e "alias", nas mesmas formas
// aceitas por tagsFromFrontmatter. E o insumo de RF-62.
func aliasesFromFrontmatter(fm map[string]any) []string {
	var out []string
	for _, key := range []string{"aliases", "alias"} {
		out = append(out, stringsFromFrontmatterValue(fm[key])...)
	}
	return out
}

// titleFromFrontmatter le a chave "title", apenas se for string.
func titleFromFrontmatter(fm map[string]any) string {
	if v, ok := fm["title"].(string); ok {
		return v
	}
	return ""
}

// stringsFromFrontmatterValue normaliza um valor de frontmatter YAML nas tres
// formas que o Obsidian aceita: string unica, string com virgulas, ou lista.
func stringsFromFrontmatterValue(v any) []string {
	var out []string
	switch val := v.(type) {
	case nil:
		return nil
	case string:
		for _, part := range strings.Split(val, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	case []any:
		for _, item := range val {
			s, ok := item.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
	case []string:
		for _, s := range val {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

// dedupeTags ordena e remove duplicatas de note.Tags, preservando a grafia da
// primeira ocorrencia (comparacao insensivel a maiusculas/minusculas).
func dedupeTags(note *ParsedNote) {
	if len(note.Tags) == 0 {
		return
	}

	seen := make(map[string]string, len(note.Tags))
	order := make([]string, 0, len(note.Tags))
	for _, tag := range note.Tags {
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = tag
		order = append(order, key)
	}

	out := make([]string, len(order))
	for i, key := range order {
		out[i] = seen[key]
	}
	sort.Strings(out)
	note.Tags = out
}

// inlineText concatena o texto visivel dos filhos de um no inline.
//
// Substitui gast.Node.Text, depreciada no goldmark: ela fazia exatamente esta
// travessia por dentro. O texto visivel de "[Ponto 3](Civil/PONTO 03.md)" e
// "Ponto 3", e e ele que vira o Alias do link.
//
// A recursao existe porque o rotulo pode ter formatacao: em "[**Ponto** 3](x)"
// o "Ponto" e filho de um no de enfase, e parar no primeiro nivel perderia a
// metade em negrito do rotulo.
func inlineText(n gast.Node, src []byte) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch t := c.(type) {
		case *gast.Text:
			b.Write(t.Segment.Value(src))
		case *gast.String:
			b.Write(t.Value)
		default:
			b.WriteString(inlineText(c, src))
		}
	}
	return b.String()
}

// PercentDecode desfaz escapes %XX de um destino de link Markdown.
//
// "%20" e o que todo editor gera para caminho com espaco, e em cofre em
// portugues nome com espaco e a regra, nao a excecao. Sem decodificar,
// "[Ponto 3](Civil/PONTO%2003.md)" nunca resolve. Confirmado contra o
// metadata cache real do Obsidian, que registra o destino ja decodificado.
//
// Escrito a mao porque net/url esta fora de alcance: nenhum pacote sob
// internal/ ou cmd/ importa net nem net/*, e a ausencia de rede e verificada
// no CI. Importar net/url por causa de uma funcao de string derrubaria a
// garantia inteira por conveniencia.
//
// Sequencia invalida devolve o byte original. "50% de desconto" nao e escape,
// e preservar o que o usuario escreveu e melhor que inventar um byte ou
// recusar o link.
//
// A decodificacao e por BYTE, nao por rune, e isso e proposital: UTF-8
// multibyte chega como varios %XX seguidos, e remontar byte a byte reconstroi
// o caractere original.
//
// Exportada porque mcpsrv tambem precisa dela, para desfazer o escape das URIs
// de resource. Uma segunda copia divergiria da primeira no dia em que alguem
// corrigisse so uma — e as regras de escape invalido sao exatamente o tipo de
// sutileza que uma copia perde.
func PercentDecode(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); i++ {
		if s[i] != '%' || i+2 >= len(s) {
			b.WriteByte(s[i])
			continue
		}

		hi, okHi := unhexDigit(s[i+1])
		lo, okLo := unhexDigit(s[i+2])
		if !okHi || !okLo {
			b.WriteByte(s[i])
			continue
		}

		b.WriteByte(hi<<4 | lo)
		i += 2
	}

	return b.String()
}

func unhexDigit(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// findMarkdownLinkSpan calcula os offsets int64 (start e end) do span bruto
// de um link ou embed Markdown na slice body (ex: "[texto](destino)" ou
// "![alt](imagem.png)"). Se nao conseguir determinar de forma confiavel,
// devolve (offsetUnknown, offsetUnknown).
func findMarkdownLinkSpan(n gast.Node, isEmbed bool, body, destino []byte) (int64, int64) {
	minStart := -1
	maxEnd := -1

	var findOffsets func(parent gast.Node)
	findOffsets = func(parent gast.Node) {
		for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
			switch t := c.(type) {
			case *gast.Text:
				s := t.Segment.Start
				e := t.Segment.Stop
				if minStart == -1 || s < minStart {
					minStart = s
				}
				if e > maxEnd {
					maxEnd = e
				}
			case *WikilinkNode:
				s := int(t.Start)
				e := int(t.End)
				if minStart == -1 || s < minStart {
					minStart = s
				}
				if e > maxEnd {
					maxEnd = e
				}
			default:
				findOffsets(c)
			}
		}
	}

	findOffsets(n)

	startIdx := -1

	if minStart != -1 {
		prefixLen := 1 // '['
		if isEmbed {
			prefixLen = 2 // '!['
		}

		cand := minStart - prefixLen
		if cand >= 0 {
			if isEmbed {
				if cand+1 < len(body) && body[cand] == '!' && body[cand+1] == '[' {
					startIdx = cand
				}
			} else {
				if body[cand] == '[' {
					startIdx = cand
				}
			}
		}
	}

	if startIdx == -1 {
		// Link SEM filhos de texto — "[](alvo.md)". Nao ha segmento nenhum de
		// onde partir, entao a ancora e o DESTINO, e nao o prefixo.
		//
		// Ate 2026-08-28 isto era `strings.Index(body, "[](")`: a PRIMEIRA
		// ocorrencia do corpo inteiro (achado M10). Sondado, num corpo com dois
		// links de texto vazio, os DOIS recebiam o span do primeiro —
		// "[](um.md)" para o link de dois.md. Nao e offset ausente: e offset
		// plausivel e errado, que alimenta reescrita de link e note_move.
		//
		// A busca por "](destino)" e especifica o bastante para distinguir os
		// dois. Quando o MESMO destino aparece duas vezes com texto vazio ela
		// volta a ser ambigua, e ai a resposta e offsetUnknown, que e honesta.
		alvo := "](" + string(destino) + ")"
		idx := strings.Index(string(body), alvo)
		if idx != -1 && strings.Count(string(body), alvo) == 1 {
			abre := idx
			if isEmbed {
				// "![](x)": o '[' esta em idx, o '!' antes dele.
				if abre >= 1 && body[abre-1] == '[' && abre >= 2 && body[abre-2] == '!' {
					startIdx = abre - 2
					maxEnd = idx
				}
			} else if abre >= 0 && body[abre] == ']' && abre >= 1 && body[abre-1] == '[' {
				startIdx = abre - 1
				maxEnd = idx
			}
		}
	}

	// Imagem aninhada dentro de link: "[![alt](img.png)](dest.md)".
	//
	// A recursao desce na imagem e acha o texto "alt", entao minStart aponta
	// para dentro dela e o '[' um byte antes e o da IMAGEM — nao o do link
	// externo. Sondado em 2026-08-28: o link para dest.md recebia o span
	// "[alt](img.png)", truncado E sobreposto ao do embed (achado M10).
	//
	// Um '[' precedido de "![" so pode ser o da imagem, e o link externo abre
	// dois bytes antes. O laco cobre aninhamento repetido.
	for !isEmbed && startIdx >= 2 && body[startIdx-1] == '!' && body[startIdx-2] == '[' {
		startIdx -= 2
	}

	if startIdx == -1 {
		return offsetUnknown, offsetUnknown
	}

	// O ']' do rotulo e o que CASA com o '[' de startIdx, e nao o primeiro
	// depois do texto.
	//
	// Ate 2026-08-28 a busca comecava em maxEnd — o fim do ultimo segmento de
	// texto encontrado — e parava no primeiro ']'. Em "[![alt](img.png)](dest.md)"
	// esse primeiro ']' e o da IMAGEM, e o span saia truncado: "[![alt](img.png)"
	// em vez do link inteiro (achado M10). Contar profundidade resolve o
	// aninhamento em qualquer nivel.
	closeBracket := -1
	profundidade := 0
	for p := startIdx; p < len(body); p++ {
		switch body[p] {
		case '\\':
			// Escape consome o proximo byte: um colchete escapado nao fecha
			// nem abre rotulo.
			p++
		case '[':
			profundidade++
		case ']':
			profundidade--
			if profundidade == 0 {
				closeBracket = p
			}
		}
		if closeBracket != -1 {
			break
		}
	}
	if closeBracket == -1 || closeBracket+1 >= len(body) || body[closeBracket+1] != '(' {
		return offsetUnknown, offsetUnknown
	}

	openParen := closeBracket + 1
	depth := 1
	p := openParen + 1
	for p < len(body) && depth > 0 {
		ch := body[p]
		if ch == '\\' {
			p += 2
			continue
		}
		if ch == '"' || ch == '\'' {
			quote := ch
			p++
			for p < len(body) && body[p] != quote {
				if body[p] == '\\' {
					p += 2
				} else {
					p++
				}
			}
			if p < len(body) {
				p++
			}
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		}
		p++
	}

	if depth != 0 {
		return offsetUnknown, offsetUnknown
	}

	endIdx := p
	return int64(startIdx), int64(endIdx)
}
