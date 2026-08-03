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
	"github.com/jonyd/gobsidian/internal/writer"
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

// invertedSaveInterval limita QUANTO TRABALHO se perde num encerramento
// abrupto durante a construcao.
//
// O que se quer limitar e tempo, nao contagem de notas: uma nota de 200 bytes
// e uma de 200 KB custam ordens de grandeza diferentes, e um contador de notas
// da garantias diferentes em cofres diferentes.
//
// A gravacao NAO e incremental por dentro — cada uma serializa o cache inteiro.
// Gravar a cada 250 notas custou +39% no tempo total de construcao (303 s
// contra 219 s), medido quando o cache do cofre de referencia tinha 472 MB no
// formato gob. Com o formato binario ele passou a 66 MB, entao esse custo so
// pode ter caido — mas nao foi medido de novo, e o intervalo de 60 s nao foi
// reajustado por causa disso.
//
// 60 s porque esta gravacao e a UNICA rede: o encerramento nao grava parcial
// (ver o caminho de cancelamento em buildInvertedIndex), entao o que se perde e
// o trabalho desde a ultima passagem por aqui.
const invertedSaveInterval = 60 * time.Second

// invertedCacheState decide o que fazer com o que veio do disco.
//
// Existe separada porque e a regra que impede o defeito mais caro desta area:
// LoadInvertedCache confere versao de formato, de parser, de analisador e o
// caminho do cofre — NAO confere cobertura. Como a construcao grava parciais,
// aceitar o cache sem olhar a contagem faria a busca servir um indice
// incompleto como se fosse completo, e o cliente receberia menos notas do que
// existem sem nada indicando isso.
//
// pronta  -> o cache cobre o cofre inteiro
// retomar -> o cache e utilizavel mas incompleto; serve de ponto de partida
//
// A comparacao e >= e nao ==: notas APAGADAS do cofre deixam entradas velhas no
// cache, e nesse caso ele cobre tudo o que existe. As sobras nao vazam para o
// resultado porque a busca so devolve o que o indice de metadados confirma.
func invertedCacheState(hdr *search.CacheHeader, err error, noteCount int) (pronta, retomar bool) {
	if err != nil || hdr == nil {
		return false, false
	}
	if hdr.NoteCount >= noteCount {
		return true, false
	}
	return false, true
}

// prepararIndiceDeBusca carrega o cache e completa o que faltar.
//
// Roda em segundo plano, fora do caminho de boot. Devolve com inv pronto ou com
// o contexto cancelado; em nenhum outro caso o indice fica marcado em
// construcao para sempre.
//
// Precisa rodar ANTES de o watcher comecar a consumir eventos: a adocao do
// cache substitui o conteudo de inv. Ver search.Inverted.AdotarDe.
func prepararIndiceDeBusca(
	ctx context.Context,
	v *vault.Vault,
	idx *index.Index,
	inv *search.Inverted,
	cfg config.Config,
	log *slog.Logger,
) {
	inicio := time.Now()
	doCache, hdr, err := search.LoadInvertedCache(ctx, cfg.CacheDir, cfg.VaultPath)
	pronta, retomar := invertedCacheState(hdr, err, idx.NoteCount())

	// adotado separa "o cache nao servia" de "o cache servia mas nao pode ser
	// aplicado". O segundo caso so acontece se alguem escreveu no indice antes
	// desta funcao, o que a ordem em runServe impede — e se a ordem mudar, o
	// recuo e construir do zero, nunca aplicar por cima.
	adotado := func() bool {
		if err := inv.AdotarDe(doCache); err != nil {
			log.Warn("cache de busca nao aplicado; construindo do zero", "err", err)
			return false
		}
		return true
	}

	switch {
	case pronta && adotado():
		inv.MarkReady()
		log.Info("indice de busca pronto",
			"origem", "cache",
			"notas", idx.NoteCount(),
			"duracao_ms", time.Since(inicio).Milliseconds())
		return
	case retomar && adotado():
		log.Info("cache de busca parcial encontrado; retomando",
			"no_cache", hdr.NoteCount, "no_cofre", idx.NoteCount(),
			"carga_ms", time.Since(inicio).Milliseconds())
	default:
		// Cache ausente, de versao incompativel ou corrompido. Os tres levam ao
		// mesmo lugar — construir do zero — e a distincao entre eles ja saiu no
		// log de LoadInvertedCache.
		if err != nil && !errors.Is(err, search.ErrCacheNotFound) {
			log.Warn("cache de busca descartado", "err", err)
		}
	}

	buildInvertedIndex(ctx, v, idx, inv, cfg, log)
}

// buildInvertedIndex tokeniza o cofre para dentro de inv e o marca pronto.
//
// Roda em segundo plano: ver o comentario no ponto de chamada. Marca pronto
// mesmo com notas individuais falhando — uma nota ilegivel nao pode deixar a
// busca desligada para sempre; ela aparece no log e fica de fora do indice.
func buildInvertedIndex(
	ctx context.Context,
	v *vault.Vault,
	idx *index.Index,
	inv *search.Inverted,
	cfg config.Config,
	log *slog.Logger,
) {
	inicio := time.Now()
	caminhos := idx.NotePaths()
	log.Info("construindo indice de busca em segundo plano", "notas", len(caminhos))

	salvar := func(parcial bool) {
		if err := search.SaveInvertedCache(ctx, cfg.CacheDir, cfg.VaultPath, inv); err != nil {
			log.Warn("falha ao salvar cache invertido de busca", "parcial", parcial, "err", err)
		}
	}

	feitas := 0
	jaCobertas := 0
	ultimoSalvo := time.Now()
	for _, p := range caminhos {
		// Retomada: o que o cache parcial ja trouxe nao e relido.
		//
		// HasDoc e nao DocLength > 0: uma nota vazia tem tamanho zero e estar
		// no indice, e a versao antiga a relia a cada retomada sem nunca
		// passar a conta-la como coberta.
		if inv.HasDoc(string(p)) {
			jaCobertas++
			feitas++
			continue
		}
		if ctx.Err() != nil {
			// Sai sem gravar, de proposito.
			//
			// SaveInvertedCache confere o context so na ENTRADA: uma vez comecada,
			// a gravacao nao e interrompivel, e mediu ate 10 s num encerramento
			// real quando o cache tinha 472 MB no formato gob. Hoje ele tem
			// 66 MB; o raciocinio nao depende do numero, so de a gravacao nao
			// ser interrompivel. runServe faz wg.Wait() DEPOIS de
			// lifecycle.Shutdown, entao essa espera nao passa por orcamento
			// nenhum — e o harness de orfaos conta como orfao o que nao morre em
			// 8 s. Um WithTimeout aqui impediria COMECAR, nunca terminar: seria
			// um orcamento decorativo.
			//
			// O que limita a perda e a gravacao periodica acima. Perder ate um
			// intervalo de trabalho e barato; furar o requisito de bloqueio de
			// release nao e.
			log.Warn("construcao do indice de busca interrompida",
				"feitas", feitas, "de", len(caminhos),
				"perdido_desde_ultima_gravacao_ms", time.Since(ultimoSalvo).Milliseconds())
			return
		}
		data, err := v.ReadAll(ctx, p)
		if err != nil {
			log.Warn("falha ao ler nota ao construir indice invertido de busca", "path", p, "err", err)
			continue
		}
		body, _ := vault.StripBOM(data)
		inv.Add(string(p), search.Analyze(string(body)))

		feitas++
		if time.Since(ultimoSalvo) >= invertedSaveInterval {
			salvar(true)
			ultimoSalvo = time.Now()
			log.Debug("cache de busca gravado parcialmente", "feitas", feitas, "de", len(caminhos))
		}
	}

	salvar(false)
	inv.MarkReady()
	log.Info("indice de busca pronto",
		"origem", "construcao",
		"notas", feitas,
		"reaproveitadas_do_cache", jaCobertas,
		"duracao_ms", time.Since(inicio).Milliseconds())
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

	// Recuperacao de crash: um processo morto no meio de uma escrita nao roda
	// defer, e deixa o temporario no cofre. Aqui e o unico lugar sem escrita em
	// voo, entao e o unico lugar onde varrer o diretorio nao corre risco de
	// apagar o temporario de outra escrita. Ver writer.CleanStaleTempFiles.
	if removidos, err := writer.SweepStaleTempFiles(ctx, cfg.VaultPath); err != nil {
		log.Warn("varredura de temporarios interrompida", "err", err)
	} else if removidos > 0 {
		log.Warn("temporarios de escritas interrompidas removidos", "count", removidos)
	}

	// Nem a construcao NEM o carregamento do cache bloqueiam o anuncio das
	// tools.
	//
	// A construcao custa proporcionalmente aos BYTES do cofre, nao a contagem de
	// notas: num cofre real de 109 MB a tokenizacao levou 219 s, contra 1,3 s do
	// indice de metadados. O host desiste do handshake MCP em 30 s e mata o
	// processo — antes de SaveInvertedCache rodar, entao a tentativa seguinte
	// recomecava do zero. Toda tentativa falhava pelo mesmo motivo, para sempre,
	// e nada no log dizia isso: "servidor pronto" so aparecia depois.
	//
	// O carregamento do cache saiu do caminho de boot pelo mesmo raciocinio,
	// aplicado a uma escala menor: medido em 1,83 s no cofre de referencia,
	// contra 1,3 s do indice de metadados. Nao e o suficiente para estourar o
	// handshake, mas era mais da metade do tempo ate o servidor responder, gasto
	// numa estrutura que nenhuma tool precisa para ser anunciada.
	//
	// O indice entregue aqui esta VAZIO e marcado como em construcao. Quem
	// consulta a busca nesse intervalo recebe INDEX_BUILDING, e nao zero
	// resultados: cofre sem a palavra e indice ainda sem a palavra nao podem
	// produzir a mesma resposta.
	inv := search.NewInverted()
	inv.MarkBuilding()

	// watcher.New ANTES da construcao, e w.Run depois dela.
	//
	// New registra os watches, entao os eventos do periodo de construcao ficam
	// enfileirados no fsnotify em vez de se perderem; Run os consome depois e
	// reindexa com o conteudo corrente do disco, que e no minimo tao novo
	// quanto o que a construcao leu. Sem isso, os dois escreveriam no mesmo
	// indice ao mesmo tempo e a construcao poderia sobrescrever uma nota que o
	// watcher acabara de atualizar.
	//
	// Se a fila estourar, o fsnotify emite ErrEventOverflow e o caminho de
	// reconciliacao por varredura completa assume — que e exatamente o que ele
	// existe para fazer.
	w, err := watcher.New(v, idx, inv, time.Duration(cfg.DebounceMS)*time.Millisecond, log)
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
		// prepararIndiceDeBusca ANTES de w.Run, e nao em paralelo com ele.
		//
		// A adocao do cache SUBSTITUI o conteudo do indice (ver
		// search.Inverted.AdotarDe). Um evento do watcher aplicado antes dela
		// seria descartado sem deixar rastro, e o sintoma seria uma nota editada
		// durante o boot sumir da busca ate o proximo reinicio. Os eventos desse
		// intervalo nao se perdem: watcher.New ja registrou os watches, entao
		// eles ficam enfileirados no fsnotify e w.Run os consome em seguida.
		prepararIndiceDeBusca(ctx, v, idx, inv, cfg, log)
		_ = w.Run(ctx)
	}()

	// index_ms cobre SO o indice de metadados, e e o que RNF-01 nomeia.
	//
	// search_ready some daqui de proposito. Com o cache carregado em segundo
	// plano, a busca NUNCA esta pronta neste ponto, e um campo que so pode
	// valer false nao informa nada — informa errado, porque parece medir algo.
	// Quando a busca fica pronta, "indice de busca pronto" diz quando e por que
	// caminho.
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
