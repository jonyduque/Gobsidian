package lifecycle

import (
	"context"
	"os"
	"os/signal"
	"sync"
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
	ch := adotarSinaisArmados()
	if ch == nil {
		ch = make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	}

	l.wg.Go(func() {
		defer signal.Stop(ch)

		select {
		case <-ch:
			l.trigger("signal")
		case <-ctx.Done():
		}
	})
}

// sinaisArmados e o canal que ArmarSinais registra, a espera do primeiro New.
var (
	muArmados     sync.Mutex
	sinaisArmados chan os.Signal
)

// ArmarSinais passa a capturar os sinais de encerramento JA, antes de existir
// qualquer Lifecycle, num canal com buffer que o primeiro New adota.
//
// # Por que existe
//
// O tratador so nascia dentro de New, e New roda em boot.VigiarHost -- depois
// de o processo montar o cobra, carregar a config e, em servePonte, discar e as
// vezes subir o daemon. Um CTRL_BREAK nessa janela nao tinha tratador e matava
// o processo pela acao padrao, sem registrar reason=. Medido no CI: o gate de
// orfaos reprovou em 4 de 15 execucoes desde 2026-09-16 com "1 de 100 ciclos
// encerraram sem registrar reason=". O conserto de 2026-09-09
// (check_partida) tirou o I/O de runServe e encolheu a janela; esta funcao a
// fecha, porque e chamada na primeira linha de main.
//
// So serve e daemon a chamam. Num comando de CLI, capturar o sinal tiraria do
// usuario o Ctrl+C que mata o processo.
//
// Idempotente.
func ArmarSinais() {
	muArmados.Lock()
	defer muArmados.Unlock()
	if sinaisArmados != nil {
		return
	}
	sinaisArmados = make(chan os.Signal, 1)
	signal.Notify(sinaisArmados, os.Interrupt, syscall.SIGTERM)
}

// adotarSinaisArmados entrega o canal armado ao primeiro Lifecycle e o tira do
// pacote: um segundo New -- o daemon e a ponte nunca criam dois, os testes
// criam muitos -- registra o proprio, e os dois nao disputam o mesmo sinal.
// Um sinal que chegou antes ja esta no buffer, e o select de watchSignals o
// recebe na hora.
func adotarSinaisArmados() chan os.Signal {
	muArmados.Lock()
	defer muArmados.Unlock()
	ch := sinaisArmados
	sinaisArmados = nil
	return ch
}
