package a

import (
	"net/http" // want `pacote de rede proibido: net/http`
	"os"
)

var _ = os.Getenv
var _ = http.Get
