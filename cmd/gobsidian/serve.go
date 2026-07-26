package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/lifecycle"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var flags config.Flags

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve o cofre via MCP sobre stdio",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Flags booleanas e inteiras nao distinguem "omitida" de "definida
			// com o valor zero". Sem isso, --read-only=false nao consegue
			// sobrepor GOBSIDIAN_READ_ONLY=true, e --debounce-ms=0 e
			// indistinguivel de nao passar a flag.
			flags.ReadOnlySet = cmd.Flags().Changed("read-only")
			flags.DebounceMSSet = cmd.Flags().Changed("debounce-ms")

			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}
			return runServe(cmd.Context(), cfg)
		},
	}

	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
	cmd.Flags().StringVar(&flags.LogLevel, "log-level", "", "debug, info, warn ou error")
	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "desabilita toda a superficie de escrita")
	cmd.Flags().IntVar(&flags.DebounceMS, "debounce-ms", 0, "janela de coalescencia de eventos do watcher")
	cmd.Flags().StringVar(&flags.CacheDir, "cache-dir", "", "diretorio do cache de indice")

	return cmd
}

func runServe(parent context.Context, cfg config.Config) error {
	// stderr, sempre. stdout carrega o JSON-RPC e um unico byte estranho
	// corrompe a sessao.
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.LogLevel}))

	v, err := vault.New(cfg.VaultPath)
	if err != nil {
		return err
	}

	// O monitor de stdin consome bytes, e o stdin aqui pertence ao JSON-RPC.
	// A saida e espelhar: o SDK le do TeeReader, e o lifecycle observa a copia.
	// Quando o host fecha o stdin, ambos veem EOF.
	pr, pw := io.Pipe()
	teed := io.TeeReader(os.Stdin, pw)

	ctx, lc := lifecycle.New(parent, lifecycle.Options{
		Stdin:     pr,
		ParentPID: lifecycle.ParentPID(),
		Logger:    log,
	})

	svc := service.New(v, nil, service.Options{
		ReadOnly:   cfg.ReadOnly,
		MaxResults: cfg.MaxResults,
	})
	srv := mcpsrv.New(svc, cfg, log)

	log.Info("servidor pronto", "vault", cfg.VaultPath, "read_only", cfg.ReadOnly)

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ctx, teed, os.Stdout) }()

	select {
	case err := <-serveErr:
		if err != nil {
			log.Error("servidor encerrou com erro", "err", err)
		}
	case <-ctx.Done():
	}

	lifecycle.Shutdown(log, 6*time.Second,
		lifecycle.Step{Name: "in-flight", Budget: 3 * time.Second, Fn: func(ctx context.Context) error {
			select {
			case <-serveErr:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		}},
		lifecycle.Step{Name: "close-pipe", Budget: 500 * time.Millisecond, Fn: func(context.Context) error {
			return pw.Close()
		}},
	)

	lc.Wait()
	os.Exit(0)
	return nil
}
