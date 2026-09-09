// Fixture do gate: URL com host FORA da lista, dentro de internal/selfupdate.
// A excecao da RNF-30 e para falar com o repositorio de releases, e nao com
// qualquer lugar. Sem este caso, a excecao seria "internal/selfupdate pode
// tudo".
package selfupdate

import "net/http"

const hostEstranho = "https://exemplo-que-nao-e-o-repositorio.invalid"

func buscar() (*http.Response, error) {
	return http.Get(hostEstranho + "/payload")
}
