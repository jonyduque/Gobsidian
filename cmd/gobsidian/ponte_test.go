package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/ipc"
	"github.com/jonyduque/Gobsidian/internal/vaulttest"
)

// TestPonteCaiParaModoEmProcesso e o teste nomeado na prova de mutacao da
// Task 91: se a chamada "return serveEmProcesso(ctx, cfg, log)" em
// servePonte for trocada por "return err", este teste tem de reprovar.
//
// A distincao nao depende de levantar um servidor MCP de verdade -- isso
// bloquearia em os.Stdin, que o processo de teste nao controla. Em vez
// disso, o cofre e um caminho que nao existe: serveEmProcesso falha DENTRO
// dele, no primeiro passo (vault.New), antes de tocar em stdin nenhum, com
// uma mensagem que so aparece se o fallback de fato chamou serveEmProcesso.
// A mutacao devolveria o erro da conexao recusada em vez disso -- um erro
// com uma mensagem completamente diferente -- e este teste distingue os
// dois.

// semDaemonParaTeste desliga a tentativa de iniciar um daemon de verdade
// nos testes deste arquivo: sob "go test", os.Executable() devolveria o
// BINARIO DE TESTE, e deixar EnsureStarted (chamada por servePonte) lanca-lo
// com argumentos de subcomando lancaria um processo imprevisivel em vez de
// nada. Encolhe daemonStartTimeout tambem, senao EnsureStarted insiste por
// segundos tentando um socket que este teste nunca deixa existir.
func semDaemonParaTeste(t *testing.T) {
	t.Helper()
	origIniciar := iniciarDaemonFn
	origTimeout := daemonStartTimeout
	iniciarDaemonFn = func(config.Config) error { return nil }
	daemonStartTimeout = 60 * time.Millisecond
	t.Cleanup(func() {
		iniciarDaemonFn = origIniciar
		daemonStartTimeout = origTimeout
	})
}

func TestPonteCaiParaModoEmProcesso(t *testing.T) {
	semDaemonParaTeste(t)

	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, nil))

	vaultInexistente := filepath.Join(t.TempDir(), "nao-existe")
	cfg := config.Config{VaultPath: vaultInexistente}

	// Nenhum socket existe para este cofre (nenhum daemon rodou nunca para
	// um diretorio temporario recem-criado), entao DialAndHandshake falha
	// de verdade -- sem precisar de um dublê. iniciarDaemonFn acima nao faz
	// nada, entao EnsureStarted tambem falha em esperarSocket, e o
	// fallback obrigatorio e alcancado do mesmo jeito que antes da Task 92.
	ctx, cancel := context.WithTimeout(context.Background(), vaulttest.Prazo)
	defer cancel()

	err := servePonte(ctx, cfg, log)
	if err == nil {
		t.Fatal("servePonte() error = nil, esperado erro (cofre inexistente)")
	}
	if !strings.Contains(err.Error(), "raiz do cofre inacessivel") {
		t.Fatalf("servePonte() error = %q, esperado mencionar %q -- essa mensagem so vem de dentro de serveEmProcesso -> vault.New; "+
			"se o texto for outro (por exemplo, o erro de conectar ao socket), o fallback nao foi chamado",
			err, "raiz do cofre inacessivel")
	}

	if !strings.Contains(logBuf.String(), "servindo em processo") {
		t.Fatalf("log nao registrou a queda para o modo em processo: %s", logBuf.String())
	}
}

// TestServePonteRespeitaGobsidianNoDaemon prova que GOBSIDIAN_NO_DAEMON pula
// a decisao inteira -- nem tenta discar um daemon, nem tenta iniciar um. Um
// socket presente e respondendo (o oposto do caso "ausente") teria que ser
// IGNORADO com a variavel definida; sem essa prova, um "-> servindo em
// processo" no log nao distingue "pulou por causa da variavel" de "tentou e
// falhou por outro motivo".
func TestServePonteRespeitaGobsidianNoDaemon(t *testing.T) {
	t.Setenv("GOBSIDIAN_NO_DAEMON", "1")

	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, nil))

	vaultInexistente := filepath.Join(t.TempDir(), "nao-existe")
	cfg := config.Config{VaultPath: vaultInexistente}

	// Um socket de verdade, respondendo com a saudacao certa -- se a
	// variavel nao fosse respeitada, este teste conectaria e
	// servePonteRemota ficaria presa em os.Stdin, travando o teste.
	ln, _, err := ipc.Listen(cfg.VaultPath)
	if err != nil {
		t.Fatalf("ipc.Listen() error = %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		_ = ipc.Greet(c, ipc.HandshakeConfig{VaultKey: config.VaultKey(cfg.VaultPath)})
		time.Sleep(200 * time.Millisecond)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), vaulttest.Prazo)
	defer cancel()

	err = servePonte(ctx, cfg, log)
	if err == nil {
		t.Fatal("servePonte() error = nil, esperado erro (cofre inexistente, modo em processo forcado)")
	}
	if !strings.Contains(err.Error(), "raiz do cofre inacessivel") {
		t.Fatalf("servePonte() error = %q, esperado mencionar %q -- essa mensagem so vem de serveEmProcesso -> vault.New; "+
			"se o texto for outro, GOBSIDIAN_NO_DAEMON nao pulou a tentativa de socket",
			err, "raiz do cofre inacessivel")
	}
	if !strings.Contains(logBuf.String(), "GOBSIDIAN_NO_DAEMON") {
		t.Fatalf("log nao registrou que GOBSIDIAN_NO_DAEMON forcou o modo em processo: %s", logBuf.String())
	}
}

// TestServePonteVersaoDiferenteCaiParaProcesso cobre a segunda verificacao
// que o brief da Task 91 pede: um socket presente, mas de uma versao que a
// ponte nao fala, tem de cair para o mesmo lugar que um socket ausente --
// sem precisar da Task 92 (o daemon de verdade) para existir, simulando o
// lado do servidor com um listener de teste.
func TestServePonteVersaoDiferenteCaiParaProcesso(t *testing.T) {
	semDaemonParaTeste(t)

	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, nil))

	vaultInexistente := filepath.Join(t.TempDir(), "nao-existe")
	cfg := config.Config{VaultPath: vaultInexistente}

	ln, _, err := ipc.Listen(cfg.VaultPath)
	if err != nil {
		t.Fatalf("ipc.Listen() error = %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		_, _ = fmt.Fprint(c, "GOBSIDIAN-IPC 999\n")
		time.Sleep(200 * time.Millisecond)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), vaulttest.Prazo)
	defer cancel()

	err = servePonte(ctx, cfg, log)
	if err == nil {
		t.Fatal("servePonte() error = nil, esperado erro (cofre inexistente, apos o fallback)")
	}
	if !strings.Contains(err.Error(), "raiz do cofre inacessivel") {
		t.Fatalf("servePonte() error = %q, esperado mencionar %q (fallback nao alcancado)", err, "raiz do cofre inacessivel")
	}
	if !strings.Contains(logBuf.String(), "servindo em processo") {
		t.Fatalf("log nao registrou a queda para o modo em processo: %s", logBuf.String())
	}
	// Versao divergente NAO e transitoria: ela se repete em toda partida ate
	// alguem reinstalar, e e o unico caso em que o daemon do outro lado esta
	// vivo e saudavel. Ate 2026-09-08 ela dava a MESMA linha que "o daemon nao
	// subiu", e os dois pedem consertos diferentes.
	if !strings.Contains(logBuf.String(), "motivo=versao-divergente") {
		t.Fatalf("a queda por versao divergente nao se distingue das outras no log: %s", logBuf.String())
	}
}

// TestQuedaParaModoEmProcessoEhWarnComMotivo cobre o silencio medido em
// 2026-09-08 na maquina do dono: o PID 42628 serviu o cofre Estudo em modo
// degradado por 20 h, com watcher e indice proprios, gravando no MESMO
// inverted_cache.gob que o daemon -- e a unica pista era uma linha INFO,
// indistinguivel em gravidade do caminho bom.
//
// docs/OPERACAO.md ja registra que essa classe de silencio custou um marco
// inteiro desligado em producao sem ninguem perceber. WARN e o minimo: quem
// filtra por gravidade precisa conseguir separar "estou no caminho bom" de
// "cai para o caminho ruim".
func TestQuedaParaModoEmProcessoEhWarnComMotivo(t *testing.T) {
	semDaemonParaTeste(t)

	var logBuf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logBuf, nil))

	cfg := config.Config{VaultPath: filepath.Join(t.TempDir(), "nao-existe")}

	ctx, cancel := context.WithTimeout(context.Background(), vaulttest.Prazo)
	defer cancel()
	_ = servePonte(ctx, cfg, log)

	saida := logBuf.String()
	if !strings.Contains(saida, "level=WARN") {
		t.Errorf("a queda para o modo em processo saiu sem WARN, entao ela nao se distingue do caminho bom:\n%s", saida)
	}
	if !strings.Contains(saida, "motivo=daemon-nao-subiu") {
		t.Errorf("a queda saiu sem motivo=daemon-nao-subiu:\n%s", saida)
	}
}

// TestServePonteRemotaFazProxyDeBytes prova que, quando o handshake da
// certo, a ponte de fato copia bytes do daemon para o stdout do host -- e
// que fechar a conexao do lado do daemon termina servePonteRemota
// normalmente. Usa duplexPipe (io.Pipe, nao net.Pipe) de proposito:
// cmd/gobsidian nao importa "net" em lugar nenhum, e este teste nao e
// excecao.
func TestServePonteRemotaFazProxyDeBytes(t *testing.T) {
	conn, outroLado := newDuplexPipe()
	t.Cleanup(func() { _ = conn.Close() })

	// O stdin do host e um pipe que este teste mantem ABERTO. Usar o
	// os.Stdin do processo tornava este teste dependente do sistema: sob
	// `go test` no Linux ele e /dev/null e devolve EOF imediatamente, a
	// ponte encerrava por stdin-eof antes da escrita abaixo, e o teste
	// falhava com "read/write on closed pipe" -- verde no Windows e no
	// macOS, vermelho no ubuntu, no mesmo commit. Manter o pipe aberto faz
	// a unica coisa que termina a ponte ser o fechamento do lado do daemon,
	// que e justamente o que este teste quer provar.
	stdinHost, _ := io.Pipe()
	t.Cleanup(func() { _ = stdinHost.Close() })

	// stdout do host num buffer proprio, para poder conferir os BYTES que
	// atravessaram. E o unico jeito de afirmar que a ponte copia: a versao
	// anterior escrevia no os.Stdout do processo de teste e so conseguia
	// dizer que a leitura nao travava.
	//
	// Com mutex, e nao um bytes.Buffer cru: servePonteRemota pode retornar
	// pelo lado do stdin enquanto a goroutine que copia daemon->host ainda
	// escreve, e ai a leitura do buffer no fim compete com essa escrita. Em
	// producao o destino e os.Stdout e o processo esta encerrando, entao nao
	// e defeito de produto -- mas aqui e uma corrida de verdade, e o -race a
	// acusa. Medido: com um stdin que devolve EOF na hora, "race detected".
	stdoutHost := &escritorSeguro{}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	done := make(chan error, 1)
	go func() {
		done <- servePonteRemota(context.Background(), t.TempDir(), conn, stdinHost, stdoutHost, log)
	}()

	// outroLado faz o papel do daemon: o que ele escreve tem que sair no
	// stdout do host sem alteracao, e fecha-lo tem que terminar
	// servePonteRemota -- o equivalente, na ponte, ao serve loop de
	// serveEmProcesso retornando quando o host desconecta.
	const resposta = "resposta do daemon"
	if _, err := outroLado.Write([]byte(resposta)); err != nil {
		t.Fatalf("escrevendo do lado do daemon: %v", err)
	}
	if err := outroLado.Close(); err != nil {
		t.Fatalf("fechando o lado do daemon: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("servePonteRemota() error = %v, esperado nil (fechamento normal)", err)
		}
	case <-time.After(vaulttest.Prazo):
		t.Fatal("servePonteRemota nao retornou apos o daemon fechar a conexao")
	}

	if got := stdoutHost.String(); got != resposta {
		t.Errorf("stdout do host = %q, quer %q -- a ponte nao copiou os bytes do daemon", got, resposta)
	}
}

// TestServePonteRemotaEncaminhaHostParaDaemon e o contrapeso de
// TestServePonteRemotaFazProxyDeBytes, que so exercita a direcao
// daemon->host: la o stdin do host e um pipe cujo escritor e DESCARTADO, e
// por isso nada nunca sobe do host para o daemon. Com so aquele teste,
// apagar io.Copy(conn, teed) e o passo half-close inteiro deixa a suite
// verde.
//
// Aqui o escritor do stdin e guardado, e o teste afirma as duas metades que
// faltavam:
//
//  1. o que o host escreve chega ao daemon byte a byte;
//  2. fechado o stdin do host, o daemon ve EOF e AINDA CONSEGUE responder --
//     que e o meio-fechamento do achado M8. Afirmar so o EOF nao bastaria:
//     sem o half-close o passo close-conn fecha a conexao inteira e o daemon
//     ve EOF do mesmo jeito. O que distingue os dois e a resposta em voo: com
//     o meio-fechamento ela chega ao stdout do host, sem ele a escrita do
//     daemon morre em pipe fechado.
func TestServePonteRemotaEncaminhaHostParaDaemon(t *testing.T) {
	conn, outroLado := newDuplexPipe()
	t.Cleanup(func() { _ = conn.Close() })

	stdinHost, hostEscreve := io.Pipe()
	t.Cleanup(func() { _ = stdinHost.Close() })

	stdoutHost := &escritorSeguro{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	done := make(chan error, 1)
	go func() {
		done <- servePonteRemota(context.Background(), t.TempDir(), conn, stdinHost, stdoutHost, log)
	}()

	// O lado do daemon le tudo o que a ponte encaminhar, anuncia o pedido
	// quando ele chega e anuncia o fim da leitura. Duas notificacoes
	// separadas porque o teste precisa distinguir "nao copiou" de "copiou e
	// nao propagou o EOF".
	const pedido = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n"
	pedidoChegou := make(chan string, 1)
	leituraAcabou := make(chan error, 1)
	go func() {
		var recebido []byte
		buf := make([]byte, 512)
		anunciado := false
		for {
			n, err := outroLado.Read(buf)
			recebido = append(recebido, buf[:n]...)
			if !anunciado && strings.Contains(string(recebido), `"method":"initialize"`) {
				anunciado = true
				pedidoChegou <- string(recebido)
			}
			if err != nil {
				leituraAcabou <- err
				return
			}
		}
	}()

	if _, err := hostEscreve.Write([]byte(pedido)); err != nil {
		t.Fatalf("host escrevendo no proprio stdin: %v", err)
	}

	select {
	case got := <-pedidoChegou:
		if got != pedido {
			t.Errorf("o daemon recebeu %q, quer %q -- a ponte alterou os bytes no caminho", got, pedido)
		}
	case <-time.After(vaulttest.Prazo):
		t.Fatal("o daemon nao recebeu nada em " + vaulttest.Prazo.String() +
			" -- a direcao host->daemon nao esta sendo copiada")
	}

	if err := hostEscreve.Close(); err != nil {
		t.Fatalf("fechando o stdin do host: %v", err)
	}

	select {
	case err := <-leituraAcabou:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("o daemon terminou a leitura com %v, quer io.EOF -- o fim do stdin do host tem de chegar como fim de arquivo, nao como conexao quebrada", err)
		}
	case <-time.After(vaulttest.Prazo):
		t.Fatal("o daemon nao viu o fim do stdin do host em " + vaulttest.Prazo.String())
	}

	// A resposta que estava em voo quando o host fechou o stdin. E ela que
	// separa o meio-fechamento do fechamento inteiro.
	//
	// Se esta escrita falhar, o suspeito nao e so uma regressao do M8: ha
	// uma corrida do proprio produto em ponte.go:193-202, onde o select
	// entre hostParaDaemon e ctx.Done() pode resolver para ctx.Done() antes
	// que o EOF do stdin do host seja lido como hostParaDaemon -- nesse
	// caso fimDoHost fica false e o shutdown nao faz o meio-fechamento que
	// este teste espera. Ver tambem o comentario em fimDoHost, acima.
	const tardia = `{"jsonrpc":"2.0","id":1,"result":{}}` + "\n"
	if _, err := outroLado.Write([]byte(tardia)); err != nil {
		t.Fatalf("o daemon nao conseguiu responder depois do EOF: %v -- a ponte fechou a conexao INTEIRA em vez de so a direcao de escrita (achado M8)", err)
	}
	if err := outroLado.CloseWrite(); err != nil {
		t.Fatalf("fechando a escrita do lado do daemon: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("servePonteRemota() error = %v, esperado nil (encerramento normal por EOF do host)", err)
		}
	case <-time.After(vaulttest.Prazo):
		t.Fatal("servePonteRemota nao retornou apos o EOF do host e o fechamento do daemon")
	}

	if got := stdoutHost.String(); got != tardia {
		t.Errorf("stdout do host = %q, quer %q -- a resposta em voo apos o EOF do host se perdeu", got, tardia)
	}
}

// duplexPipe implementa ipc.Conn (Read, Write, Close, CloseWrite) em cima de
// dois io.Pipe -- um por sentido -- para os testes deste pacote poderem
// simular o lado do daemon sem importar "net". CloseWrite fecha so a
// escrita, como o net.UnixConn de verdade faria: o outro lado ainda pode
// mandar o resto do que tinha para dizer antes de a leitura tambem acabar.
type duplexPipe struct {
	r      *io.PipeReader
	w      *io.PipeWriter
	closeR func() error
}

func (d duplexPipe) Read(p []byte) (int, error)  { return d.r.Read(p) }
func (d duplexPipe) Write(p []byte) (int, error) { return d.w.Write(p) }
func (d duplexPipe) CloseWrite() error           { return d.w.Close() }
func (d duplexPipe) Close() error {
	_ = d.w.Close()
	return d.closeR()
}

// newDuplexPipe devolve os dois lados de um par ligado: o que "a" escreve,
// "b" le, e vice-versa.
func newDuplexPipe() (a, b duplexPipe) {
	ar, aw := io.Pipe() // a escreve aqui, b le
	br, bw := io.Pipe() // b escreve aqui, a le
	a = duplexPipe{r: br, w: aw, closeR: br.Close}
	b = duplexPipe{r: ar, w: bw, closeR: ar.Close}
	return a, b
}

// escritorSeguro e um buffer protegido por mutex. Ver o comentario em
// TestServePonteRemotaFazProxyDeBytes: sem ele o -race acusa competicao entre
// a goroutine de copia da ponte e a leitura do buffer no fim do teste.
type escritorSeguro struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (e *escritorSeguro) Write(p []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.buf.Write(p)
}

func (e *escritorSeguro) String() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.buf.String()
}
