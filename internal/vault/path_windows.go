//go:build windows

package vault

import (
	"fmt"
	"strings"
)

// validatePlatformPath rejeita componentes terminados em ponto ou espaco.
//
// O Win32 os remove ao abrir, entao `A.md.`, `A.md ` e `A.md` sao o mesmo
// arquivo — mas tres chaves canonicas distintas. Como CanonicalPath e a
// identidade da nota no indice, isso da a um arquivo mais de uma identidade:
// a varredura indexaria uma grafia enquanto uma tool escreve sob outra, e
// qualquer verificacao chaveada no caminho canonico seria contornavel
// acrescentando um ponto.
//
// A regra vive atras de build tag porque a justificativa e inteiramente do
// Windows. Em Linux e macOS `Notas ` e `Arquivo.` sao nomes legais e
// distintos, que o Obsidian preserva — rejeita-los ali tornaria notas reais
// inalcancaveis, que e a mesma falha de fronteira que a travessia, so que na
// direcao oposta.
//
// A comparacao e do ultimo BYTE, nao do ultimo rune, e isso e proposital: o
// Win32 remove apenas espaco ASCII e ponto. Espacos unicode como U+00A0 nao
// sao removidos por ele e portanto nao criam identidade duplicada.
func validatePlatformPath(cleaned string) error {
	for _, part := range strings.Split(cleaned, "/") {
		if part == "" {
			continue
		}
		if last := part[len(part)-1]; last == '.' || last == ' ' {
			return fmt.Errorf(
				"%w: componente %q termina em ponto ou espaco, o que o Windows remove ao abrir e faria o mesmo arquivo ter mais de um caminho canonico",
				ErrInvalidPath, part)
		}
	}
	return nil
}
