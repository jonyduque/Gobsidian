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
	// A saida e espelhar: o SDK le do espelho, e o lifecycle observa a copia.
	pr, pw := io.Pipe()
	teed := &mirrorReader{src: os.Stdin, dst: pw}

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

	// serveErr tem capacidade 1 e recebe exatamente um valor. Se o select
	// abaixo ja o consumiu, a etapa in-flight nao pode tentar le-lo de novo —
	// ficaria bloqueada ate estourar o orcamento, atrasando todo encerramento
	// normal em 3 segundos e registrando uma falha que nao aconteceu.
	serveReturned := false
	var loopErr error

	select {
	case err := <-serveErr:
		serveReturned = true
		loopErr = err
		if err != nil {
			log.Error("servidor encerrou com erro", "err", err)
		}
	case <-ctx.Done():
	}

	lifecycle.Shutdown(log, 6*time.Second,
		lifecycle.Step{Name: "in-flight", Budget: 3 * time.Second, Fn: func(ctx context.Context) error {
			if serveReturned {
				return nil
			}
			select {
			case err := <-serveErr:
				loopErr = err
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
	if loopErr != nil {
		os.Exit(1)
	}
	os.Exit(0)
	return nil
}

// mirrorReader espelha o que le para dst e, crucialmente, propaga o fim da
// leitura fechando dst. E o que faz o EOF do host chegar ao monitor de stdin
// do lifecycle — io.TeeReader nao serve aqui porque so copia bytes, e EOF nao
// e um byte.
type mirrorReader struct {
	src io.Reader
	dst *io.PipeWriter
}

func (m *mirrorReader) Read(p []byte) (int, error) {
	n, err := m.src.Read(p)
	if n > 0 {
		if _, werr := m.dst.Write(p[:n]); werr != nil {
			return n, werr
		}
	}
	if err != nil {
		_ = m.dst.CloseWithError(err)
	}
	return n, err
}
