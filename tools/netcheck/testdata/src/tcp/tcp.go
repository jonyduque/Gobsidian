package tcp

import "net"

var _ = func() {
	_, _ = net.Dial("tcp", "example.com:80") // want `rede proibida: net.Dial so aceita a constante "unix"`
}
