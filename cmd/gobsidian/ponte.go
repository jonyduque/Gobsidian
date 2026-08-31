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
	"syscall"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/daemon"
	"github.com/jonyd/gobsidian/internal/ipc"
	"github.com/jonyd/gobsidian/internal/lifecycle"
)

// ipcDialTimeout limita quanto tempo a ponte espera por um daemon antes de
// desistir e cair no modo em processo. Curto de proposito: D-M7-6 mediu
// 25,7 us de ida e volta para um socket Unix local respondendo — o tempo
// aqui e so para nao travar quando o arquivo do socket existe mas ninguem
// mais esta escutando nele (daemon morto sem limpar).
const ipcDialTimeout = 300 * time.Millisecond

// daemonStartTimeout limita quanto tempo a ponte espera o daemon recem-
// pedido abrir o socket -- o que inclui o tempo de montar o indice de
// metadados (RNF-01 mediu ~900ms num cofre de 109 MB via varredura, bem
// abaixo deste orcamento). Generoso o suficiente para o boot real, mas
// finito: um daemon que nunca sobe nao pode travar a ponte para sempre --
// o fallback em processo continua valendo (ver o comentario de servePonte).
//
// Variavel, nao const: os testes deste pacote encolhem este numero para nao
// esperar segundos por um daemon que o teste nunca deixa subir de verdade.
var daemonStartTimeout = 10 * time.Second

// iniciarDaemonFn e a implementacao real de "iniciar" que EnsureStarted
// chama quando esta ponte vence a corrida (internal/daemon.EnsureStarted,
// decisao 2 da Task 92). Indireto via variavel de pacote para os testes
// poderem substituir o lancamento de um processo real por um dublê: sob
// "go test", os.Executable() devolve o BINARIO DE TESTE, e deixar
// EnsureStarted chama-lo com argumentos de subcomando lancaria um processo
// imprevisivel em vez de nada.
var iniciarDaemonFn = func(cfg config.Config) error {
	return daemon.SpawnDetached(cfg, daemon.DefaultIdleSeconds, cfg.LogLevel.String())
}

// servePonte decide entre falar com um daemon via socket, iniciar um se
// nenhum estiver rodando, ou servir o cofre neste processo.
//
// O fallback em processo e obrigatorio em QUALQUER dos tres pontos onde
// pode ser alcancado (decisao 2 da Task 91, mantida pela Task 92): socket
// ausente na primeira tentativa, EnsureStarted incapaz de iniciar ou de
// confirmar que o daemon subiu, ou o handshake apos iniciar ainda assim
// falhando (por exemplo, corrida perdida contra outro processo que
// removeu o socket). Sem isso um daemon quebrado transformaria a
// ferramenta em nada, e quem a chama nao teria como diagnosticar; por
// isso o log abaixo sempre registra o motivo de cada queda.
//
// GOBSIDIAN_NO_DAEMON, se definida com qualquer valor nao vazio, pula
// TODA esta decisao e vai direto para o modo em processo -- nem tenta
// discar, nem tenta iniciar. Dois usos: medir o "antes" da Task 92 sem
// precisar reverter codigo (a Parte II inteira existe para mover esse
// numero, e comparar exige um jeito de desligar o lado novo sem recompilar
// um binario velho), e dar a quem opera o servidor um jeito de desligar o
// daemon numa maquina onde ele nao e desejado (documentado no README).
func servePonte(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	if os.Getenv("GOBSIDIAN_NO_DAEMON") != "" {
		log.Info("GOBSIDIAN_NO_DAEMON definida; servindo em processo sem tentar o daemon")
		return serveEmProcesso(ctx, cfg, log)
	}

	conn, err := ipc.DialAndHandshake(ctx, cfg.VaultPath, cfg.ReadOnly, cfg.MaxResults, ipcDialTimeout)
	if err == nil {
		log.Info("conectado ao daemon via socket")
		return servePonteRemota(ctx, conn, os.Stdin, os.Stdout, log)
	}
	log.Info("socket do daemon indisponivel; tentando iniciar o daemon",
		"err", err, "errno", errnoDe(err))

	iniciar := func() error { return iniciarDaemonFn(cfg) }
	if startErr := daemon.EnsureStarted(ctx, cfg, daemonStartTimeout, iniciar); startErr != nil {
		log.Info("nao foi possivel iniciar o daemon; servindo em processo",
			"err", startErr, "errno", errnoDe(startErr))
		return serveEmProcesso(ctx, cfg, log)
	}

	conn, err = ipc.DialAndHandshake(ctx, cfg.VaultPath, cfg.ReadOnly, cfg.MaxResults, ipcDialTimeout)
	if err != nil {
		log.Info("daemon nao respondeu apos iniciar; servindo em processo",
			"err", err, "errno", errnoDe(err))
		return serveEmProcesso(ctx, cfg, log)
	}
	log.Info("conectado ao daemon recem-iniciado via socket")
	return servePonteRemota(ctx, conn, os.Stdin, os.Stdout, log)
}

// errnoDe extrai o numero do erro de sistema, ou -1 quando nao houver um.
//
// Existe porque a prosa do Windows nao distingue casos que se comportam de
// forma diferente. Medido em 2026-08-26 com net.Dial("unix", ...):
//
//	10061  ECONNREFUSED   arquivo comum, socket orfao de dono morto a forca,
//	                      E caminho inexistente -- os tres iguais
//	10022  EINVAL         diretorio no lugar do socket
//
// A distincao decide se ha daemon vivo ou lixo no caminho, e nos logs de campo
// do dono apareceu o 10022, que nenhum dos reprodutores conhecidos explica.
// Sem o numero foi preciso reverter a mensagem para descobrir isso; com ele, a
// proxima ocorrencia se explica sozinha.
//
// NAO use este numero para decidir se o daemon esta vivo. O criterio e
// comportamental -- handshake completo -- porque a taxonomia acima mostra que
// errnos diferentes descrevem o mesmo estado e o mesmo errno descreve estados
// diferentes.
func errnoDe(err error) int {
	var se syscall.Errno
	if errors.As(err, &se) {
		return int(se)
	}
	return -1
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
//
// stdin e stdout sao parametros, e nao os.Stdin/os.Stdout lidos aqui dentro,
// porque sem isso esta funcao nao e testavel de forma deterministica: sob
// `go test` no Linux o stdin do processo e /dev/null e devolve EOF na hora,
// entao a ponte encerrava antes de o teste conseguir escrever do lado do
// daemon, e o teste falhava com "read/write on closed pipe" -- so no Linux,
// porque no Windows o stdin de teste bloqueia. Medido: verde no Windows e no
// macOS, vermelho no ubuntu, no MESMO commit.
//
// O ganho e maior que a estabilidade: com stdout injetavel, o teste passa a
// conferir os BYTES que atravessaram a ponte, que e o que ela existe para
// fazer. Antes ele so conseguia afirmar que a leitura nao travava.
func servePonteRemota(parent context.Context, conn ipc.Conn, stdin io.Reader, stdout io.Writer, log *slog.Logger) error {
	// O monitor de stdin do lifecycle consome bytes, e o stdin aqui pertence
	// ao daemon do outro lado do socket. A saida e espelhar: a copia de
	// verdade le do espelho, e o lifecycle observa so a copia. io.TeeReader
	// nao serve — nao propaga EOF, copia bytes e EOF nao e byte — por isso
	// mirrorReader, que faz dst.CloseWithError(err) (ver serve.go).
	pr, pw := io.Pipe()
	teed := &mirrorReader{src: stdin, dst: pw}

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
		_, err := io.Copy(stdout, conn)
		daemonParaHost <- err
	}()

	var loopErr error
	// fimDoHost diz que a copia host->daemon terminou — tipicamente EOF do
	// stdin, que e como o host MCP encerra. So nesse caso o meio-fechamento
	// tem sentido: se quem caiu foi o daemon, nao ha resposta para esperar.
	fimDoHost := false
	select {
	case err := <-daemonParaHost:
		// O daemon fechou a conexao (ou ela quebrou): e o equivalente, na
		// ponte, ao serve loop de serveEmProcesso retornando.
		loopErr = err
	case err := <-hostParaDaemon:
		loopErr = err
		fimDoHost = true
	case <-ctx.Done():
	}

	lifecycle.Shutdown(ctx, log, 6*time.Second,
		lifecycle.Step{Name: "close-pipe", Budget: 500 * time.Millisecond, Fn: func(context.Context) error {
			return pw.Close()
		}},
		// MEIO-FECHAMENTO antes de fechar a conexao inteira.
		//
		// `ipc.Conn` exige `CloseWrite` desde sempre — `DialAndHandshake`
		// RECUSA uma conexao que nao o suporte — e nada nunca o chamava
		// (achado M8). No EOF do stdin a ponte fechava a conexao inteira, e
		// toda resposta do daemon ainda em voo era descartada: exatamente o que
		// o meio-fechamento existe para evitar.
		//
		// Fechar so a direcao de escrita sinaliza EOF ao daemon; ele termina o
		// que estava respondendo e fecha do lado dele, o que encerra a copia
		// daemon->host naturalmente. O orcamento existe porque um daemon travado
		// nao pode segurar o encerramento da ponte para sempre.
		lifecycle.Step{Name: "half-close", Budget: 2 * time.Second, Fn: func(sctx context.Context) error {
			if !fimDoHost {
				return nil
			}
			if err := conn.CloseWrite(); err != nil {
				return err
			}
			select {
			case <-daemonParaHost:
				return nil
			case <-sctx.Done():
				// Orcamento estourado: o close-conn abaixo corta o que sobrou.
				// Perder uma resposta aqui e visivel no log, e nao silencioso.
				return sctx.Err()
			}
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
