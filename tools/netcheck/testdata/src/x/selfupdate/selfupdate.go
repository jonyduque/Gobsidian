package selfupdate

import (
	"net/http"
	"net/http/httptest" // want `pacote de rede proibido: net/http/httptest`
)

var _ = http.Get
var _ = httptest.NewRecorder
