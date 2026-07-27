package lifecycle

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// Step e uma etapa da sequencia de encerramento, com orcamento proprio.
// Um timeout global unico nao basta: se as chamadas em voo consumirem o
// orcamento inteiro, a persistencia do cache comeca ja estourada.
type Step struct {
	Name   string
	Budget time.Duration
	Fn     func(context.Context) error
}

// Shutdown executa as etapas em ordem, cada uma com seu proprio relogio.
//
// Nao devolve codigo de saida, e a ausencia e deliberada. Uma etapa que
// devolve erro e registrada, nunca fatal: o cache de indice que nao gravou
// custa alguns segundos no proximo boot, e transformar isso em codigo de erro
// faria o host reportar falha de um encerramento que deu certo. Um panic
// dentro de uma etapa NAO esta coberto por essa garantia — derruba o processo,
// como em qualquer goroutine do pacote. Nao ha recover aqui de proposito: o
// processo ja esta encerrando, e um panic durante o encerramento e um bug que
// deve aparecer inteiro no stderr, nao ser engolido.
//
// ATENCAO ao ctx: Shutdown recebe o context raiz e DESCARTA o cancelamento
// dele, de proposito, via context.WithoutCancel. Nao e descuido, e nao adianta
// passar um context cancelado esperando interromper o encerramento — nada aqui
// observa cancelamento.
//
// O motivo e que o context raiz ja esta cancelado quando Shutdown roda: e o
// cancelamento dele que traz o processo ate aqui. Derivar os orcamentos das
// etapas desse context faria toda etapa nascer expirada e ser abandonada antes
// de rodar. WithoutCancel preserva os valores do context (rastreamento, logging
// com escopo) e joga fora so a parte que aqui e veneno.
//
// Quem limita o tempo sao os orcamentos por etapa e o guarda-chuva abaixo, no
// lugar do context. hardLimit arma esse guarda-chuva: se qualquer etapa travar
// de um jeito que o orcamento dela nao previu, o processo morre mesmo assim.
// Encerrar com codigo de erro e ruim; sobreviver ao pai e pior.
func Shutdown(ctx context.Context, log *slog.Logger, hardLimit time.Duration, steps ...Step) {
	base := context.WithoutCancel(ctx)

	guard := time.AfterFunc(hardLimit, func() {
		log.Error("encerramento travou alem do limite rigido", "limit", hardLimit)
		os.Exit(1)
	})
	defer guard.Stop()

	for _, step := range steps {
		start := time.Now()
		stepCtx, cancel := context.WithTimeout(base, step.Budget)

		done := make(chan error, 1)
		go func() { done <- step.Fn(stepCtx) }()

		select {
		case err := <-done:
			if err != nil {
				log.Warn("etapa de encerramento falhou", "step", step.Name, "err", err, "duration", time.Since(start))
			} else {
				log.Debug("etapa de encerramento concluida", "step", step.Name, "duration", time.Since(start))
			}
		case <-stepCtx.Done():
			// A goroutine da etapa fica orfa de proposito. Ela recebeu um ctx
			// cancelado e deve sair sozinha; esperar por ela e exatamente o
			// que o orcamento existe para evitar.
			log.Warn("etapa de encerramento abandonada por estouro de orcamento",
				"step", step.Name, "budget", step.Budget)
		}
		cancel()
	}
}
