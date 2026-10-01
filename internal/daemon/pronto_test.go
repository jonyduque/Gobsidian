package daemon_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/daemon"
	"github.com/jonyduque/Gobsidian/internal/ipc"
	"github.com/jonyduque/Gobsidian/internal/mcpsrv"
	"github.com/jonyduque/Gobsidian/internal/service"
	"github.com/jonyduque/Gobsidian/internal/vault"
	"github.com/jonyduque/Gobsidian/internal/vaulttest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// daemonSemServico e newTestDaemon antes de o servico existir: o estado do
// daemon enquanto boot.Montar ainda constroi o indice.
func daemonSemServico(t *testing.T) (*daemon.Daemon, string, func() *mcpsrv.Server) {
	t.Helper()
	root := t.TempDir()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	montar := func() *mcpsrv.Server {
		return mcpsrv.New(context.Background(), service.New(v, nil, nil, nil, service.Options{}), config.Defaults(), log)
	}

	vaultDir := t.TempDir()
	ln, _, err := ipc.Listen(vaultDir)
	if err != nil {
		t.Fatalf("ipc.Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	cfg := daemon.Config{Vault: config.Config{VaultPath: vaultDir}, OciosidadeMax: time.Hour}
	return daemon.New(ln, nil, cfg, log), vaultDir, montar
}

// TestDaemonSaudaAntesDoServicoPronto: a saudacao nao espera o indice.
//
// Medido em 2026-09-26 no daemon do cofre Estudo: o socket abria antes de
// boot.Montar, mas o accept so comecava depois dele, e a saudacao saiu 37 s
// depois da conexao. A ponte desistiu aos 10 s e serviu em processo,
// construindo o MESMO indice uma segunda vez ao lado do daemon. Agora a
// saudacao sai na hora, e e a sessao MCP que espera o servico -- a mesma espera
// que o modo em processo ja impunha ao host, sem a construcao dobrada.
func TestDaemonSaudaAntesDoServicoPronto(t *testing.T) {
	d, vaultDir, montar := daemonSemServico(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.Run(ctx, func(string) {})

	// O mesmo prazo curto que a ponte usa no primeiro dial (ipcDialTimeout).
	conn, err := ipc.DialAndHandshake(ctx, vaultDir, false, 0, 300*time.Millisecond)
	if err != nil {
		t.Fatalf("saudacao com o servico ainda em montagem: %v", err)
	}
	defer func() { _ = conn.Close() }()

	transport := &mcp.IOTransport{Reader: io.NopCloser(conn), Writer: nopWriteCloser{conn}}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	conectou := make(chan error, 1)
	var sess *mcp.ClientSession
	go func() {
		s, err := client.Connect(ctx, transport, nil)
		sess = s
		conectou <- err
	}()

	select {
	case err := <-conectou:
		t.Fatalf("initialize respondeu antes de o servico existir (err=%v)", err)
	case <-time.After(150 * time.Millisecond):
	}

	d.Pronto(montar())

	select {
	case err := <-conectou:
		if err != nil {
			t.Fatalf("initialize depois de Pronto: %v", err)
		}
	case <-time.After(vaulttest.Prazo):
		t.Fatal("initialize nao respondeu depois de Pronto")
	}
	defer func() { _ = sess.Close() }()
	if _, err := sess.ListTools(ctx, nil); err != nil {
		t.Fatalf("ListTools: %v", err)
	}
}

// TestConexaoEsperandoOServicoNaoPrendeOEncerramento: uma conexao saudada que
// espera o servico tem de sair no cancelamento. Se ela ficasse presa, o
// wg.Wait de Run nunca voltaria -- a forma exata do daemon PID 42856, vivo 20 h
// em 2026-09-07 (ver lifecycle.ArmarGuardaChuva).
func TestConexaoEsperandoOServicoNaoPrendeOEncerramento(t *testing.T) {
	d, vaultDir, _ := daemonSemServico(t)
	ctx, cancel := context.WithCancel(context.Background())
	fim := make(chan struct{})
	go func() {
		d.Run(ctx, func(string) {})
		close(fim)
	}()

	conn, err := ipc.DialAndHandshake(ctx, vaultDir, false, 0, vaulttest.Prazo)
	if err != nil {
		t.Fatalf("DialAndHandshake: %v", err)
	}
	defer func() { _ = conn.Close() }()

	cancel()
	select {
	case <-fim:
	case <-time.After(vaulttest.Prazo):
		t.Fatal("Run nao voltou: a conexao que esperava o servico prendeu o encerramento")
	}
}
