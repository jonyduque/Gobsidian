package text

import (
	"strconv"
	"strings"
	"unicode"
)

// VersaoDasTabelas devolve a geracao das tabelas Unicode da TOOLCHAIN --
// 17 para `unicode.Version` "17.0.0".
//
// # Por que isto existe
//
// Ha duas fontes de Unicode neste binario, e so uma se mexe sozinha:
//
//   - `norm.NFC` e `norm.NFD` vem de `golang.org/x/text`, modulo FIXADO no
//     go.mod. Trocar de toolchain nao os move.
//   - `strings.ToLower`, `unicode.IsLetter` e `unicode.IsDigit` vem da
//     stdlib. Trocar de toolchain move os tres, e ninguem e avisado.
//
// O Go 1.27 subiu as tabelas da 15 para a 17. `internal/search/analyzer.go`
// decide o que e termo com `unicode.IsLetter`/`IsDigit`, e o resultado dessa
// decisao e PERSISTIDO no indice invertido. Um code point nao atribuido na 15
// e atribuido como letra na 17 passa a formar termo -- e um cache gravado pelo
// binario velho continuava sendo lido como valido pelo novo, porque os tres
// botoes de versao do cabecalho (formato, parser, analisador) sao bumpados a
// mao e ninguem bumpa um deles por causa de upgrade de toolchain.
//
// O cache de METADADOS (`internal/index`) nao precisa disto: ele recalcula
// toda chave derivada ao carregar -- `publishNoteLocked` chama `ChaveDeTag`,
// e `LoadIndexCache` refaz byAlias, backlinks e a resolucao de links pelas
// MESMAS funcoes que `Build` chama. So o cache de BUSCA guarda o que o
// analisador produziu.
//
// `config.VaultKey` tambem passava por `strings.ToLower` e por isso deixou de
// passar -- ver o comentario de la: chave que nomeia socket nao pode se mover.
func VersaoDasTabelas() int {
	maior, _, _ := strings.Cut(unicode.Version, ".")
	n, err := strconv.Atoi(maior)
	if err != nil {
		// unicode.Version e uma constante da stdlib e sempre comeca com o
		// numero maior. Se um dia nao comecar, zero e a resposta honesta: ela
		// difere de qualquer geracao real e descarta o cache, que e o lado
		// seguro de errar.
		return 0
	}
	return n
}
