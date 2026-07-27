package parser

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

var fmDelim = []byte("---")

// SplitFrontmatter separa o bloco YAML do corpo, devolvendo tambem o offset
// em que o corpo comeca no arquivo original. O offset e o que mantem todos os
// offsets de heading e de bloco corretos em relacao ao arquivo, nao ao corpo.
func SplitFrontmatter(data []byte) ([]byte, []byte, int64) {
	if !bytes.HasPrefix(data, fmDelim) {
		return nil, data, 0
	}

	// A primeira linha tem que ser exatamente "---" (com CR opcional).
	firstNL := bytes.IndexByte(data, '\n')
	if firstNL < 0 {
		return nil, data, 0
	}
	first := bytes.TrimRight(data[:firstNL], "\r")
	if !bytes.Equal(first, fmDelim) {
		return nil, data, 0
	}

	rest := data[firstNL+1:]
	offset := int64(firstNL + 1)

	for len(rest) > 0 {
		nl := bytes.IndexByte(rest, '\n')
		var line []byte
		var advance int
		if nl < 0 {
			line, advance = rest, len(rest)
		} else {
			line, advance = rest[:nl], nl+1
		}

		if bytes.Equal(bytes.TrimRight(line, "\r"), fmDelim) {
			fmEnd := int(offset) - (firstNL + 1)
			body := rest[advance:]
			return data[firstNL+1 : firstNL+1+fmEnd], body, offset + int64(advance)
		}

		rest = rest[advance:]
		offset += int64(advance)
	}

	// Delimitador de abertura sem fechamento: nao ha frontmatter, ha um
	// arquivo que comeca com tres tracos.
	return nil, data, 0
}

// DecodeFrontmatter decodifica o YAML preservando os tipos do Obsidian:
// string, numero, booleano, lista e data.
func DecodeFrontmatter(fm []byte) (map[string]any, error) {
	if len(bytes.TrimSpace(fm)) == 0 {
		return nil, nil
	}

	var out map[string]any
	if err := yaml.Unmarshal(fm, &out); err != nil {
		return nil, fmt.Errorf("decodificando frontmatter: %w", err)
	}
	return out, nil
}
