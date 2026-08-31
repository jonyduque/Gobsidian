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
// goroutine garante thread-safety sem reconstrução custosa — o padrão de sync.Pool deixa
// cada goroutine com sua própria instância, sem contenção. O Reset() antes de usar limpa
// qualquer estado residual da chamada anterior na mesma goroutine.
var transformerPool = sync.Pool{
	New: func() any {
		return transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	},
}

// RemoveAccents remove acentos mantendo a caixa original.
// Reutiliza transformer de um pool para evitar alocação a cada chamada.
// Thread-safe: sync.Pool garante que cada goroutine tenha sua própria instância.
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

// ParaNFC devolve a string na forma de normalizacao NFC, sem tocar em caixa
// nem em acento.
//
// E o oposto de Normalize em proposito: Normalize existe para BUSCA, onde
// "Capitulo" tem de casar com "Capítulo", e por isso remove acento. ParaNFC
// existe para CHAVE DE INDICE, onde remover acento faria "Capitulo" e
// "Capítulo" virarem a mesma nota — duas notas distintas colidindo numa
// entrada so.
//
// O que ela resolve: `í` precomposto (U+00ED, NFC) e `i` + acento combinante
// (U+0069 U+0301, NFD) sao a mesma letra para quem le e strings DIFERENTES para
// um mapa de Go. Um cofre sincronizado com macOS grava NFD e um cliente Windows
// pede NFC — e ate 2026-08-31 ResolvePath respondia "nao encontrada" para uma
// nota que existia.
//
// NFC e a forma canonica escolhida porque e o que a maioria dos clientes envia
// e o que o Go emite por padrao.
func ParaNFC(s string) string { return norm.NFC.String(s) }

// Normalize remove acentos e converte para caixa baixa.
// Reutiliza transformer de um pool para evitar alocação a cada chamada.
// Thread-safety: sync.Pool garante que cada goroutine tenha sua própria instância.
func Normalize(s string) string {
	return strings.ToLower(RemoveAccents(s))
}
