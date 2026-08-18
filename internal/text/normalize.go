// Package text oferece utilidades de processamento de texto.
package text

import (
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// transformerPool reutiliza transformers entre chamadas de Normalize e RemoveAccents. Um transformer por
// goroutine garante thread-safety sem reconstrução custosa — o padrÃo de sync.Pool deixa
// cada goroutina com sua própria instância, sem contenção. O Reset() antes de usar limpa
// qualquer estado residual da chamada anterior na mesma goroutina.
var transformerPool = sync.Pool{
	New: func() any {
		return transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	},
}

// RemoveAccents remove acentos mantendo a caixa original.
// Reutiliza transformer de um pool para evitar alocação a cada chamada.
// Thread-safe: sync.Pool garante que cada goroutina tenha sua própria instância.
func RemoveAccents(s string) string {
	t := transformerPool.Get().(transform.Transformer)
	defer transformerPool.Put(t)
	t.Reset()
	res, _, err := transform.String(t, s)
	if err != nil {
		res = s
	}
	return res
}

// Normalize remove acentos e converte para caixa baixa.
// Reutiliza transformer de um pool para evitar alocação a cada chamada.
// Thread-safety: sync.Pool garante que cada goroutina tenha sua própria instância.
func Normalize(s string) string {
	return strings.ToLower(RemoveAccents(s))
}
