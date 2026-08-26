package daemon_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/daemon"
	"github.com/jonyd/gobsidian/internal/ipc"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

// listenerFalho devolve N erros transitórios antes de delegar ao listener real.
//
// Erro transitório de Accept é clássico e não hipotético: EMFILE/ENFILE quando
// o processo bate no teto de descritores, e o teto volta a subir assim que
// qualquer conexão fecha.
type listenerFalho struct {
	net.Listener
	restantes atomic.Int32
	falhas    atomic.Int32
}

func (l *listenerFalho) Accept() (net.Conn, error) {
	if l.restantes.Add(-1) >= 0 {
		l.falhas.Add(1)
		return nil, errors.New("erro transitorio de accept (simulado)")
	}
	return l.Listener.Accept()
}

// TestAcceptLoopSobreviveAErroTransitorio cobre o A4, exercitando o
// acceptLoop DE PRODUÇÃO — daemon.New recebe o listener, então o dublê entra
// por injeção, sem reimplementar o laço.
//
// `acceptLoop` fazia `log.Warn` e `return` em QUALQUER erro sem cancelamento.
// O processo seguia vivo — socket bound, ticker de ociosidade rodando — e
// nenhuma conexão era aceita nunca mais: os dials conectam no backlog do SO e
// ninguém atende. A ponte pendura a sessão, o probe do EnsureStarted estoura o
// prazo, e sobra um daemon surdo que só reinício resolve.
func TestAcceptLoopSobreviveAErroTransitorio(t *testing.T) {
	root := t.TempDir()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	svc := service.New(v, nil, nil, nil, service.Options{})
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := mcpsrv.New(context.Background(), svc, config.Defaults(), log)

	chaveDoCofre := t.TempDir()
	ouvinte, caminho, err := ipc.Listen(chaveDoCofre)
	if err != nil {
		t.Skipf("socket unix indisponivel: %v", err)
	}
	t.Cleanup(func() { _ = ouvinte.Close() })

	falho := &listenerFalho{Listener: ouvinte}
	falho.restantes.Store(3)

	d := daemon.New(falho, srv, daemon.Config{
		Vault:         config.Config{VaultPath: chaveDoCofre},
		OciosidadeMax: time.Hour, // longe: quem encerra este teste e o cancel
	}, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	parou := make(chan struct{})
	go func() {
		defer close(parou)
		d.Run(ctx, func(string) {})
	}()

	// Depois dos erros transitorios, uma conexao real tem de ser ACEITA. O
	// contador de conexoes ativas do daemon e o que prova isso — dial sozinho
	// nao prova nada, porque o SO aceita no backlog mesmo sem ninguem atender.
	limite := time.Now().Add(5 * time.Second)
	var aceitou bool
	for time.Now().Before(limite) && !aceitou {
		conn, errDial := net.Dial("unix", caminho)
		if errDial != nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		// A saudacao so chega se alguem do outro lado atendeu de verdade.
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		buf := make([]byte, 1)
		if _, errRead := conn.Read(buf); errRead == nil {
			aceitou = true
		}
		_ = conn.Close()
		if !aceitou {
			time.Sleep(20 * time.Millisecond)
		}
	}

	if !aceitou {
		t.Errorf("nenhuma conexao atendida depois de %d erro(s) transitorio(s): "+
			"o laco de Accept morreu e o daemon ficou surdo", falho.falhas.Load())
	}

	cancel()
	select {
	case <-parou:
	case <-time.After(5 * time.Second):
		t.Error("Run nao encerrou apos o cancelamento")
	}
}
