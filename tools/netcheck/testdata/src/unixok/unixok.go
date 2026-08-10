// Package unixok exercita os dois casos que a regra tem de deixar passar:
// net.Dial e net.Listen com a constante literal "unix". Se o analisador
// reportar qualquer coisa aqui, ele está barrando o caso que a Task 91/92
// precisa usar de verdade.
package unixok

import "net"

var _ = func() {
	_, _ = net.Dial("unix", "/tmp/x.sock")
}

var _ = func() {
	_, _ = net.Listen("unix", "/tmp/x.sock")
}
