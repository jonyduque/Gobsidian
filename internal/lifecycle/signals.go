package lifecycle

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// watchSignals cobre o encerramento cooperativo: Ctrl+C no terminal, SIGTERM
// de um supervisor. No Windows, os.Interrupt cobre CTRL_C_EVENT e
// CTRL_BREAK_EVENT; SIGTERM e aceito pela API mas nao e entregue por
// taskkill sem /F. Nao cobre SIGKILL nem taskkill /F — e nao precisa:
// nesses casos o processo morre de fato, que e o resultado desejado.
// Diferente do observador de stdin, este gorrotina vai para o WaitGroup,
// porque tem uma saida de verdade. O select cobre tanto a chegada de um
// sinal quanto o cancelamento do contexto, assim o gorrotina termina
// quando o processo comeca o encerramento — nao importa qual dos tres
// mecanismos o acionou. signal.Stop e deferido para a runtime deixar de
// entregar sinais para um canal que ninguem le mais.
func (l *Lifecycle) watchSignals(ctx context.Context) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		defer signal.Stop(ch)

		select {
		case <-ch:
			l.trigger("signal")
		case <-ctx.Done():
		}
	}()
}
