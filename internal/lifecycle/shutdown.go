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
// Nao devolve codigo de saida, e a ausencia e deliberada. Falha de etapa e
// registrada, nunca fatal: o cache de indice que nao gravou custa alguns
// segundos no proximo boot, e transformar isso em codigo de erro faria o host
// reportar falha de um encerramento que deu certo. O unico caminho que sai
// com codigo diferente de zero e o guarda-chuva abaixo.
//
// hardLimit arma esse guarda-chuva: se qualquer etapa travar de um jeito que o
// orcamento dela nao previu, o processo morre mesmo assim. Encerrar com codigo
// de erro e ruim; sobreviver ao pai e pior.
func Shutdown(log *slog.Logger, hardLimit time.Duration, steps ...Step) {
	guard := time.AfterFunc(hardLimit, func() {
		log.Error("encerramento travou alem do limite rigido", "limit", hardLimit)
		os.Exit(1)
	})
	defer guard.Stop()

	for _, step := range steps {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), step.Budget)

		done := make(chan error, 1)
		go func() { done <- step.Fn(ctx) }()

		select {
		case err := <-done:
			if err != nil {
				log.Warn("etapa de encerramento falhou", "step", step.Name, "err", err, "duration", time.Since(start))
			} else {
				log.Debug("etapa de encerramento concluida", "step", step.Name, "duration", time.Since(start))
			}
		case <-ctx.Done():
			// A goroutine da etapa fica orfa de proposito. Ela recebeu um ctx
			// cancelado e deve sair sozinha; esperar por ela e exatamente o
			// que o orcamento existe para evitar.
			log.Warn("etapa de encerramento abandonada por estouro de orcamento",
				"step", step.Name, "budget", step.Budget)
		}
		cancel()
	}
}
