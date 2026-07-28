package mcpsrv

import (
	"fmt"
	"strings"

	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
)

// O esquema tem TRES barras de proposito.
//
// "gobsidian://" + caminho parece natural e esta errado: em RFC 3986, o que vem
// logo depois de "//" e a AUTORIDADE, nao o caminho. Concatenar assim faz o
// primeiro segmento do caminho canonico virar nome de host, e nome de host nao
// aceita espaco. O resultado era um panic no boot, dentro do AddResource do SDK:
//
//	panic: parse "gobsidian://test vault/Origem.md": invalid character " " in host name
//
// Pasta com espaco no nome e o caso comum num cofre do Obsidian. Com a forma
// antiga, um cofre assim derrubava o servidor antes de ele anunciar uma unica
// tool — o host via o processo sumir sem erro utilizavel.
//
// "gobsidian:///" declara autoridade vazia e faz o caminho comecar onde deve.
const resourceScheme = "gobsidian://"

// unreserved sao os caracteres que RFC 3986 permite crus em qualquer posicao.
// A barra entra na lista porque separa segmentos: escapa-la achataria a
// hierarquia num nome unico, e o resource deixaria de casar com o caminho
// canonico que o handler resolve.
func isUnreservedPathByte(b byte) bool {
	switch {
	case b >= 'A' && b <= 'Z', b >= 'a' && b <= 'z', b >= '0' && b <= '9':
		return true
	case b == '-', b == '.', b == '_', b == '~', b == '/':
		return true
	}
	return false
}

const hexDigits = "0123456789ABCDEF"

// resourceURI monta a URI publicada para uma nota.
//
// Escapa byte a byte, e nao rune a rune: um caractere acentuado em UTF-8 vira
// varios %XX seguidos, e e assim que a decodificacao do outro lado remonta o
// caractere. Escapar por rune produziria algo que so este servidor entenderia.
func resourceURI(p vault.CanonicalPath) string {
	s := string(p)

	var b strings.Builder
	b.Grow(len(resourceScheme) + 1 + len(s))
	b.WriteString(resourceScheme)
	b.WriteByte('/')

	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreservedPathByte(c) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hexDigits[c>>4])
		b.WriteByte(hexDigits[c&0x0F])
	}

	return b.String()
}

// pathFromResourceURI faz o caminho inverso.
//
// Aceita tambem a forma antiga de duas barras, que e a que docs/TOOLS.md e o
// AD-08 descreviam. Um host que tenha montado a URI a partir daquele texto
// continua conseguindo ler a nota; recusar transformaria documentacao
// desatualizada em nota inalcancavel, que e um custo desproporcional a uma
// linha de tolerancia.
//
// A decodificacao vem de parser.PercentDecode em vez de uma copia local: as
// regras de escape invalido sao sutis — "50% de desconto" tem um percent que
// nao inicia escape — e duas implementacoes divergem no dia em que alguem
// corrigir so uma.
func pathFromResourceURI(uri string) (string, error) {
	if !strings.HasPrefix(uri, resourceScheme) {
		return "", fmt.Errorf("esquema de URI invalido: %s", uri)
	}

	rest := strings.TrimPrefix(uri, resourceScheme)
	rest = strings.TrimPrefix(rest, "/")

	return parser.PercentDecode(rest), nil
}
