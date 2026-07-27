package parser

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Slug normaliza um heading para comparacao com a ancora de um wikilink:
// minusculas, sem acentos, sem pontuacao, espacos colapsados.
//
// O Obsidian casa ancora de forma mais permissiva do que igualdade textual,
// e reproduzir isso e o que faz [[nota#Capitulo 118]] encontrar
// "## Capítulo 118" escrito com acento.
func Slug(text string) string {
	stripped, _, err := transform.String(
		transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC),
		text,
	)
	if err != nil {
		stripped = text
	}

	var b strings.Builder
	b.Grow(len(stripped))

	lastSpace := true
	for _, r := range strings.ToLower(stripped) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastSpace = false
		case unicode.IsSpace(r):
			if !lastSpace && b.Len() > 0 {
				b.WriteByte(' ')
				lastSpace = true
			}
		default:
			// Pontuacao vira nada, nao vira espaco: "Art. 1.234" precisa
			// virar "art 1234", nao "art 1 234".
		}
	}

	return strings.TrimSpace(b.String())
}
