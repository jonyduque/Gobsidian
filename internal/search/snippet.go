package search

import (
	"context"
	"unicode/utf8"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Constantes de recorte de trecho (RF-22, TOOLS.md schema).
const (
	DefaultSnippetChars = 240
	MaxSnippetChars     = 1000
)

// Snippet representa um trecho recortado e destacado de uma nota.
type Snippet struct {
	Text           string
	HighlightStart int
	HighlightEnd   int
	MatchedHeading string
}

// GenerateSnippet recorta do disco o trecho em volta das ocorrências dos termos de busca.
func GenerateSnippet(ctx context.Context, v *vault.Vault, ix *Inverted, idx *index.Index, path string, queryTerms []string, maxChars int) (Snippet, error) {
	if maxChars <= 0 {
		maxChars = DefaultSnippetChars
	}
	if maxChars > MaxSnippetChars {
		maxChars = MaxSnippetChars
	}

	cPath := vault.CanonicalPath(path)

	// 1. Arquivo somente-nuvem NUNCA é aberto para evitar downloads síncronos.
	var noteHadBOM bool
	var headings []parser.Heading
	if idx != nil {
		note, ok := idx.Get(cPath)
		if ok {
			if note.CloudOnly {
				return Snippet{}, nil
			}
			noteHadBOM = note.BOM
			headings = note.Headings
		}
	}

	if ix == nil || len(queryTerms) == 0 {
		return Snippet{}, nil
	}

	// 2. Coleta posições do primeiro termo correspondente no índice invertido
	type matchPos struct {
		term  string
		start int64
		end   int64
	}

	var bestMatch *matchPos

	for _, termStr := range queryTerms {
		toks := Analyze(termStr)
		for _, tok := range toks {
			termsToSearch := []string{tok.Raw}
			if tok.Reduced != "" && tok.Reduced != tok.Raw {
				termsToSearch = append(termsToSearch, tok.Reduced)
			}

			for _, t := range termsToSearch {
				postings := ix.Postings(t)
				for _, p := range postings {
					if p.Path == string(cPath) && len(p.Positions) > 0 {
						pos := p.Positions[0]
						bestMatch = &matchPos{
							term:  t,
							start: pos.Start,
							end:   pos.End,
						}
						break
					}
				}
				if bestMatch != nil {
					break
				}
			}
			if bestMatch != nil {
				break
			}
		}
		if bestMatch != nil {
			break
		}
	}

	if bestMatch == nil {
		return Snippet{}, nil
	}

	// 3. Offset do BOM:
	// Posições no Inverted são relativas ao corpo sem BOM.
	// Se a nota no disco tem BOM (3 bytes \xEF\xBB\xBF), os offsets no disco são deslocados em +3 bytes.
	var bomOffset int64
	if noteHadBOM {
		bomOffset = int64(vault.BOMLen)
	}

	diskMatchStart := bestMatch.start + bomOffset
	diskMatchEnd := bestMatch.end + bomOffset

	var matchedHeadingText string
	for _, h := range headings {
		if diskMatchStart >= h.Start && diskMatchEnd <= h.End {
			matchedHeadingText = h.Text
			break
		}
	}

	// 4. Calcula a janela de recorte ao redor da ocorrência
	half := int64(maxChars / 2)
	winStart := diskMatchStart - half
	if winStart < 0 {
		winStart = 0
	}
	winEnd := winStart + int64(maxChars)

	buf, err := v.ReadRange(ctx, cPath, winStart, winEnd)
	if err != nil {
		data, readErr := v.ReadAll(ctx, cPath)
		if readErr != nil {
			return Snippet{}, nil
		}
		buf = data
		winStart = 0
	}

	// Trima sequências UTF-8 parciais nas extremidades e calcula destaques relativos ao trecho
	snippetText, relStart, relEnd := adjustUTF8Highlight(buf, winStart, diskMatchStart, diskMatchEnd)

	return Snippet{
		Text:           snippetText,
		HighlightStart: relStart,
		HighlightEnd:   relEnd,
		MatchedHeading: matchedHeadingText,
	}, nil
}

func adjustUTF8Highlight(buf []byte, winStart, matchStart, matchEnd int64) (string, int, int) {
	if len(buf) == 0 {
		return "", 0, 0
	}

	cropStart := 0
	for cropStart < len(buf) && !utf8.RuneStart(buf[cropStart]) {
		cropStart++
	}

	cropEnd := len(buf)
	for cropEnd > cropStart && !utf8.Valid(buf[cropStart:cropEnd]) {
		cropEnd--
	}

	slice := buf[cropStart:cropEnd]
	text := string(slice)

	relStart := int(matchStart - (winStart + int64(cropStart)))
	relEnd := int(matchEnd - (winStart + int64(cropStart)))

	if relStart < 0 {
		relStart = 0
	}
	if relEnd > len(text) {
		relEnd = len(text)
	}
	if relStart > relEnd {
		relStart = relEnd
	}

	return text, relStart, relEnd
}
