package writer

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jonyd/gobsidian/internal/parser"
)

var (
	// ErrInvalidLinkOffset e retornado quando um link possui offsets invalidos (-1, fora dos limites ou Start >= End).
	ErrInvalidLinkOffset = errors.New("invalid link offset (-1 or out of bounds)")
	// ErrOverlappingLinks e retornado quando ha sobreposicao de intervalos entre substituicoes.
	ErrOverlappingLinks = errors.New("overlapping link replacement offsets")
)

// LinkReplacement define um link a ser substituido em uma nota e o seu novo alvo.
type LinkReplacement struct {
	Link      parser.Link
	NewTarget string
}

// RewriteLinks substitui os links especificadas em src pelos seus novos alvos.
// As substituicoes sao ordenadas de tras para a frente (Start decrescente)
// para preservar intactos os offsets anteriores da mesma nota.
func RewriteLinks(src []byte, replacements []LinkReplacement) ([]byte, error) {
	if len(replacements) == 0 {
		return src, nil
	}

	work := make([]LinkReplacement, len(replacements))
	copy(work, replacements)

	for _, r := range work {
		if r.Link.Start < 0 || r.Link.End < 0 || r.Link.Start > int64(len(src)) || r.Link.End > int64(len(src)) || r.Link.Start >= r.Link.End {
			return nil, fmt.Errorf("%w: link %q at [%d:%d] in buffer of len %d", ErrInvalidLinkOffset, r.Link.Raw, r.Link.Start, r.Link.End, len(src))
		}
	}

	sort.Slice(work, func(i, j int) bool {
		return work[i].Link.Start > work[j].Link.Start
	})

	for i := 0; i < len(work)-1; i++ {
		if work[i+1].Link.End > work[i].Link.Start {
			return nil, fmt.Errorf("%w: [%d:%d] and [%d:%d]", ErrOverlappingLinks, work[i+1].Link.Start, work[i+1].Link.End, work[i].Link.Start, work[i].Link.End)
		}
	}

	out := make([]byte, len(src))
	copy(out, src)

	for _, r := range work {
		newText := BuildLinkText(r.Link, r.NewTarget)
		s := int(r.Link.Start)
		e := int(r.Link.End)

		prefix := out[:s]
		suffix := out[e:]
		res := make([]byte, 0, len(prefix)+len(newText)+len(suffix))
		res = append(res, prefix...)
		res = append(res, []byte(newText)...)
		res = append(res, suffix...)
		out = res
	}

	return out, nil
}

// BuildLinkText constroi a string do novo link preservando o tipo (Wiki, Embed, Markdown),
// alias e ancora do link original.
func BuildLinkText(orig parser.Link, newTarget string) string {
	switch orig.Kind {
	case parser.LinkWiki:
		return formatWikiLink("", newTarget, orig.Anchor, orig.Alias)
	case parser.LinkEmbed:
		if strings.HasPrefix(orig.Raw, "!") && strings.HasPrefix(strings.TrimPrefix(orig.Raw, "!"), "[[") {
			return formatWikiLink("!", newTarget, orig.Anchor, orig.Alias)
		}
		targetEnc := encodeMarkdownTarget(newTarget, orig.Raw)
		return fmt.Sprintf("![%s](%s)", orig.Alias, targetEnc)
	case parser.LinkMarkdown:
		targetEnc := encodeMarkdownTarget(newTarget, orig.Raw)
		return fmt.Sprintf("[%s](%s)", orig.Alias, targetEnc)
	default:
		return formatWikiLink("", newTarget, orig.Anchor, orig.Alias)
	}
}

func formatWikiLink(prefix, target, anchor, alias string) string {
	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString("[[")
	sb.WriteString(target)
	if anchor != "" {
		sb.WriteByte('#')
		sb.WriteString(anchor)
	}
	if alias != "" {
		sb.WriteByte('|')
		sb.WriteString(alias)
	}
	sb.WriteString("]]")
	return sb.String()
}

func encodeMarkdownTarget(target string, origRaw string) string {
	if strings.Contains(origRaw, "%20") || strings.Contains(target, " ") {
		return strings.ReplaceAll(target, " ", "%20")
	}
	return target
}
