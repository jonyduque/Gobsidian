//go:build !windows

package console

import (
	"os"
	"strings"
)

// suportaUnicode le o locale.
//
// Fora do Windows nao ha code page de console: quem decide e o locale, e a
// convencao de trinta anos e o sufixo ".UTF-8" em LC_ALL, LC_CTYPE ou LANG,
// nessa ordem de precedencia -- a mesma que a libc usa.
//
// Sem nenhuma das tres, a resposta e ASCII. Um ambiente que nao declara o
// locale costuma ser CI, cron ou container minimo, e nenhum deles tem o que
// fazer com uma caixa arredondada.
func suportaUnicode() bool {
	for _, chave := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := os.Getenv(chave); v != "" {
			return strings.Contains(strings.ToUpper(v), "UTF-8") ||
				strings.Contains(strings.ToUpper(v), "UTF8")
		}
	}
	return false
}
