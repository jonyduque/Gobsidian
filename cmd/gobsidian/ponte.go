// ponte.go implementa o processo-ponte: tenta falar com um daemon de cofre
// local via socket Unix (internal/ipc) e, se nao conseguir, serve o cofre
// neste mesmo processo — o caminho que o servidor sempre usou, em
// serveEmProcesso (serve.go).
//
// A ponte e burra (decisao 1 da Task 91): quando conectada a um daemon, ela
// so copia bytes entre o stdio que o host lhe deu e o socket. Nao interpreta
// JSON-RPC, nao tem indice, nao tem estado — e o que a mantem em poucos MB.
// O daemon do outro lado do socket e a Task 92; ate ela existir, o socket
// nunca esta la, e este arquivo so exercita o caminho de fallback em
// producao.
package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/ipc"
	"github.com/jonyd/gobsidian/internal/lifecycle"
)

// ipcDialTimeout limita quanto tempo a ponte espera por um daemon antes de
// desistir e cair no modo em processo. Curto de proposito: D-M7-6 mediu
// 25,7 us de ida e volta para um socket Unix local respondendo — o tempo
// aqui e so para nao travar quando o arquivo do socket existe mas ninguem
// mais esta escutando nele (daemon morto sem limpar).
const ipcDialTimeout = 300 * time.Millisecond

// servePonte decide entre falar com um daemon via socket ou servir o cofre
// neste processo.
//
// O fallback e obrigatorio (decisao 2 da Task 91): DialAndHandshake falha
// dos mesmos tres jeitos — socket ausente, conexao recusada, versao
// incompativel — e os tres levam ao mesmo lugar. Sem isso um socket quebrado
// transformaria a ferramenta em nada, e quem a chama nao teria como
// diagnosticar; por isso o log abaixo sempre registra o motivo da queda.
func servePonte(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	conn, err := ipc.DialAndHandshake(ctx, cfg.VaultPath, ipcDialTimeout)
	if err != nil {
		log.Info("socket do daemon indisponivel; servindo em processo", "err", err)
		return serveEmProcesso(ctx, cfg, log)
	}
	log.Info("conectado ao daemon via socket")
	return servePonteRemota(ctx, conn, log)
}

// servePonteRemota copia bytes entre o stdio deste processo e o socket do
// daemon, ate um dos dois lados fechar ou um dos tres mecanismos de
// encerramento (EOF em stdin, sinal do SO, morte do processo pai) disparar.
// Ela nao interpreta o que passa por dentro.
//
// Os tres mecanismos continuam valendo aqui exatamente como em
// serveEmProcesso: lifecycle.New e o mesmo pacote, montado do mesmo jeito,
// porque a garantia de nao deixar orfao nao pode depender de qual dos dois
// caminhos serviu a sessao.
func servePonteRemota(parent context.Context, conn ipc.Conn, log *slog.Logger) error {
	// O monitor de stdin do lifecycle consome bytes, e o stdin aqui pertence
	// ao daemon do outro lado do socket. A saida e espelhar: a copia de
	// verdade le do espelho, e o lifecycle observa so a copia. io.TeeReader
	// nao serve — nao propaga EOF, copia bytes e EOF nao e byte — por isso
	// mirrorReader, que faz dst.CloseWithError(err) (ver serve.go).
	pr, pw := io.Pipe()
	teed := &mirrorReader{src: os.Stdin, dst: pw}

	ctx, lc := lifecycle.New(parent, lifecycle.Options{
		Stdin:     pr,
		ParentPID: lifecycle.ParentPID(),
		Logger:    log,
	})

	// As duas direcoes da copia sao goroutines independentes, e nenhuma
	// delas entra no WaitGroup do lifecycle: uma goroutine parada em Read
	// nao e desenrolavel por cancelamento de context (a mesma razao pela
	// qual watchStdin fica fora do WaitGroup em internal/lifecycle), entao
	// so o fechamento explicito de pw e conn nos passos de shutdown abaixo
	// as desbloqueia.
	hostParaDaemon := make(chan error, 1)
	go func() {
		_, err := io.Copy(conn, teed)
		hostParaDaemon <- err
	}()

	daemonParaHost := make(chan error, 1)
	go func() {
		_, err := io.Copy(os.Stdout, conn)
		daemonParaHost <- err
	}()

	var loopErr error
	select {
	case err := <-daemonParaHost:
		// O daemon fechou a conexao (ou ela quebrou): e o equivalente, na
		// ponte, ao serve loop de serveEmProcesso retornando.
		loopErr = err
	case err := <-hostParaDaemon:
		loopErr = err
	case <-ctx.Done():
	}

	lifecycle.Shutdown(ctx, log, 6*time.Second,
		lifecycle.Step{Name: "close-pipe", Budget: 500 * time.Millisecond, Fn: func(context.Context) error {
			return pw.Close()
		}},
		lifecycle.Step{Name: "close-conn", Budget: 500 * time.Millisecond, Fn: func(context.Context) error {
			return conn.Close()
		}},
	)

	lc.Wait()

	// ctx.Canceled no retorno do loop de copia e encerramento normal — a
	// mesma regra que vale para o serve loop de serveEmProcesso (ver
	// shutdownExitCode). Os tres mecanismos cancelam o mesmo context, e uma
	// copia interrompida por isso nao e falha; io.EOF/io.ErrClosedPipe/
	// os.ErrClosed sao as formas como o fechamento de pw ou conn aparecem
	// nos dois lados da copia.
	if loopErr != nil &&
		!errors.Is(loopErr, context.Canceled) &&
		!errors.Is(loopErr, io.EOF) &&
		!errors.Is(loopErr, io.ErrClosedPipe) &&
		!errors.Is(loopErr, os.ErrClosed) {
		return loopErr
	}
	return nil
}
