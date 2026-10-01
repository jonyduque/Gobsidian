package httptestfora

import (
	"net/http/httptest" // want `pacote de rede proibido: net/http/httptest`
	"testing"
)

// Fora do pacote da excecao, nem em teste.
func TestFora(t *testing.T) { _ = httptest.NewTestServer(t, nil) }
