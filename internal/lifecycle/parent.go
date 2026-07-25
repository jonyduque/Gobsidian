package lifecycle

import (
	"context"
	"time"
)

// maxConsecutiveFailures e quantas consultas seguidas precisam falhar antes
// de concluir que o pai morreu.
//
// Uma consulta que falha nao e prova de morte: OpenProcess devolve erro sob
// pressao de memoria ou de handles, e kill(pid, 0) devolve EPERM se as
// credenciais do pai mudarem. Tratar a primeira falha como morte encerra um
// servidor saudavel no meio de uma chamada — e o custo de esperar e apenas
// alguns ticks a mais para detectar uma morte real, que nao vai se desfazer.
const maxConsecutiveFailures = 3

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

		failures := 0

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				current, err := parentIdentity(pid)

				// Os dois casos de saida sao diferentes e merecem tratamento
				// diferente. Falha de consulta e ambigua: pode ser morte, pode
				// ser condicao transitoria. Identidade divergente nao e
				// ambigua: o PID existe e pertence a outro processo, o que so
				// acontece se o pai morreu e o PID foi reciclado.
				if err != nil {
					failures++
					if failures < maxConsecutiveFailures {
						continue
					}
					l.trigger("parent-gone")
					return
				}
				failures = 0

				if !sameProcess(initial, current) {
					l.trigger("parent-gone")
					return
				}
			}
		}
	}()
}
