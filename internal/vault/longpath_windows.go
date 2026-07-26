//go:build windows

package vault

import (
	"path/filepath"
	"strings"
)

// longPathThreshold e conservador de proposito: o limite e 260, e a folga
// cobre o nome do arquivo temporario que a escrita atomica cria ao lado do
// alvo.
const longPathThreshold = 240

// LongPath prefixa com \\?\ quando o caminho se aproxima de MAX_PATH.
//
// Restricoes do prefixo, que o chamador precisa ter respeitado antes: exige
// caminho absoluto, exige separador "\", e nao aceita "." nem "..".
func LongPath(abs string) string {
	if len(abs) < longPathThreshold {
		return abs
	}
	if strings.HasPrefix(abs, `\\?\`) {
		return abs
	}
	clean := filepath.Clean(abs)
	if strings.HasPrefix(clean, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(clean, `\\`)
	}
	return `\\?\` + clean
}
