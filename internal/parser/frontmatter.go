package parser

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

var fmDelim = []byte("---")

// SplitFrontmatter separa o bloco YAML do corpo, devolvendo tambem o offset
// em que o corpo comeca. O offset e o que mantem os offsets de heading e de
// bloco corretos em relacao ao inicio do buffer, nao ao corpo.
//
// O offset e relativo AO SLICE RECEBIDO, nao ao arquivo em disco. A distincao
// so importa quando ha BOM: como esta funcao exige entrada ja sem BOM, o
// buffer e tres bytes mais curto que o arquivo. Para converter um offset daqui
// em posicao no arquivo, quem tiver o arquivo precisa somar len(bom) quando
// vault.StripBOM tiver reportado true. Nao somar produz deslocamento de
// exatamente tres bytes em toda leitura de secao — silencioso, e so em notas
// com BOM.
//
// Exige entrada ja sem BOM UTF-8. Quem produz essa garantia e
// vault.StripBOM — este pacote nao a chama, para nao duplicar logica que ja
// mora em internal/vault. Sem ela, "\xEF\xBB\xBF---" nao bate com o prefixo
// "---" esperado: o frontmatter nao fica malformado, fica invisivel. Nao ha
// FrontmatterErr, tags/aliases/title somem em silencio, e as linhas "---"
// viram conteudo da nota.
func SplitFrontmatter(data []byte) ([]byte, []byte, int64) {
	if !bytes.HasPrefix(data, fmDelim) {
		return nil, data, 0
	}

	// A primeira linha tem que ser exatamente "---" (com CR opcional).
	firstNL := bytes.IndexByte(data, '\n')
	if firstNL < 0 {
		return nil, data, 0
	}
	first := bytes.TrimRight(data[:firstNL], " \t\r")
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

		if bytes.Equal(bytes.TrimRight(line, " \t\r"), fmDelim) {
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
