// Fixture do gate: o MESMO import em qualquer outro pacote continua proibido.
// Se este caso passar, a excecao virou porta escancarada.
package qualquer

import "net/http"

func buscar() (*http.Response, error) {
	return http.Get("https://api.github.com")
}
