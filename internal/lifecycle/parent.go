package lifecycle

import (
	"context"
	"time"
)

// watchParent e a rede de seguranca, para o caso em que o host morre de forma
// que deixe o stdin do filho sem fechar: reparentamento, heranca de handle por
// outro processo, comportamento anomalo do host.
//
// A verificacao compara identidade, nao apenas existencia. PIDs sao reciclados
// agressivamente no Windows, e checar so o PID produz falso negativo quando o
// PID do pai morto e atribuido a um processo novo.
//
// Como watchSignals, esta goroutine vai para o WaitGroup: o select cobre
// tanto o tick do ticker quanto o cancelamento do contexto, entao ela sempre
// tem uma saida real, diferente da goroutine de stdin presa num Read
// bloqueante.
func (l *Lifecycle) watchParent(ctx context.Context, pid int, interval time.Duration) {
	initial, err := parentIdentity(pid)
	if err != nil {
		// Nao da para vigiar o que nao da para identificar. Os outros dois
		// mecanismos continuam de pe.
		l.log.Warn("vigilia do processo pai desabilitada", "pid", pid, "err", err)
		return
	}

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				current, err := parentIdentity(pid)
				if err != nil || !sameProcess(initial, current) {
					l.trigger("parent-gone")
					return
				}
			}
		}
	}()
}
