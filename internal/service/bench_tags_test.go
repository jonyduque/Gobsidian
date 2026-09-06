package service_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

// BenchmarkSearchFiltroTags mede a busca com filtro de tag: hoje o filtro
// baixa a caixa de cada tag de cada resultado por consulta; depois da
// Task 180 resolve o conjunto uma vez e faz busca binaria por resultado.
func BenchmarkSearchFiltroTags(b *testing.B) {
	svc := benchServicoDeCache(b)
	benchBusca(b, svc, service.SearchOptions{
		Query: "nota",
		Limit: 200,
		Tags:  []string{"golang"},
	}, 1)
}
