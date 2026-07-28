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
			// A grafia Markdown de um embed. Sem este caso,
			// "![alt](diagrama.png)" fica invisivel para o grafo enquanto
			// "![[diagrama.png]]" e visto — a mesma nota perde metade dos
			// anexos dependendo de como foram escritos.
			note.Links = append(note.Links, Link{
				// Raw fica byte-exato: note_move reescreve a partir dele, e
				// normalizar aqui corromperia o texto do usuario. So o alvo
				// usado para resolver e decodificado.
				Raw:    string(node.Destination),
				Target: PercentDecode(string(node.Destination)),
				Alias:  inlineText(node, body),
				Kind:   LinkEmbed,
				Start:  offsetUnknown,
				End:    offsetUnknown,
			})

		case *gast.Link:
			// Link Markdown padrao. So interessa quando aponta para dentro do
			// cofre; a decisao de o que e interno cabe ao indice, entao aqui
			// registramos tudo e deixamos a resolucao filtrar.
			note.Links = append(note.Links, Link{
				// Ver o comentario no caso *gast.Image: Raw byte-exato,
				// alvo decodificado.
				Raw:    string(node.Destination),
				Target: PercentDecode(string(node.Destination)),
				Alias:  inlineText(node, body),
				Kind:   LinkMarkdown,
				Start:  offsetUnknown,
				End:    offsetUnknown,
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
