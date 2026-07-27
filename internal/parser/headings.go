package parser

import (
	"bytes"
	"strings"
)

// ExtractHeadings percorre o corpo linha a linha e devolve a hierarquia com
// os offsets de secao ja calculados.
//
// A varredura e propria em vez de vir da AST do goldmark por um motivo:
// precisamos do offset de FIM de secao, que e uma propriedade da hierarquia,
// nao do no. O goldmark da a posicao de cada heading; o fim de uma secao e o
// inicio do proximo heading de nivel menor ou igual, e calcular isso exige
// uma passada com pilha de qualquer forma.
func ExtractHeadings(body []byte, bodyOffset int64) []Heading {
	var out []Heading

	inFence := false
	var fenceMarker string

	pos := int64(0)
	for len(body) > 0 {
		nl := bytes.IndexByte(body, '\n')
		var line []byte
		var advance int64
		if nl < 0 {
			line, advance = body, int64(len(body))
		} else {
			line, advance = body[:nl], int64(nl)+1
		}

		trimmed := bytes.TrimRight(line, "\r")
		text := string(trimmed)

		// Blocos de codigo cercados. Um '#' dentro de um deles nao e heading,
		// e ignorar isso e a forma mais rapida de produzir hierarquia falsa.
		if marker := fenceMarkerOf(text); marker != "" {
			if !inFence {
				inFence, fenceMarker = true, marker
			} else if strings.HasPrefix(strings.TrimSpace(text), fenceMarker) {
				inFence, fenceMarker = false, ""
			}
			pos += advance
			body = body[advance:]
			continue
		}

		if !inFence {
			if level, title, ok := parseATXHeading(text); ok {
				out = append(out, Heading{
					Level:     level,
					Text:      title,
					Slug:      Slug(title),
					Start:     bodyOffset + pos,
					BodyStart: bodyOffset + pos + advance,
				})
			}
		}

		pos += advance
		body = body[advance:]
	}

	total := bodyOffset + pos
	closeSections(out, total)
	return out
}

// closeSections preenche End: o inicio do proximo heading de nivel menor ou
// igual, ou o fim do arquivo.
func closeSections(hs []Heading, total int64) {
	for i := range hs {
		hs[i].End = total
		for j := i + 1; j < len(hs); j++ {
			if hs[j].Level <= hs[i].Level {
				hs[i].End = hs[j].Start
				break
			}
		}
	}
}

func parseATXHeading(line string) (int, string, bool) {
	// Ate tres espacos de indentacao sao permitidos pelo CommonMark.
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 {
		return 0, "", false
	}

	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return 0, "", false
	}
	if level < len(trimmed) && trimmed[level] != ' ' && trimmed[level] != '\t' {
		return 0, "", false
	}

	title := strings.TrimSpace(trimmed[level:])
	// Fechamento opcional: "## Titulo ##".
	title = strings.TrimRight(title, "#")
	return level, strings.TrimSpace(title), true
}

func fenceMarkerOf(line string) string {
	t := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(t, "```"):
		return "```"
	case strings.HasPrefix(t, "~~~"):
		return "~~~"
	}
	return ""
}
