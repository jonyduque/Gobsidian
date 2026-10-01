package selfupdate

import (
	"net/http/httptest"
	"testing"
)

// Em teste do pacote da excecao, httptest e aceito -- mas so NewTestServer,
// que roda numa rede em memoria. Os construtores que abrem socket continuam
// recusados.
func TestTransporte(t *testing.T) {
	_ = httptest.NewTestServer(t, nil)
	_ = httptest.NewServer(nil)          // want `httptest.NewServer abre socket`
	_ = httptest.NewTLSServer(nil)       // want `httptest.NewTLSServer abre socket`
	_ = httptest.NewUnstartedServer(nil) // want `httptest.NewUnstartedServer abre socket`
}
