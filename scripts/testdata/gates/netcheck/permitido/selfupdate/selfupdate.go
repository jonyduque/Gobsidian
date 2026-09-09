// Fixture do gate: net/http DENTRO de internal/selfupdate e o unico lugar onde
// a excecao da RNF-30 vale (decisao D-13 do dono, 2026-09-08). O host esta numa
// constante literal.
package selfupdate

import "net/http"

const hostDaAPI = "https://api.github.com"

func buscar() (*http.Response, error) {
	return http.Get(hostDaAPI + "/repos/x/y/releases/latest")
}
