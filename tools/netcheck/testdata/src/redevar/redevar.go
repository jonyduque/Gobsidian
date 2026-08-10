package redevar

import "net"

var _ = func() {
	rede := "unix"
	_, _ = net.Dial(rede, "/tmp/x.sock") // want `rede proibida: net.Dial so aceita a constante "unix"`
}
