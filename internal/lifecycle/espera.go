package lifecycle

import (
	"context"
	"log/slog"
	"runtime/pprof"
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
//
// # O rotulo, e por que ele mora aqui
//
// pprof.Do prende o nome da espera na GOROUTINE enquanto fn roda. Com a
// diretiva `go 1.27` no go.mod, esse rotulo sai no cabecalho da goroutine em
// qualquer traceback -- entao o dump de ArmarGuardaChuva nomeia a espera que
// travou em vez de mostrar um endereco.
//
// E aqui, e nao em cada chamador, pela mesma razao de a linha de log ser aqui:
// o nome da espera e UMA conta. Rotular no chamador seria escrever o mesmo
// nome duas vezes, e a copia menos usada e a que diverge.
//
// ctx aqui e PORTADOR DE ROTULO, e nao fonte de cancelamento -- a mesma
// excecao documentada de lifecycle.Shutdown, por outro motivo. Quem limita o
// tempo e ArmarGuardaChuva; este ctx ja chega cancelado nos cinco sitios, e
// isso nao afeta rotulo nenhum. Ele existe porque pprof.Do encadeia os rotulos
// do ctx que recebe: com context.Background() aqui dentro, todo rotulo que o
// chamador ja tivesse posto seria descartado -- que foi o que o contextcheck
// apontou na primeira redacao.
func Esperar(ctx context.Context, log *slog.Logger, nome string, fn func()) {
	log.Info("esperando no encerramento", "espera", nome)
	inicio := time.Now()
	pprof.Do(ctx, pprof.Labels("espera", nome), func(context.Context) {
		fn()
	})
	log.Info("espera concluida", "espera", nome, "duracao_ms", time.Since(inicio).Milliseconds())
}
