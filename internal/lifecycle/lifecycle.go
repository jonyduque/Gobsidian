// Package lifecycle decide quando o processo deve morrer, e garante que ele
// morra. Tres mecanismos independentes cancelam o mesmo context raiz:
// EOF em stdin, sinais do sistema operacional e vigilia do processo pai.
// A redundancia e deliberada — cada um falha em cenarios diferentes.
package lifecycle

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Options configura os tres mecanismos de encerramento. Cada campo zerado
// desliga o mecanismo correspondente, o que existe para os testes poderem
// isolar um mecanismo de cada vez.
type Options struct {
	// Stdin e a fonte cujo EOF encerra o processo. os.Stdin em producao;
	// um io.Reader controlado nos testes.
	Stdin io.Reader

	// ParentPID e o processo cuja morte deve encerrar este. Zero desliga
	// a vigilia.
	ParentPID int

	// ParentCheckInterval e o intervalo entre consultas ao pai. Zero usa o
	// default do pacote.
	ParentCheckInterval time.Duration

	// Logger recebe os eventos de encerramento. Nulo vira um logger que
	// descarta — nunca stdout, que pertence ao JSON-RPC.
	Logger *slog.Logger
}

// Lifecycle coordena os mecanismos de encerramento e guarda qual deles
// disparou. So o primeiro a disparar fica registrado: os demais chegam
// depois, em cima de um context ja cancelado, e sobrescrever o motivo
// apagaria a informacao de diagnostico que interessa.
type Lifecycle struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu     sync.Mutex
	reason string

	log *slog.Logger
}

// New devolve um context que e cancelado quando qualquer um dos mecanismos
// dispara, e o Lifecycle que os coordena.
func New(parent context.Context, opts Options) (context.Context, *Lifecycle) {
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if opts.ParentCheckInterval == 0 {
		opts.ParentCheckInterval = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(parent)
	lc := &Lifecycle{cancel: cancel, log: opts.Logger}

	if opts.Stdin != nil {
		lc.watchStdin(ctx, opts.Stdin)
	}
	lc.watchSignals(ctx)
	if opts.ParentPID > 0 {
		lc.watchParent(ctx, opts.ParentPID, opts.ParentCheckInterval)
	}

	return ctx, lc
}

// trigger cancela o context, registrando o primeiro motivo que chegou.
// Chamadas subsequentes nao sobrescrevem o motivo: o que importa e quem
// disparou primeiro.
func (l *Lifecycle) trigger(reason string) {
	l.mu.Lock()
	if l.reason == "" {
		l.reason = reason
		l.log.Info("encerramento solicitado", "reason", reason)
	}
	l.mu.Unlock()
	l.cancel()
}

// Reason devolve o mecanismo que pediu o encerramento — "stdin-eof",
// "signal" ou "parent-gone" — ou string vazia se nada pediu ainda. E o que
// o harness de orfaos le nos logs para provar que algum mecanismo disparou:
// zero orfaos sem motivo registrado nao prova nada.
func (l *Lifecycle) Reason() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reason
}

// Wait bloqueia ate todas as goroutines do lifecycle que podem ser
// desenroladas terminarem — hoje o handler de sinais (Task 4) e a vigilia
// do processo pai (Task 5). Deliberadamente NAO espera pela goroutine de
// stdin: uma leitura bloqueada em io.Reader nao tem saida por cancelamento
// de context, entao inclui-la no WaitGroup faria Wait() travar sempre que
// outro mecanismo disparasse o desligamento primeiro e stdin permanecesse
// aberto. Ver o comentario de watchStdin em stdin.go.
func (l *Lifecycle) Wait() { l.wg.Wait() }

// ParentPID devolve o PID do processo pai no momento da chamada.
func ParentPID() int { return os.Getppid() }
