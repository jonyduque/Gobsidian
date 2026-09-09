package lifecycle

import (
	"log/slog"
	"time"
)

// Esperar roda uma espera de encerramento dizendo NO LOG que ela comecou e
// quanto levou.
//
// A linha de entrada e o ponto inteiro. Medido em 2026-09-07: o daemon PID
// 42856 registrou "encerramento solicitado reason=idle", nunca registrou
// "daemon encerrado", e ficou vivo 20 h. Sabe-se que o travamento estava numa
// das tres esperas seguintes; QUAL delas nao foi possivel determinar, porque
// nenhuma dizia que tinha comecado. Com esta linha, a ultima do log nomeia a
// espera que nao voltou -- que e a resposta que custou horas de analise e
// continuou faltando.
//
// INFO, e nao Debug: o log do daemon roda em INFO em producao, e uma linha que
// so aparece quando alguem ja sabia que precisava dela nao serve para o caso
// em que ninguem sabia.
//
// Nao ha orcamento aqui: quem limita e lifecycle.ArmarGuardaChuva, que cobre o
// encerramento inteiro. Duas contas do mesmo teto divergiriam.
func Esperar(log *slog.Logger, nome string, fn func()) {
	log.Info("esperando no encerramento", "espera", nome)
	inicio := time.Now()
	fn()
	log.Info("espera concluida", "espera", nome, "duracao_ms", time.Since(inicio).Milliseconds())
}
