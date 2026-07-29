package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/lifecycle"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/watcher"
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

// shutdownExitCode traduz o erro final do serve loop em codigo de saida.
//
// Existe como funcao separada porque e a unica parte de runServe que da para
// testar sem levantar um processo: runServe termina em os.Exit por desenho, e
// os.Exit nao volta.
//
// Os tres erros tratados como encerramento normal sao os que aparecem quando
// o host simplesmente vai embora. context.Canceled vem do proprio lifecycle,
// que cancela o contexto quando o stdin fecha, um sinal chega ou o pai morre.
// io.EOF e io.ErrClosedPipe podem vir do SDK, que detecta o fim do stdin por
// conta propria — as duas deteccoes correm, e qual delas vence decide qual
// valor chega aqui. Tratar qualquer uma como falha faz um host supervisor ver
// erro aleatorio a cada desconexao limpa.
func shutdownExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, context.Canceled),
		errors.Is(err, io.EOF),
		errors.Is(err, io.ErrClosedPipe):
		return 0
	default:
		return 1
	}
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

	// A duracao da indexacao a frio e RNF-01, e ate aqui ninguem a media: quem
	// quisesse o numero cronometrava o processo inteiro por fora, misturando
	// boot do Go e handshake do MCP com o que o alvo cobre. Logar aqui torna a
	// medicao reproduzivel e recorta exatamente o trecho que o requisito nomeia.
	buildStart := time.Now()
	idx := index.New()
	if err := idx.Build(ctx, v); err != nil {
		return err
	}
	indexMS := time.Since(buildStart).Milliseconds()

	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		if data, err := v.ReadAll(ctx, p); err == nil {
			body, _ := vault.StripBOM(data)
			inv.Add(string(p), search.Analyze(string(body)))
		}
	}

	w, err := watcher.New(v, idx, time.Duration(cfg.DebounceMS)*time.Millisecond, log)
	if err != nil {
		return err
	}

	svc := service.New(v, idx, inv, watcherStats{w: w}, service.Options{
		ReadOnly:   cfg.ReadOnly,
		MaxResults: cfg.MaxResults,
	})
	srv := mcpsrv.New(ctx, svc, cfg, log)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = w.Run(ctx)
	}()

	log.Info("servidor pronto",
		"vault", cfg.VaultPath,
		"read_only", cfg.ReadOnly,
		"notes", idx.NoteCount(),
		"assets", idx.AssetCount(),
		"index_ms", indexMS)

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ctx, teed, os.Stdout) }()

	// serveErr tem capacidade 1 e recebe exatamente um valor. Se o select
	// abaixo ja o consumiu, a etapa in-flight nao pode tentar le-lo de novo —
	// ficaria bloqueada ate estourar o orcamento, atrasando todo encerramento
	// normal em 3 segundos e registrando uma falha que nao aconteceu.
	serveReturned := false
	var loopErr error

	// A etapa in-flight roda na goroutine que lifecycle.Shutdown lanca para
	// ela, e pode ficar orfa se o orcamento estourar antes dela terminar —
	// "abandonada" quer dizer exatamente isso. Se ela escrevesse direto em
	// loopErr, essa escrita correria com a leitura da goroutine principal
	// logo depois de lc.Wait(). Um canal com buffer 1 carrega a garantia de
	// happens-before que uma variavel compartilhada nao tem.
	lateErr := make(chan error, 1)

	select {
	case err := <-serveErr:
		serveReturned = true
		loopErr = err
		if err != nil {
			log.Error("servidor encerrou com erro", "err", err)
		}
	case <-ctx.Done():
	}

	lifecycle.Shutdown(ctx, log, 6*time.Second,
		lifecycle.Step{Name: "in-flight", Budget: 3 * time.Second, Fn: func(ctx context.Context) error {
			if serveReturned {
				return nil
			}
			select {
			case err := <-serveErr:
				lateErr <- err
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		}},
		lifecycle.Step{Name: "close-pipe", Budget: 500 * time.Millisecond, Fn: func(context.Context) error {
			return pw.Close()
		}},
		lifecycle.Step{Name: "watcher", Budget: 500 * time.Millisecond, Fn: func(context.Context) error {
			return w.Close()
		}},
	)

	lc.Wait()
	wg.Wait()

	// Depois de Wait, nao antes: a etapa in-flight pode ter sido abandonada
	// por estouro de orcamento, e sua goroutine ainda estar a caminho do
	// canal. Wait e o ultimo ponto em que esperar por ela e de graca.
	//
	// A drenagem continua nao-bloqueante. Se mesmo assim nao houver valor, o
	// erro tardio e descartado de proposito: o encerramento ja estourou o
	// orcamento, e travar aqui trocaria um exit code impreciso por um
	// servidor que nao encerra.
	select {
	case err := <-lateErr:
		if loopErr == nil {
			loopErr = err
		}
	default:
	}

	// runServe nao retorna: termina o processo aqui para que o codigo de saida
	// seja o desta decisao e nao o que cobra derivaria de um error. O return
	// abaixo e inalcancavel e existe so para satisfazer a assinatura que RunE
	// exige.
	os.Exit(shutdownExitCode(loopErr))
	return nil
}

// mirrorDst e o subconjunto de *io.PipeWriter que mirrorReader usa. Extrair
// para interface nao muda nada em producao — pw continua sendo um
// *io.PipeWriter de verdade — mas deixa o teste trocar o espelho por um
// dublê que conta tentativas de escrita. Sem isso nao ha como observar, de
// fora, que a guarda !m.broken impediu uma segunda escrita: o retorno de
// Read e o mesmo com ou sem a guarda, so o numero de chamadas ao espelho
// difere.
type mirrorDst interface {
	io.Writer
	CloseWithError(error) error
}

// mirrorReader espelha o que le para dst e, crucialmente, propaga o fim da
// leitura fechando dst. E o que faz o EOF do host chegar ao monitor de stdin
// do lifecycle — io.TeeReader nao serve aqui porque so copia bytes, e EOF nao
// e um byte.
type mirrorReader struct {
	src    io.Reader
	dst    mirrorDst
	broken bool // espelho desistiu; a leitura principal segue intacta
}

func (m *mirrorReader) Read(p []byte) (int, error) {
	n, err := m.src.Read(p)

	// O espelho e auxiliar: existe so para o lifecycle enxergar o EOF. Se a
	// escrita nele falhar, o JSON-RPC continua — devolver o erro da escrita
	// no lugar do resultado da leitura injetaria uma falha inventada em uma
	// sessao saudavel, e o cliente veria a conexao morrer sem motivo.
	if n > 0 && !m.broken {
		if _, werr := m.dst.Write(p[:n]); werr != nil {
			m.broken = true
		}
	}

	if err != nil {
		_ = m.dst.CloseWithError(err)
	}
	return n, err
}

type watcherStats struct{ w *watcher.Watcher }

func (a watcherStats) Stats() service.WatchCounters {
	c := a.w.Stats()
	return service.WatchCounters{
		Active:            c.Active,
		EventsReceived:    c.EventsReceived,
		EventsDropped:     c.EventsDropped,
		DroppedByReason:   c.DroppedByReason,
		EventsCoalesced:   c.EventsCoalesced,
		EventsProcessed:   c.EventsProcessed,
		EventsSkipped:     c.EventsSkipped,
		Reconciliations:   c.Reconciliations,
		ReconciledUpdated: c.ReconciledUpdated,
		ReconciledRemoved: c.ReconciledRemoved,
	}
}
