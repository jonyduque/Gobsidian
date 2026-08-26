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

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/daemon"
	"github.com/jonyd/gobsidian/internal/ipc"
	"github.com/jonyd/gobsidian/internal/lifecycle"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
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

	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
	cmd.Flags().StringVar(&flags.LogLevel, "log-level", "", "debug, info, warn ou error")
	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "desabilita toda a superficie de escrita")
	cmd.Flags().IntVar(&flags.DebounceMS, "debounce-ms", 0, "janela de coalescencia de eventos do watcher")
	cmd.Flags().IntVar(&flags.MaxResults, "max-results", 0, "teto de resultados por consulta")
	cmd.Flags().StringVar(&flags.CacheDir, "cache-dir", "", "diretorio do cache de indice")
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
	return daemon.CaminhoDoLog(vaultPath)
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
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("abrindo log do daemon %s: %w", path, err)
	}

	log := slog.New(slog.NewTextHandler(io.MultiWriter(f, os.Stderr), &slog.HandlerOptions{Level: level}))
	return log, f.Close, nil
}

// runDaemon monta o indice/watcher/servico UMA VEZ (construirServico,
// servico.go -- a mesma sequencia de boot que serveEmProcesso usa) e o
// serve para N conexoes -- a diferenca central entre o daemon e
// serveEmProcesso, que serve exatamente uma sessao sobre stdio.
//
// Diferente de serveEmProcesso, o daemon nao tem stdin de host nem pai
// vigiavel: quem o inicia e uma ponte que sai logo depois de lanca-lo (ver
// internal/daemon, comentario do pacote). lifecycle.New roda so com o
// mecanismo de sinal -- Stdin e ParentPID ficam no zero-valor de
// proposito, o que desliga os outros dois mecanismos (ver
// internal/lifecycle.New: Stdin nil pula watchStdin, ParentPID<=0 pula
// watchParent). A ociosidade substitui os dois: lc.Trigger("idle") usa a
// MESMA infraestrutura de cancelamento e log que sinal e EOF de stdin usam
// nos outros processos, e o gate de orfaos confere "reason=idle" do mesmo
// jeito que confere os outros tres motivos (ver
// scripts/test_orphans.ps1, cenario "daemon-idle").
func runDaemon(parent context.Context, cfg config.Config, ociosidade time.Duration, log *slog.Logger) error {
	ln, sockPath, err := ipc.Listen(cfg.VaultPath)
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

	log.Info("daemon iniciado",
		"vault", cfg.VaultPath,
		"socket", sockPath,
		"read_only", cfg.ReadOnly,
		"ociosidade_s", ociosidade.Seconds())

	montado, err := construirServico(ctx, cfg, log)
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

	srv := mcpsrv.New(ctx, montado.svc, cfg, log)

	d := daemon.New(ln, srv, daemon.Config{Vault: cfg, OciosidadeMax: ociosidade}, log)
	d.Run(ctx, lc.Trigger)

	lifecycle.Shutdown(ctx, log, 6*time.Second,
		lifecycle.Step{Name: "watcher", Budget: 500 * time.Millisecond, Fn: func(context.Context) error {
			return montado.w.Close()
		}},
	)
	lc.Wait()
	montado.wg.Wait()

	log.Info("daemon encerrado", "reason", lc.Reason())
	return nil
}
