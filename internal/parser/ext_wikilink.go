package parser

import (
	"bytes"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var KindWikilink = gast.NewNodeKind("Wikilink")

// WikilinkNode e o no de AST para "[[alvo]]" e "![[alvo]]". LinkKind guarda
// wiki vs. embed; o nome nao pode ser "Kind" porque gast.Node ja exige um
// metodo Kind() gast.NodeKind, e Go nao permite campo e metodo com o mesmo
// nome no mesmo tipo.
type WikilinkNode struct {
	gast.BaseInline

	Target   string
	Alias    string
	Anchor   string
	LinkKind LinkKind
	Raw      string
	Start    int64
	End      int64
}

func (n *WikilinkNode) Kind() gast.NodeKind { return KindWikilink }

func (n *WikilinkNode) Dump(src []byte, level int) {
	gast.DumpHelper(n, src, level, map[string]string{
		"Target": n.Target,
		"Alias":  n.Alias,
		"Anchor": n.Anchor,
		"Raw":    n.Raw,
	}, nil)
}

type wikilinkParser struct{}

// Trigger diz ao goldmark em quais bytes oferecer este parser. '[' cobre
// [[...]]; '!' cobre ![[...]].
func (p *wikilinkParser) Trigger() []byte { return []byte{'[', '!'} }

func (p *wikilinkParser) Parse(_ gast.Node, block text.Reader, _ gparser.Context) gast.Node {
	line, seg := block.PeekLine()

	embed := false
	offset := 0
	if len(line) > 0 && line[0] == '!' {
		embed = true
		offset = 1
	}
	if len(line) < offset+4 || line[offset] != '[' || line[offset+1] != '[' {
		return nil
	}

	closeIdx := bytes.Index(line[offset+2:], []byte("]]"))
	if closeIdx < 0 {
		return nil
	}
	inner := line[offset+2 : offset+2+closeIdx]

	// Um wikilink nao atravessa linha, e nao contem "[[" aninhado.
	if bytes.ContainsAny(inner, "\n") || bytes.Contains(inner, []byte("[[")) {
		return nil
	}

	total := offset + 2 + closeIdx + 2
	raw := string(line[:total])

	target, anchor, alias := splitWikilink(string(inner))
	if target == "" && anchor == "" {
		return nil
	}

	node := &WikilinkNode{
		Target: target,
		Alias:  alias,
		Anchor: anchor,
		Raw:    raw,
		Start:  int64(seg.Start),
		End:    int64(seg.Start + total),
	}
	node.LinkKind = LinkWiki
	if embed {
		node.LinkKind = LinkEmbed
	}

	block.Advance(total)
	return node
}

// splitWikilink reparte "alvo#ancora|alias" nas tres partes. A ordem importa:
// o alias e sempre o ultimo, e a ancora vem antes dele.
func splitWikilink(inner string) (target, anchor, alias string) {
	if i := bytes.IndexByte([]byte(inner), '|'); i >= 0 {
		alias = inner[i+1:]
		inner = inner[:i]
	}
	if i := bytes.IndexByte([]byte(inner), '#'); i >= 0 {
		anchor = inner[i+1:]
		inner = inner[:i]
	}
	return trimSpace(inner), trimSpace(anchor), trimSpace(alias)
}

func trimSpace(s string) string { return string(bytes.TrimSpace([]byte(s))) }

// WikilinkExtension registra o inline parser no goldmark.
type WikilinkExtension struct{}

func (WikilinkExtension) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		gparser.WithInlineParsers(
			// Prioridade abaixo da do link padrao do CommonMark (200), para
			// que "[[x]]" seja oferecido a nos antes de virar dois links
			// aninhados.
			util.Prioritized(&wikilinkParser{}, 150),
		),
	)
}
