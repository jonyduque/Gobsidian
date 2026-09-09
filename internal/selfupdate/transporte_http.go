package selfupdate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TransporteHTTP e a implementacao de producao de Transporte, e o UNICO lugar
// do produto que fala HTTP.
//
// Ele existe num arquivo separado de proposito: quem procura "onde este
// produto acessa a rede?" encontra um arquivo, com um import, com um metodo.
// A resposta cabe numa tela.
//
// Ver o comentario do pacote e PRD 6.4 para a excecao da RNF-30 que autoriza
// este import, e o que a estreita.
type TransporteHTTP struct {
	// Cliente permite ao chamador ajustar prazos. Nulo usa clienteDefault.
	Cliente *http.Client
}

// prazoDeRede e o teto de uma requisicao inteira, conexao incluida.
//
// 60 s. O download de um binario de ~11 MB numa conexao ruim cabe nisso, e um
// `gobsidian update` que trava para sempre e pior que um que falha dizendo o
// que aconteceu -- a mesma razao pela qual todo encerramento deste produto tem
// orcamento desde 2026-09-08.
const prazoDeRede = 60 * time.Second

func (t TransporteHTTP) cliente() *http.Client {
	if t.Cliente != nil {
		return t.Cliente
	}
	return &http.Client{Timeout: prazoDeRede}
}

// Buscar faz a requisicao e devolve o corpo.
//
// Nao valida o host: quem valida e buscarValidando, ANTES de chegar aqui. Uma
// segunda validacao neste ponto seria uma segunda conta da mesma regra, e a
// que fica na copia menos usada e a que diverge.
func (t TransporteHTTP) Buscar(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("montando requisicao para %s: %w", url, err)
	}
	// A API do GitHub versiona por header. Sem ele, uma mudanca de default do
	// lado deles muda o formato da resposta sem aviso.
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "gobsidian-selfupdate")

	resp, err := t.cliente().Do(req)
	if err != nil {
		return nil, fmt.Errorf("buscando %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("buscando %s: status %d", url, resp.StatusCode)
	}
	return resp.Body, nil
}
