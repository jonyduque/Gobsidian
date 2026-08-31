// Package daemon implementa o processo de longa vida que serve UM cofre
// para N conexoes simultaneas sobre o socket de internal/ipc -- decisao 1
// da Task 92: um daemon por cofre, chaveado pelo mesmo hash de caminho que
// ja nomeia o diretorio de cache (config.VaultKey, via ipc.SocketPath).
//
// O daemon nao tem pai vigiavel: quem o inicia e uma ponte (cmd/gobsidian)
// que sai logo depois de lanca-lo (ver EnsureStarted em lock.go). Os tres
// mecanismos de encerramento de internal/lifecycle pressupoem um
// subprocesso de host -- EOF em stdin e vigilia do pai nao se aplicam
// aqui, e ligar a vigilia do pai "por consistencia" compararia contra um
// processo que ja saiu (a propria ponte que lancou o daemon), derrubando-o
// na hora. O que substitui os dois e a ociosidade: sem nenhum cliente
// conectado por Config.OciosidadeMax, o daemon sai -- usando a MESMA
// infraestrutura de lifecycle.Lifecycle.Trigger que sinal usa, para o gate
// de orfaos continuar conferindo "reason=" do mesmo jeito para os quatro
// mecanismos.
package daemon

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/ipc"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
)

// Config e o que o daemon precisa alem do config.Config do cofre.
type Config struct {
	Vault         config.Config
	OciosidadeMax time.Duration
}

// Daemon aceita conexoes no socket do cofre e as serve contra o MESMO
// *mcpsrv.Server -- o indice, o watcher e o servico por baixo dele sao
// compartilhados entre todas as sessoes (ver cmd/gobsidian/daemon.go, que
// monta esse Server uma unica vez, via construirServico, antes de chamar
// New).
type Daemon struct {
	ln  net.Listener
	srv *mcpsrv.Server
	cfg Config
	log *slog.Logger

	mu             sync.Mutex
	conexoesAtivas int
	ultimoCliente  time.Time
}

// New monta o daemon em cima de um listener e um servidor MCP ja
// construidos. Nao abre nada sozinho -- ln vem de ipc.Listen, chamado por
// quem orquestra o boot (cmd/gobsidian/daemon.go), porque abrir o socket
// cedo e o que permite ao arquivo de lock (lock.go) ser liberado assim que
// o processo esta de fato escutando, nao antes.
func New(ln net.Listener, srv *mcpsrv.Server, cfg Config, log *slog.Logger) *Daemon {
	return &Daemon{
		ln:            ln,
		srv:           srv,
		cfg:           cfg,
		log:           log,
		ultimoCliente: time.Now(),
	}
}

// intervaloMinimoDeChecagem evita um intervalo de checagem de ociosidade
// igual a zero quando OciosidadeMax e muito curto (como nos testes e no
// cenario novo do gate de orfaos) -- um time.Ticker com intervalo <= 0
// entra em panic.
const intervaloMinimoDeChecagem = 10 * time.Millisecond

// Run aceita conexoes ate ctx ser cancelado -- por sinal (ver
// lifecycle.Lifecycle, que quem chama monta) ou porque o proprio Run
// decidiu, via aoOcioso, que a ociosidade estourou -- e so retorna depois
// que o listener fechou e toda conexao em andamento terminou.
//
// aoOcioso e chamado no lugar de Run encerrar por conta propria porque quem
// decide COMO o processo morre e o lifecycle.Lifecycle de quem chama (ver
// o comentario do pacote): aoOcioso tipicamente e lc.Trigger, que cancela
// o MESMO context que Run observa, registrando "reason=idle" com a mesma
// infraestrutura de log que os outros tres mecanismos usam.
func (d *Daemon) Run(ctx context.Context, aoOcioso func(razao string)) {
	cfg := d.cfg

	intervalo := cfg.OciosidadeMax / 4
	if intervalo < intervaloMinimoDeChecagem {
		intervalo = intervaloMinimoDeChecagem
	}
	ticker := time.NewTicker(intervalo)
	defer ticker.Stop()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		d.acceptLoop(ctx, &wg)
	}()

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case <-ticker.C:
			d.mu.Lock()
			ativos := d.conexoesAtivas
			ultimoCliente := d.ultimoCliente
			d.mu.Unlock()

			// O log roda em toda checagem, independente do resultado --
			// alem de dar visibilidade de quanto falta para a ociosidade
			// estourar, isso mantem ultimoCliente referenciada mesmo se a
			// mutacao da Task 92 apagar so o "if" abaixo (ver o comentario
			// analogo em tools/netcheck/netcheck.go: um mutante que quebra
			// a compilacao nao e cobertura, e o mutate.ps1 sai inconclusivo
			// em vez de provar a regra).
			d.log.Debug("checagem de ociosidade", "ativos", ativos, "ocioso_ha", time.Since(ultimoCliente))

			if ativos == 0 {
				if time.Since(ultimoCliente) > cfg.OciosidadeMax {
					aoOcioso("idle")
				}
			}
		}
	}
}

// acceptLoop aceita conexoes ate o listener fechar. Uma goroutine separada
// fecha ln assim que ctx e cancelado -- e o que desbloqueia o Accept()
// preso, que nao tem outra forma portavel de ser interrompido por
// cancelamento de context (a mesma razao pela qual watchStdin, em
// internal/lifecycle, fica fora do WaitGroup: uma goroutine parada em Read
// nao e desenrolavel por cancelamento).
func (d *Daemon) acceptLoop(ctx context.Context, wg *sync.WaitGroup) {
	go func() {
		<-ctx.Done()
		_ = d.ln.Close()
	}()

	// falhasSeguidas conta erros CONSECUTIVOS de Accept; qualquer sucesso
	// zera. Ver o ramo de erro abaixo para o motivo do teto.
	var falhasSeguidas int

	for {
		conn, err := d.ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				// Encerramento normal: fechamos ln de proposito acima.
				return
			}
			// Erro de Accept nao mata o daemon na primeira vez.
			//
			// Ate 2026-08-26 este ramo fazia log.Warn e return. O processo
			// seguia vivo — socket bound, ticker de ociosidade rodando — e
			// nenhuma conexao era aceita nunca mais: os dials conectam no
			// backlog do SO e ninguem atende. A ponte pendura a sessao, o probe
			// do EnsureStarted estoura o prazo, e sobra um daemon surdo que so
			// reinicio resolve. EMFILE/ENFILE e o caso classico, e e
			// transitorio por definicao: o teto de descritores volta a subir
			// assim que qualquer conexao fecha.
			//
			// O TETO existe porque insistir sem limite troca um defeito por
			// outro. Um erro PERMANENTE nao classificado — listener corrompido,
			// socket removido debaixo do processo — viraria laco quente
			// consumindo CPU para sempre, em vez de morrer. Falhas
			// CONSECUTIVAS contam; qualquer accept bem-sucedido zera.
			falhasSeguidas++
			d.log.Warn("aceitar conexao no socket do daemon falhou",
				"err", err, "falhas_seguidas", falhasSeguidas, "teto", maxFalhasSeguidasDeAccept)
			if falhasSeguidas >= maxFalhasSeguidasDeAccept {
				d.log.Error("Accept falhou seguidamente ate o teto; encerrando o daemon",
					"falhas_seguidas", falhasSeguidas, "ultimo_erro", err)
				return
			}
			// Recuo curto: sem ele, um erro que reaparece na hora gira o laco.
			select {
			case <-ctx.Done():
				return
			case <-time.After(recuoDeAccept):
			}
			continue
		}
		falhasSeguidas = 0

		d.mu.Lock()
		d.conexoesAtivas++
		d.ultimoCliente = time.Now()
		d.mu.Unlock()

		wg.Add(1)
		go func() {
			defer wg.Done()
			d.handleConn(ctx, conn)
		}()
	}
}

// maxFalhasSeguidasDeAccept e o teto de erros CONSECUTIVOS antes de o daemon
// desistir. Cinco e folgado para o caso transitorio que motiva o recuo
// (EMFILE/ENFILE, que se resolve assim que uma conexao fecha) e curto o
// bastante para um erro permanente nao virar laco quente por muito tempo.
const maxFalhasSeguidasDeAccept = 5

// recuoDeAccept espaca as tentativas. Curto: um erro transitorio de descritor
// se resolve em milissegundos, e esperar mais so atrasa a ponte que ja esta
// discando do outro lado.
const recuoDeAccept = 50 * time.Millisecond

// handleConn serve UMA conexao. Uma sessao que quebra no meio de uma
// chamada -- erro de I/O, ou um panic em algo que guard() (internal/mcpsrv,
// RNF-13) nao cobre -- nao pode derrubar o daemon nem as outras conexoes: o
// recover aqui e a ultima linha de defesa depois da que ja existe por
// tool, cobrindo o resto da pilha da sessao.
func (d *Daemon) handleConn(ctx context.Context, conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			d.log.Error("sessao de cliente sofreu panic", "panic", r)
		}

		d.mu.Lock()
		d.conexoesAtivas--
		d.ultimoCliente = time.Now()
		d.mu.Unlock()

		_ = conn.Close()
	}()

	saudacao := ipc.HandshakeConfig{
		ReadOnly: d.cfg.Vault.ReadOnly,
		VaultKey: config.VaultKey(d.cfg.Vault.VaultPath),
		// O daemon ANUNCIA o teto com que subiu; a ponte que quer outro recusa
		// a conexao e cai para o modo em processo, onde a configuracao dela
		// vale (achado M9). O handshake e unidirecional, entao impor o da ponte
		// exigiria protocolo bidirecional — e divergencia visivel ja e melhor
		// que o silencio de antes.
		MaxResults: d.cfg.Vault.MaxResults,
	}
	if err := ipc.Greet(conn, saudacao); err != nil {
		d.log.Warn("saudacao ao cliente falhou", "err", err)
		return
	}

	// srv.Serve aceita io.Reader/io.Writer -- passar a MESMA conn duas
	// vezes basta, net.Conn ja separa leitura e escrita internamente.
	// Nenhum tipo do SDK MCP atravessa esta chamada (mcpsrv.Server.Serve
	// ja tinha essa assinatura desde a Task 91; nao precisou mudar para o
	// daemon).
	err := d.srv.Serve(ctx, conn, conn)
	if err != nil &&
		!errors.Is(err, context.Canceled) &&
		!errors.Is(err, io.EOF) &&
		!errors.Is(err, io.ErrClosedPipe) &&
		!errors.Is(err, os.ErrClosed) {
		d.log.Warn("sessao de cliente encerrada com erro", "err", err)
	}
}
