package outronet

import "net"

var _ = func() {
	_, _ = net.DialTCP("tcp", nil, nil) // want `chamada de rede proibida: net.DialTCP`
}
