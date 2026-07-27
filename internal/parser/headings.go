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
	var fence fenceInfo

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
		//
		// Dentro de uma cerca so interessa saber se a linha a FECHA. Tratar
		// qualquer linha de crases como abertura-ou-fechamento indiferente faz
		// uma cerca de quatro crases ser fechada por uma linha de tres — e a
		// partir dali toda a hierarquia sai errada, porque o conteudo que
		// ainda esta dentro do bloco passa a ser lido como estrutura.
		if inFence {
			if closesFence(text, fence) {
				inFence = false
			}
			pos += advance
			body = body[advance:]
			continue
		}

		if f, ok := openFence(text); ok {
			inFence, fence = true, f
			pos += advance
			body = body[advance:]
			continue
		}

		if level, title, ok := parseATXHeading(text); ok {
			out = append(out, Heading{
				Level:     level,
				Text:      title,
				Slug:      Slug(title),
				Start:     bodyOffset + pos,
				BodyStart: bodyOffset + pos + advance,
			})
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

	// Sequencia de fechamento opcional: "## Titulo ##". O CommonMark so a
	// remove quando vem precedida de espaco ou quando e todo o conteudo.
	// Remover incondicionalmente transforma "# Notas sobre C#" em
	// "Notas sobre C", e um heading que termina em '#' deixa de ser
	// enderecavel por note_read, por note_patch e por ancora de wikilink.
	if closing := len(title) - len(strings.TrimRight(title, "#")); closing > 0 {
		rest := title[:len(title)-closing]
		if rest == "" || strings.HasSuffix(rest, " ") || strings.HasSuffix(rest, "\t") {
			title = rest
		}
	}

	return level, strings.TrimSpace(title), true
}

// fenceInfo descreve uma cerca aberta. O comprimento importa: o CommonMark
// exige que a cerca de fechamento tenha pelo menos o tamanho da de abertura,
// o que e justamente o que permite mostrar um bloco de tres crases dentro de
// um bloco de quatro — padrao comum em nota que documenta Markdown.
type fenceInfo struct {
	char  byte
	count int
}

// openFence reconhece uma linha que ABRE uma cerca e devolve o caractere e
// quantos dele. Ate tres espacos de indentacao sao permitidos; quatro fazem a
// linha virar bloco de codigo indentado.
func openFence(line string) (fenceInfo, bool) {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 {
		return fenceInfo{}, false
	}

	var c byte
	switch {
	case strings.HasPrefix(t, "```"):
		c = '`'
	case strings.HasPrefix(t, "~~~"):
		c = '~'
	default:
		return fenceInfo{}, false
	}

	n := 0
	for n < len(t) && t[n] == c {
		n++
	}

	// A info string de uma cerca de crase nao pode conter crase — senao
	// `` `x` `` em texto corrido abriria um bloco.
	if c == '`' && strings.ContainsRune(t[n:], '`') {
		return fenceInfo{}, false
	}

	return fenceInfo{char: c, count: n}, true
}

// closesFence diz se a linha fecha a cerca aberta. Precisa do mesmo caractere,
// pelo menos o mesmo comprimento, e nada alem de espaco depois — uma cerca de
// fechamento nao aceita info string.
func closesFence(line string, open fenceInfo) bool {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 {
		return false
	}

	n := 0
	for n < len(t) && t[n] == open.char {
		n++
	}
	if n < open.count {
		return false
	}

	return strings.TrimSpace(t[n:]) == ""
}
