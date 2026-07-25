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
