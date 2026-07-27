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
				Raw:    string(node.Destination),
				Target: string(node.Destination),
				Alias:  string(node.Text(body)),
				Kind:   LinkEmbed,
				Start:  offsetUnknown,
				End:    offsetUnknown,
			})

		case *gast.Link:
			// Link Markdown padrao. So interessa quando aponta para dentro do
			// cofre; a decisao de o que e interno cabe ao indice, entao aqui
			// registramos tudo e deixamos a resolucao filtrar.
			note.Links = append(note.Links, Link{
				Raw:    string(node.Destination),
				Target: string(node.Destination),
				Alias:  string(node.Text(body)),
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
