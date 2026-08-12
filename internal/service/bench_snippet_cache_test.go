package service_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
)

// BenchmarkSearchLimit200CacheTrechoRepetido mede a MESMA consulta repetida com
// o cache de trechos ligado. O nome carrega "Repetido" porque isso é o que ele
// mede, e vendê-lo como o caso frio seria mentira: b.Loop repete a consulta,
// então a partir da segunda volta todos os 200 recortes acertam o cache.
//
// A repetição não é artificial — é o padrão de um agente MCP, que refaz a mesma
// busca ao longo de uma conversa. Mas o alvo do RNF-04 é o caminho frio, e quem
// o mede é BenchmarkSearchLimit200Cache, com o cache desligado.
//
// O par frio/quente sobre o mesmo cofre, o mesmo índice vindo do cache e a mesma
// consulta é o que permite atribuir a diferença ao cache de trechos.
func BenchmarkSearchLimit200CacheTrechoRepetido(b *testing.B) {
	entradas := search.DefaultSnippetCacheEntries
	svc := benchServicoDeCacheCom(b, &entradas)
	benchBusca(b, svc, service.SearchOptions{Query: "nota", Limit: 200}, 200)
}
