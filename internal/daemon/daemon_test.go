package daemon_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/daemon"
	"github.com/jonyd/gobsidian/internal/ipc"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// boundedWait e o prazo usado para esperar por algo neste pacote. Um
// defeito real (por exemplo, a condicao de ociosidade nunca disparando) nao
// pode travar "go test -race ./..." ate o timeout padrao de 10 minutos.
const boundedWait = 5 * time.Second

// newTestDaemon monta um *daemon.Daemon completo: um *mcpsrv.Server minimo
// (o mesmo padrao de internal/mcpsrv/server_test.go — service.New aceita
// idx/inv/watchStats nil) sobre um socket Unix real aberto via ipc.Listen,
// nunca net.Listen("tcp", ...) direto: tools/netcheck varre os testes deste
// pacote tambem, e so aceita net.Dial/net.Listen com a constante "unix".
// Devolve tambem o caminho usado como chave do socket, para o teste poder
// discar de volta.
func newTestDaemon(t *testing.T, ociosidade time.Duration) (*daemon.Daemon, string) {
	t.Helper()

	root := t.TempDir()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	svc := service.New(v, nil, nil, nil, service.Options{})
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := mcpsrv.New(context.Background(), svc, config.Defaults(), log)

	vaultDir := t.TempDir() // chave do socket -- distinta de root, so precisa ser estavel
	ln, _, err := ipc.Listen(vaultDir)
	if err != nil {
		t.Fatalf("ipc.Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	cfg := daemon.Config{
		Vault:         config.Config{VaultPath: vaultDir, ReadOnly: false},
		OciosidadeMax: ociosidade,
	}
	return daemon.New(ln, srv, cfg, log), vaultDir
}

// nopWriteCloser espelha o mesmo tipo interno de internal/mcpsrv/server.go:
// a biblioteca padrao tem io.NopCloser para leitura e nao tem o equivalente
// para escrita, e fechar conn aqui seria errado -- os testes fecham conn
// eles mesmos.
type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

// connectMCPClient disca e faz o handshake IPC (ipc.DialAndHandshake), e
// entao completa o handshake MCP por cima -- a mesma composicao que a ponte
// real faz, so que aqui e o proprio teste quem fala o protocolo, em vez de
// so copiar bytes.
func connectMCPClient(ctx context.Context, t *testing.T, vaultPath string) (*mcp.ClientSession, ipc.Conn) {
	t.Helper()

	conn, err := ipc.DialAndHandshake(ctx, vaultPath, false, boundedWait)
	if err != nil {
		t.Fatalf("DialAndHandshake: %v", err)
	}

	transport := &mcp.IOTransport{Reader: io.NopCloser(conn), Writer: nopWriteCloser{conn}}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	return session, conn
}

// TestDaemonSaiPorOciosidade e o teste nomeado na prova de mutacao do brief
// da Task 92: se "if time.Since(ultimoCliente) > cfg.OciosidadeMax {" virar
// "if false {", Run nunca chama aoOcioso, ctx nunca e cancelado por este
// mecanismo, e o teste reprova por timeout em vez de terminar.
func TestDaemonSaiPorOciosidade(t *testing.T) {
	d, _ := newTestDaemon(t, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var razoes []string
	// aoOcioso simula lifecycle.Lifecycle.Trigger: registra o motivo E
	// cancela o MESMO context que Run observa -- e o contrato real (ver o
	// comentario de Daemon.Run).
	aoOcioso := func(r string) {
		mu.Lock()
		razoes = append(razoes, r)
		mu.Unlock()
		cancel()
	}

	done := make(chan struct{})
	go func() {
		d.Run(ctx, aoOcioso)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(boundedWait):
		t.Fatal("Run nao retornou apos a ociosidade estourar -- aoOcioso nunca foi chamado")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(razoes) == 0 {
		t.Fatal("aoOcioso nunca foi chamado")
	}
	if razoes[0] != "idle" {
		t.Fatalf("primeiro motivo = %q, esperado \"idle\"", razoes[0])
	}
}

// TestDaemonNaoSaiComClienteConectado prova a outra metade da condicao de
// ociosidade: "ativos == 0" -- uma conexao aberta e sem trafego nao pode
// contar como ociosidade, mesmo que dure mais que OciosidadeMax.
func TestDaemonNaoSaiComClienteConectado(t *testing.T) {
	ociosidade := 80 * time.Millisecond
	d, vaultDir := newTestDaemon(t, ociosidade)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	aoOciosoChamado := make(chan string, 1)
	go d.Run(ctx, func(r string) {
		select {
		case aoOciosoChamado <- r:
		default:
		}
		cancel()
	})

	// A conexao fica aberta (handshake IPC completo, nenhuma chamada MCP
	// feita) por mais que OciosidadeMax -- se "ativos" nao fosse conferido,
	// o daemon sairia por ociosidade mesmo com um cliente pendurado.
	conn, err := ipc.DialAndHandshake(context.Background(), vaultDir, false, boundedWait)
	if err != nil {
		t.Fatalf("DialAndHandshake: %v", err)
	}
	defer func() { _ = conn.Close() }()

	select {
	case r := <-aoOciosoChamado:
		t.Fatalf("aoOcioso(%q) chamado com uma conexao ainda ativa", r)
	case <-time.After(3 * ociosidade):
		// Nenhum encerramento por ociosidade dentro de varios intervalos --
		// exatamente o esperado com uma conexao ativa.
	}
}

// TestDaemonAceitaDuasConexoesSimultaneas prova o perigo 2 do brief: o
// daemon serve VARIAS conexoes sobre o MESMO *mcpsrv.Server. Duas sessoes
// MCP completas (initialize + ListTools) ao mesmo tempo, sobre o mesmo
// daemon, tem de funcionar as duas.
func TestDaemonAceitaDuasConexoesSimultaneas(t *testing.T) {
	d, vaultDir := newTestDaemon(t, time.Hour) // ociosidade longa: nao e o que este teste mede

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.Run(ctx, func(string) {})

	sess1, conn1 := connectMCPClient(ctx, t, vaultDir)
	defer func() { _ = sess1.Close(); _ = conn1.Close() }()
	sess2, conn2 := connectMCPClient(ctx, t, vaultDir)
	defer func() { _ = sess2.Close(); _ = conn2.Close() }()

	if _, err := sess1.ListTools(ctx, nil); err != nil {
		t.Fatalf("sessao 1 ListTools: %v", err)
	}
	if _, err := sess2.ListTools(ctx, nil); err != nil {
		t.Fatalf("sessao 2 ListTools: %v", err)
	}
}

// TestDaemonConexaoQuebradaNaoDerrubaAsOutras prova a outra metade do
// perigo 2: uma conexao que morre abruptamente (fechada sem handshake de
// encerramento MCP nenhum) nao pode derrubar o daemon nem impedir que uma
// conexao NOVA seja aceita e servida em seguida.
func TestDaemonConexaoQuebradaNaoDerrubaAsOutras(t *testing.T) {
	d, vaultDir := newTestDaemon(t, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.Run(ctx, func(string) {})

	_, connQuebrada := connectMCPClient(ctx, t, vaultDir)
	// Fecha abruptamente no meio -- so o socket, sem sess.Close() nenhum.
	if err := connQuebrada.Close(); err != nil {
		t.Fatalf("fechando a conexao quebrada: %v", err)
	}

	// Da tempo do daemon perceber a quebra e desalocar a conexao.
	time.Sleep(200 * time.Millisecond)

	sessOK, connOK := connectMCPClient(ctx, t, vaultDir)
	defer func() { _ = sessOK.Close(); _ = connOK.Close() }()
	if _, err := sessOK.ListTools(ctx, nil); err != nil {
		t.Fatalf("conexao nova falhou apos outra quebrar: %v", err)
	}
}
