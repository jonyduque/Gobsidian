package writer

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
)

// HeadingNotFoundError indica que o heading especificado nao foi encontrado na nota.
type HeadingNotFoundError struct {
	Heading      string
	Alternatives []string
}

func (e *HeadingNotFoundError) Error() string {
	if len(e.Alternatives) == 0 {
		return fmt.Sprintf("heading %q nao encontrado na nota (nenhum heading disponivel)", e.Heading)
	}
	return fmt.Sprintf("heading %q nao encontrado na nota. Disponiveis: %s", e.Heading, strings.Join(e.Alternatives, ", "))
}

// AmbiguousHeadingError indica colisao de slug: multiplos headings possuem o mesmo slug/titulo.
type AmbiguousHeadingError struct {
	Heading     string
	Occurrences int
}

func (e *AmbiguousHeadingError) Error() string {
	return fmt.Sprintf("heading %q e ambiguo (%d ocorrencias encontradas)", e.Heading, e.Occurrences)
}

// DetectEOL identifica se o buffer utiliza CRLF (\r\n) ou LF (\n).
//
// Delega para vault.DetectEOL, que e a UNICA conta de EOL do projeto.
//
// Ate 2026-08-27 esta funcao tinha regra propria — QUALQUER "\r\n" no arquivo
// bastava para chamar o arquivo inteiro de CRLF —, enquanto vault.DetectEOL usa
// o estilo PREDOMINANTE, e e a resposta dela que o indice persiste em Note.EOL.
// Um arquivo com uma linha CRLF e mil LF era LF para o indice e CRLF para a
// escrita: a edicao de secao gravava CRLF num arquivo que o resto do sistema
// tratava como LF. Achado M14; e a mesma classe do bug que este projeto ja
// pagou caro, duas contas do mesmo fato concordando por coincidencia.
//
// A assinatura continua devolvendo string porque os cinco chamadores concatenam
// o resultado em texto; vault.EOLStyle.Bytes() ja e a conversao unica.
func DetectEOL(content []byte) string {
	return string(vault.DetectEOL(content).Bytes())
}

// NormalizeEOL normaliza o texto fornecido para a convencao de EOL alvo (\r\n ou \n).
func NormalizeEOL(text, targetEOL string) string {
	if targetEOL == "\r\n" {
		clean := strings.ReplaceAll(text, "\r\n", "\n")
		clean = strings.ReplaceAll(clean, "\r", "\n")
		return strings.ReplaceAll(clean, "\n", "\r\n")
	}
	clean := strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(clean, "\r", "\n")
}

// FindHeading busca um heading unico na lista por slug ou titulo exato.
func FindHeading(headings []parser.Heading, headingQuery string) (*parser.Heading, error) {
	targetSlug := parser.Slug(headingQuery)
	var matches []parser.Heading

	for _, h := range headings {
		if h.Slug == targetSlug || strings.EqualFold(h.Text, headingQuery) {
			matches = append(matches, h)
		}
	}

	if len(matches) == 0 {
		var alternatives []string
		for _, h := range headings {
			alternatives = append(alternatives, h.Text)
		}
		return nil, &HeadingNotFoundError{Heading: headingQuery, Alternatives: alternatives}
	}

	if len(matches) > 1 {
		return nil, &AmbiguousHeadingError{Heading: headingQuery, Occurrences: len(matches)}
	}

	return &matches[0], nil
}

// PatchSectionContent substitui o conteudo da secao delimitada por h em rawContent por replacement.
// Preserva o BOM do arquivo se presente, a linha do titulo do heading, a convencao EOL (CRLF/LF)
// e todo o conteudo fora do intervalo [h.BodyStart, h.End].
func PatchSectionContent(rawContent []byte, h parser.Heading, replacement string) []byte {
	eol := DetectEOL(rawContent)
	normReplacement := NormalizeEOL(replacement, eol)

	if normReplacement != "" && !strings.HasSuffix(normReplacement, eol) {
		normReplacement += eol
	}

	var buf bytes.Buffer
	buf.Write(rawContent[:h.BodyStart])
	buf.WriteString(normReplacement)
	buf.Write(rawContent[h.End:])

	return buf.Bytes()
}

// AppendSectionContent anexa contentToAppend no final da secao h (se h != nil) ou no final da nota (se h == nil).
// Preserva a convencao EOL do arquivo.
func AppendSectionContent(rawContent []byte, h *parser.Heading, contentToAppend string) []byte {
	eol := DetectEOL(rawContent)
	normAppend := NormalizeEOL(contentToAppend, eol)

	if h == nil {
		var buf bytes.Buffer
		buf.Write(rawContent)
		if len(rawContent) > 0 && !bytes.HasSuffix(rawContent, []byte("\n")) {
			buf.WriteString(eol)
		}
		buf.WriteString(normAppend)
		if !strings.HasSuffix(normAppend, eol) {
			buf.WriteString(eol)
		}
		return buf.Bytes()
	}

	insertOffset := h.End
	var buf bytes.Buffer
	buf.Write(rawContent[:insertOffset])

	if insertOffset > 0 && !bytes.HasSuffix(rawContent[:insertOffset], []byte("\n")) {
		buf.WriteString(eol)
	}

	buf.WriteString(normAppend)
	if !strings.HasSuffix(normAppend, eol) {
		buf.WriteString(eol)
	}
	buf.Write(rawContent[insertOffset:])

	return buf.Bytes()
}
