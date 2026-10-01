package selfupdate

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestTransporteHTTPBuscaComOsHeadersDaAPI exercita o UNICO codigo do produto
// que faz rede, pela pilha HTTP de verdade do cliente.
//
// httptest.NewTestServer (Go 1.27) roda numa rede em memoria e recebe qualquer
// URL -- inclusive https://api.github.com -- sem abrir socket: a RNF-30 e
// respeitada no proprio teste. tools/netcheck aceita o httptest so aqui, so em
// _test.go e so este construtor (excecao de 2026-10-01). Ate essa data o
// pacote so tinha o transporteFalso, e nada provava os headers que a API do
// GitHub exige nem o tratamento de status.
func TestTransporteHTTPBuscaComOsHeadersDaAPI(t *testing.T) {
	recebidos := make(chan http.Header, 1)
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recebidos <- r.Header.Clone()
		if r.URL.Path != "/repos/jonyduque/Gobsidian/releases/latest" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"tag_name":"v9.9.9"}`)
	}))
	tr := TransporteHTTP{Cliente: srv.Client()}

	corpo, err := tr.Buscar(context.Background(), "https://api.github.com/repos/jonyduque/Gobsidian/releases/latest")
	if err != nil {
		t.Fatalf("Buscar: %v", err)
	}
	b, err := io.ReadAll(corpo)
	_ = corpo.Close()
	if err != nil || string(b) != `{"tag_name":"v9.9.9"}` {
		t.Fatalf("corpo = %q, err = %v", b, err)
	}
	h := <-recebidos
	for nome, quer := range map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-Github-Api-Version": "2022-11-28",
		"User-Agent":           "gobsidian-selfupdate",
	} {
		if got := h.Get(nome); got != quer {
			t.Errorf("header %s = %q, quer %q", nome, got, quer)
		}
	}

	// Status diferente de 200 e erro com o status, e nao corpo de pagina de erro
	// tratado como resposta.
	_, err = tr.Buscar(context.Background(), "https://api.github.com/repos/outro/repo/releases/latest")
	<-recebidos
	if err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Errorf("err = %v, quer erro com status 404", err)
	}
}

// TestTransporteHTTPTemPrazo: o cliente padrao nao pode esperar para sempre.
func TestTransporteHTTPTemPrazo(t *testing.T) {
	if got := (TransporteHTTP{}).cliente().Timeout; got != prazoDeRede {
		t.Errorf("Timeout do cliente padrao = %v, quer %v", got, prazoDeRede)
	}
}
