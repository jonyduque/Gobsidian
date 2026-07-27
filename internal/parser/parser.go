package parser

import (
	"github.com/yuin/goldmark"
	gtext "github.com/yuin/goldmark/text"
)

// md e construido uma vez: goldmark.Markdown e seguro para uso concorrente
// apos a construcao, e o worker pool de indexacao depende disso.
var md = goldmark.New(
	goldmark.WithExtensions(
		WikilinkExtension{},
		BlockIDExtension{},
		TagExtension{},
		InlineFieldExtension{},
	),
)

// Parse transforma os bytes de uma nota em ParsedNote. Puro: nao toca disco,
// nao guarda estado, nao conhece o caminho do arquivo.
func Parse(data []byte) (*ParsedNote, error) {
	fm, body, bodyOffset := SplitFrontmatter(data)

	note := &ParsedNote{}

	if len(fm) > 0 {
		decoded, err := DecodeFrontmatter(fm)
		if err != nil {
			// Frontmatter quebrado nao invalida a nota: o corpo continua
			// tendo headings e links uteis, e recusar tudo seria desproporcional.
			note.FrontmatterErr = err.Error()
		} else {
			note.Frontmatter = decoded
			note.Tags = append(note.Tags, tagsFromFrontmatter(decoded)...)
			note.Aliases = aliasesFromFrontmatter(decoded)
			note.Title = titleFromFrontmatter(decoded)
		}
	}

	note.Headings = ExtractHeadings(body, bodyOffset)
	if note.Title == "" {
		note.Title = firstH1(note.Headings)
	}

	// O reader recebe o CORPO, e todos os offsets do goldmark sao relativos a
	// ele. bodyOffset e somado na coleta.
	doc := md.Parser().Parse(gtext.NewReader(body))
	collect(doc, body, bodyOffset, note)

	dedupeTags(note)
	return note, nil
}

func firstH1(hs []Heading) string {
	for _, h := range hs {
		if h.Level == 1 {
			return h.Text
		}
	}
	return ""
}
