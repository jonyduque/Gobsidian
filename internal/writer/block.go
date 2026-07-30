package writer

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jonyd/gobsidian/internal/parser"
)

// BlockNotFoundError indica que o block id especificado nao foi encontrado na nota.
type BlockNotFoundError struct {
	BlockID string
}

func (e *BlockNotFoundError) Error() string {
	return fmt.Sprintf("bloco ^%s nao encontrado na nota", strings.TrimPrefix(e.BlockID, "^"))
}

// AmbiguousBlockError indica colisao de block id (mais de um marcador igual na mesma nota).
type AmbiguousBlockError struct {
	BlockID     string
	Occurrences int
}

func (e *AmbiguousBlockError) Error() string {
	return fmt.Sprintf("bloco ^%s e ambiguo (%d ocorrencias encontradas)", strings.TrimPrefix(e.BlockID, "^"), e.Occurrences)
}

// FindBlock busca um bloco unico por ID na lista de blocos da nota.
func FindBlock(blocks []parser.Block, blockID string) (*parser.Block, error) {
	cleanID := strings.TrimPrefix(blockID, "^")
	var matches []parser.Block

	for _, b := range blocks {
		if b.ID == cleanID {
			matches = append(matches, b)
		}
	}

	if len(matches) == 0 {
		return nil, &BlockNotFoundError{BlockID: cleanID}
	}

	if len(matches) > 1 {
		return nil, &AmbiguousBlockError{BlockID: cleanID, Occurrences: len(matches)}
	}

	return &matches[0], nil
}

// ReplaceBlockContent substitui o conteudo do bloco b em rawContent por replacement.
// Preserva o identificador ^id no final do bloco substituido para manter as referencias do Obsidian validas.
// Preserva a convencao EOL (\r\n ou \n) e o BOM original do arquivo.
func ReplaceBlockContent(rawContent []byte, b parser.Block, replacement string) []byte {
	eol := DetectEOL(rawContent)
	normReplacement := strings.TrimRight(NormalizeEOL(replacement, eol), eol)

	idMarker := "^" + b.ID
	if !strings.HasSuffix(normReplacement, idMarker) {
		if normReplacement != "" && !strings.HasSuffix(normReplacement, " ") {
			normReplacement += " "
		}
		normReplacement += idMarker
	}

	var buf bytes.Buffer
	buf.Write(rawContent[:b.Start])
	buf.WriteString(normReplacement)
	buf.Write(rawContent[b.End:])

	return buf.Bytes()
}
