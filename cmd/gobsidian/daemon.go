// daemon.go implementa o subcomando `gobsidian daemon` -- o processo de
// longa vida que uma ponte (ponte.go) lanca quando nenhum daemon do cofre
// ja esta rodando (ver internal/daemon.EnsureStarted). Uso interno: uma
// ponte lanca isto sozinha, ninguem digita `gobsidian daemon` a mao no
// fluxo normal -- por isso o comando fica Hidden na ajuda.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jonyd/gobsidian/internal/boot"
	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/daemon"
	"github.com/jonyd/gobsidian/internal/instalar"
	"github.com/jonyd/gobsidian/internal/lifecycle"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/spf13/cobra"
)

func newDaemonCmd() *cobra.Command {
	var flags config.Flags
	var idleSeconds int

	cmd := &cobra.Command{
		Use:    "daemon",
		Short:  "Roda o daemon de cofre compartilhado (uso interno da ponte)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			flags.ReadOnlySet = cmd.Flags().Changed("read-only")
			flags.DebounceMSSet = cmd.Flags().Changed("debounce-ms")
			flags.MaxResultsSet = cmd.Flags().Changed("max-results")

			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}
			if idleSeconds < 1 {
				return fmt.Errorf("--idle-seconds precisa ser >= 1 (recebido %d)", idleSeconds)
			}

			log, closeLog, err := novoLoggerDoDaemon(cfg.VaultPath, cfg.LogLevel)
			if err != nil {
				return err
			}
			defer func() { _ = closeLog() }()

			return runDaemon(cmd.Context(), cfg, time.Duration(idleSeconds)*time.Second, log)
		},
	}

	flagsDeCofre(cmd, &flags)
	flagsDeCache(cmd, &flags)
	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "desabilita toda a superficie de escrita")
	cmd.Flags().IntVar(&flags.DebounceMS, "debounce-ms", 0, "janela de coalescencia de eventos do watcher")
	cmd.Flags().IntVar(&flags.MaxResults, "max-results", 0, "teto de resultados por consulta")
	cmd.Flags().BoolVar(&flags.EagerSearch, "eager-search", false,
		"carrega o indice de busca no boot em vez de esperar a primeira vault_search")
	cmd.Flags().IntVar(&idleSeconds, "idle-seconds", daemon.DefaultIdleSeconds,
		"segundos sem cliente conectado antes do daemon encerrar (decisao 3 da Task 92; padrao 15 minutos)")

	return cmd
}

// daemonLogPath delega para daemon.CaminhoDoLog: a conta do caminho do log
// mora num lugar so, ao lado da do socket e da do lock. Ate 2026-08-26 ela
// estava em tres lugares -- aqui, em internal/daemon e em internal/doctor --
// e tres contas do mesmo valor concordam por coincidencia ate uma mudar.
func daemonLogPath(vaultPath string) (string, error) {
	return caminhoDoLogFn(vaultPath)
}

// caminhoDoLogFn e daemon.CaminhoDoLog numa variavel, para o teste desviar o
// log para um t.TempDir().
//
// Desviar por VARIAVEL DE AMBIENTE nao funciona nas tres plataformas: o caminho
// sai de os.UserCacheDir(), que respeita LOCALAPPDATA no Windows e
// XDG_CACHE_HOME no Linux, e IGNORA os dois no macOS -- la ele devolve
// $HOME/Library/Caches. Um teste que confiasse no env escreveria no cache real
// do usuario naquela plataforma, que e a classe de defeito registrada em
// docs/ARMADILHAS.md. Producao nunca troca esta variavel.
var caminhoDoLogFn = daemon.CaminhoDoLog

// tetoDoLogDoDaemon e o tamanho a partir do qual o log e rotacionado.
//
// 5 MB. Medido em 2026-09-08: o log do cofre Estudo tinha 727 261 bytes e
// nenhum limite -- ele so cresce, para sempre, porque N instancias fazem
// append no MESMO arquivo ao longo de meses.
//
// E variavel, e nao constante, para o teste poder encolhe-la: escrever 5 MB
// num teste gastaria disco para provar o mesmo comportamento. Producao nunca
// a troca -- o mesmo padrao de daemonStartTimeout em ponte.go.
var tetoDoLogDoDaemon int64 = 5 << 20

// rotacionarLogDoDaemon guarda o log corrente como ".1" quando ele passa do
// teto, e nao faz nada abaixo dele.
//
// ROTACIONA, nunca apaga. O log e a unica memoria do daemon: ele nao tem
// terminal, e a investigacao de 2026-09-08 dependeu de linhas de 2026-08-24
// para reconstruir a sequencia de partidas e mortes de um cofre. Um arquivo
// anterior basta para nao perder a janela recente sem crescer sem limite.
//
// Falha de rotacao NAO impede a abertura do log, e por isso esta funcao nao
// devolve erro: um daemon sem log e pior que um log grande, e este processo e
// detachado -- se ele nao escrever aqui, nao escreve em lugar nenhum. Foi
// exatamente esse o defeito de 2026-08-26, com dois daemons morrendo mudos.
func rotacionarLogDoDaemon(path string) {
	fi, err := os.Stat(path)
	if err != nil || fi.Size() < tetoDoLogDoDaemon {
		return
	}
	// os.Rename sobre um destino existente substitui, nas tres plataformas --
	// que e o comportamento desejado: guarda-se UM anterior, nao uma serie.
	_ = os.Rename(path, path+".1")
}

// novoLoggerDoDaemon abre o arquivo de log do daemon e devolve o logger
// junto com uma funcao de fechamento. O daemon nao tem terminal -- stderr de
// um processo detachado de verdade (SpawnDetached, internal/daemon/spawn.go)
// nao vai a lugar nenhum que alguem possa ler -- entao ele precisa de um
// destino de log proprio, ao contrario de `serve`, que sempre usa stderr
// porque tem um host do outro lado.
//
// O log vai para os DOIS lugares -- arquivo E stderr -- via io.MultiWriter.
// stderr sozinho nao bastaria (o caso comum e detachado, sem ninguem
// lendo); o arquivo sozinho tornaria o daemon opaco para quem o lanca
// diretamente sem detachar (depuracao manual, e o cenario "daemon-idle" do
// gate de orfaos, que redireciona o stderr do processo para o log do ciclo
// exatamente como os outros tres cenarios fazem com o servidor). Escrever
// em stderr aqui nao viola "stdout pertence ao JSON-RPC": o daemon nunca
// fala JSON-RPC por stdout OU stderr, so pelo socket.
func novoLoggerDoDaemon(vaultPath string, level slog.Level) (*slog.Logger, func() error, error) {
	path, err := daemonLogPath(vaultPath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolvendo caminho do log do daemon: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, nil, fmt.Errorf("criando diretorio do log do daemon: %w", err)
	}
	rotacionarLogDoDaemon(path)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("abrindo log do daemon %s: %w", path, err)
	}

	// pid e versao em TODA linha, e nao so na de partida.
	//
	// O arquivo e UNICO por cofre e recebe append de N instancias ao longo de
	// meses -- 727 261 bytes na maquina do dono, medido em 2026-09-08. Sem
	// estes dois campos, descobrir qual processo escreveu uma linha exige
	// cruzar mtime de arquivo com StartTime de processo, e o resultado fica
	// ambiguo justamente no caso que importa: duas instancias do mesmo cofre
	// convivendo, que foi o estado investigado.
	//
	// Sao campos ACRESCENTADOS. Nenhuma mensagem muda de texto:
	// scripts/measure.ps1 casa "servidor pronto" e a regex index_ms=(\d+), e
	// scripts/test_orphans.ps1 casa reason= -- os dois continuam lendo o que
	// liam.
	log := slog.New(slog.NewTextHandler(io.MultiWriter(f, os.Stderr), &slog.HandlerOptions{Level: level})).
		With("pid", os.Getpid(), "versao", version)
	return log, f.Close, nil
}

// runDaemon monta o indice/watcher/servico UMA VEZ (boot.Montar -- a mesma
// sequencia de boot que serveEmProcesso usa) e o
// serve para N conexoes -- a diferenca central entre o daemon e
// serveEmProcesso, que serve exatamente uma sessao sobre stdio.
//
// Diferente de serveEmProcesso, o daemon nao tem stdin de host nem pai
// vigiavel: quem o inicia e uma ponte que sai logo depois de lanca-lo (ver
// internal/daemon, comentario do pacote). Por isso ele NAO usa
// boot.VigiarHost — nao ha host para vigiar, e forcar um espelho de stdin
// aqui ligaria uma vigilia sobre um descritor que ninguem alimenta.
// lifecycle.New roda so com o
// mecanismo de sinal -- Stdin e ParentPID ficam no zero-valor de
// proposito, o que desliga os outros dois mecanismos (ver
// internal/lifecycle.New: Stdin nil pula watchStdin, ParentPID<=0 pula
// watchParent). A ociosidade substitui os dois: lc.Trigger("idle") usa a
// MESMA infraestrutura de cancelamento e log que sinal e EOF de stdin usam
// nos outros processos, e o gate de orfaos confere "reason=idle" do mesmo
// jeito que confere os outros tres motivos (ver
// scripts/test_orphans.ps1, cenario "daemon-idle").
func runDaemon(parent context.Context, cfg config.Config, ociosidade time.Duration, log *slog.Logger) error {
	// A sonda de orfao e o bind vao sob o MESMO lock.
	//
	// ipc.Listen ja prova que o socket esta orfao antes de desvincula-lo, mas a
	// sonda e o bind nao sao atomicos entre si: dois daemons lancados no mesmo
	// instante podem ambos sondar "ninguem escuta" antes de qualquer um bindar.
	// E o item 4 do brief da Task 126, a metade que a prova de orfao nao fecha.
	ln, sockPath, err := daemon.EscutarComLock(cfg.VaultPath)
	if err != nil {
		// Todo caminho de saida do daemon loga a causa ANTES de sair.
		//
		// Nao e zelo: o daemon e detachado (SpawnDetached), entao o erro
		// devolvido daqui sobe ate o cobra e e impresso num stderr que nao
		// tem leitor. Medido em 2026-08-26 na maquina do dono: dois daemons
		// morreram deixando so a linha "daemon iniciado" no arquivo de log, a
		// ponte esperou o prazo inteiro do EnsureStarted, e o unico rastro no
		// host foi "Server transport closed unexpectedly". A causa existia e
		// era especifica; nunca chegou a ninguem.
		log.Error("daemon nao pode abrir o socket",
			"vault", cfg.VaultPath, "err", err)
		return fmt.Errorf("abrindo socket do daemon: %w", err)
	}

	ctx, lc := lifecycle.New(parent, lifecycle.Options{Logger: log})

	// O guarda-chuva cobre TUDO daqui para a frente: d.Run (e o wg.Wait dentro
	// dele), Shutdown, lc.Wait e c.Esperar.
	//
	// Armado AQUI, e nao antes de Shutdown, porque wg.Wait mora DENTRO de
	// d.Run: um guarda armado depois de Run nunca alcancaria a espera que
	// provavelmente pendurou o PID 42856 em 2026-09-07. Ver ArmarGuardaChuva
	// para a medicao.
	//
	// defer, e nao uma chamada no fim: o retorno da funcao E o fim do
	// encerramento, e os ramos de erro acima e abaixo saem por ele tambem.
	defer lifecycle.ArmarGuardaChuva(ctx, log, lifecycle.OrcamentoDeEncerramento)()

	// DEPOIS de lifecycle.New, nunca antes: ver prepararProcesso (serve.go).
	//
	// O daemon precisa disto tanto quanto o serve, e por um motivo proprio: ele
	// e lancado por uma ponte que ja saiu, entao ninguem estaria olhando se ele
	// subisse no meio da troca do binario.
	//
	// runDaemon retorna normalmente, entao o defer roda; em serve.go, que
	// termina em os.Exit, a liberacao e explicita.
	if prepararProcesso(log, "daemon", cfg.VaultPath) {
		_ = ln.Close()
		return nil
	}
	defer instalar.LiberarPresenca()

	log.Info("daemon iniciado",
		"vault", cfg.VaultPath,
		"socket", sockPath,
		"read_only", cfg.ReadOnly,
		"ociosidade_s", ociosidade.Seconds())

	c, err := boot.Montar(ctx, cfg, service.ModoDaemon, log)
	if err != nil {
		// Este e o ramo que matou os dois daemons de 2026-08-26: e aqui que
		// vault.New recusa o cofre (internal/vault/vault.go:90-95, "raiz do
		// cofre inacessivel %q"). A validacao ja existia e ja nomeava o
		// caminho -- o defeito era o erro nao chegar ao log.
		log.Error("daemon nao pode montar o servico",
			"vault", cfg.VaultPath, "err", err)
		_ = ln.Close()
		return err
	}

	srv := mcpsrv.New(ctx, c.Service, cfg, log)

	d := daemon.New(ln, srv, daemon.Config{Vault: cfg, OciosidadeMax: ociosidade}, log)
	d.Run(ctx, lc.Trigger)

	lifecycle.Shutdown(ctx, log, lifecycle.OrcamentoDeEncerramento,
		c.PassoWatcher(),
	)
	lifecycle.Esperar(ctx, log, "lifecycle", lc.Wait)
	lifecycle.Esperar(ctx, log, "goroutines-de-fundo", c.Esperar)

	log.Info("daemon encerrado", "reason", lc.Reason())
	return nil
}
