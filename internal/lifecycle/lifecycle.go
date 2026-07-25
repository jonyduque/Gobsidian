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

type Options struct {
	// Stdin e a fonte cujo EOF encerra o processo. os.Stdin em producao;
	// um io.Reader controlado nos testes.
	Stdin io.Reader

	// ParentPID e o processo cuja morte deve encerrar este. Zero desliga
	// a vigilia.
	ParentPID int

	ParentCheckInterval time.Duration

	Logger *slog.Logger
}

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

func (l *Lifecycle) Reason() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reason
}

// Wait bloqueia ate todas as goroutines do lifecycle terminarem.
func (l *Lifecycle) Wait() { l.wg.Wait() }

// ParentPID devolve o PID do processo pai no momento da chamada.
func ParentPID() int { return os.Getppid() }
