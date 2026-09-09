package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jonyd/gobsidian/internal/boot"
	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/instalar"
	"github.com/jonyd/gobsidian/internal/ipc"
	"github.com/jonyd/gobsidian/internal/lifecycle"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/jonyd/gobsidian/internal/service"
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
			flags.MaxResultsSet = cmd.Flags().Changed("max-results")

			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}
			return runServe(cmd.Context(), cfg)
		},
	}

	flagsDeCofre(cmd, &flags)
	flagsDeCache(cmd, &flags)
	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "desabilita toda a superficie de escrita")
	cmd.Flags().IntVar(&flags.DebounceMS, "debounce-ms", 0, "janela de coalescencia de eventos do watcher")
	cmd.Flags().IntVar(&flags.MaxResults, "max-results", 0, "teto de resultados por consulta")
	cmd.Flags().BoolVar(&flags.EagerSearch, "eager-search", false,
		"carrega o indice de busca no boot em vez de esperar a primeira vault_search")

	return cmd
}

// shutdownExitCode traduz o erro final do serve loop em codigo de saida.
//
// Existe como funcao separada porque e a unica parte de runServe que da para
// testar sem levantar um processo: runServe termina em os.Exit por desenho, e
// os.Exit nao volta.
//
// ipc.EhDesconexaoLimpa e a conta unica de "o outro lado foi embora" — ver
// esse comentario para o que cada forma significa.
func shutdownExitCode(err error) int {
	if ipc.EhDesconexaoLimpa(err) {
		return 0
	}
	return 1
}

// runServe e o ponto de entrada do comando `serve`. Cria o logger — o unico
// que existe no processo inteiro, ver a nota sobre stdout abaixo — e entrega
// a decisao entre falar com um daemon via socket ou servir o cofre neste
// processo para servePonte (ponte.go). runServe nao retorna: termina o
// processo aqui para que o codigo de saida seja o desta decisao e nao o que
// cobra derivaria de um error. O return no fim e inalcancavel e existe so
// para satisfazer a assinatura que RunE exige.
func runServe(parent context.Context, cfg config.Config) error {
	// stderr, sempre. stdout carrega o JSON-RPC e um unico byte estranho
	// corrompe a sessao.
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.LogLevel}))

	// NADA de I/O aqui. A trava de instalacao e a presenca rodam depois de
	// boot.VigiarHost, dentro de serveEmProcesso e de servePonteRemota -- ver
	// prepararProcesso e o comentario dela para o defeito que colocou as duas
	// aqui e o que ele custou.
	codigo := shutdownExitCode(servePonte(parent, cfg, log))

	// Antes do os.Exit, e nao por defer: defer nao roda depois de os.Exit.
	// Sem esta linha o arquivo de presenca fica para sempre -- medido em
	// 2026-09-09, 100 ciclos do gate de orfaos levaram o diretorio de runtime
	// de 6 para 130 arquivos.
	instalar.LiberarPresenca()

	os.Exit(codigo)
	return nil
}

// prepararProcesso faz o que este processo precisa do instalador: recusar subir
// durante uma instalacao, e anunciar a propria presenca.
//
// Devolve true quando o processo deve SAIR sem servir.
//
// # Por que aqui, e nao no comeco de runServe
//
// As duas chamadas fazem I/O -- criar diretorio, abrir arquivo, pedir trava do
// kernel, gravar JSON, fsync. Na primeira versao elas rodavam em runServe,
// ANTES de boot.VigiarHost, que e onde lifecycle.New instala o tratador de
// sinal. Um sinal que chegasse nessa janela nao tinha tratador: o processo
// morria pela acao padrao, sem registrar "reason=".
//
// Medido no CI em 2026-09-09: o cenario `signal` do gate de orfaos reprovou com
// "2 de 100 ciclos encerraram sem registrar reason=", nas duas rodadas, e o
// mesmo job estava verde no commit anterior (26bb00d). O harness manda o sinal
// ~50-150 ms depois de lancar o processo, e o I/O que eu tinha acrescentado
// cabia dentro disso num runner carregado.
//
// A invariante, que vale para qualquer coisa que venha depois: NADA roda antes
// de os mecanismos de encerramento estarem armados. Quem precisa de I/O na
// partida faz depois de VigiarHost.
func prepararProcesso(log *slog.Logger, papel, cofre string) (sair bool) {
	if recusarDuranteInstalacao(log, papel) {
		return true
	}

	// A presenca e o que responde "quem esta servindo este cofre agora?".
	//
	// Em 2026-09-08 havia dois processos servindo o cofre Estudo e gravando o
	// mesmo inverted_cache.gob, e a unica forma de descobrir isso foi comparar
	// milissegundos entre linhas de log duplicadas.
	//
	// Falha ao registrar NAO impede servir: presenca e diagnostico, e um
	// diretorio de runtime inacessivel nao pode derrubar o servidor.
	if dir, err := instalar.DiretorioDeRuntime(); err == nil {
		if err := instalar.RegistrarAteMorrer(dir, cofre, papel, version); err != nil {
			log.Debug("nao foi possivel registrar presenca", "err", err)
		}
	}
	return false
}

// recusarDuranteInstalacao faz o processo sair na hora se houver uma instalacao
// em curso, e diz por que.
//
// Sai com codigo ZERO, de proposito. Instalacao em curso nao e falha do
// servidor, e um codigo de erro faria o host MCP tentar de novo em laco durante
// os segundos da troca -- transformando uma pausa curta numa tempestade de
// processos que o instalador teria de encerrar um a um.
//
// Erro ao consultar a trava devolve false (ver instalar.InstalacaoEmCurso): nao
// poder perguntar nao pode impedir o produto de subir.
func recusarDuranteInstalacao(log *slog.Logger, papel string) bool {
	dir, err := instalar.DiretorioDeRuntime()
	if err != nil {
		return false
	}
	if !instalar.InstalacaoEmCurso(dir) {
		return false
	}
	log.Warn("instalacao em curso; este processo nao vai subir",
		"papel", papel,
		"trava", filepath.Join(dir, instalar.NomeDaTravaGlobal),
		"o_que_fazer", "o host vai reiniciar o servidor sozinho quando a instalacao terminar")
	return true
}

// serveEmProcesso monta o indice, o watcher e o servidor MCP dentro deste
// processo — o caminho que o servidor sempre usou, antes de existir ponte
// nenhuma. E tambem o fallback obrigatorio de servePonte (ponte.go) quando
// nao ha daemon para conversar: socket ausente, conexao recusada ou versao
// incompativel caem todos aqui.
func serveEmProcesso(parent context.Context, cfg config.Config, log *slog.Logger) error {
	// O monitor de stdin consome bytes, e o stdin aqui pertence ao JSON-RPC.
	// A saida e espelhar: o SDK le de vig.Stdin, e o lifecycle observa a
	// copia. boot.VigiarHost monta o andaime — pipe, espelho e lifecycle.New,
	// nessa ordem — que este caminho e a ponte (ponte.go) compartilham.
	ctx, vig := boot.VigiarHost(parent, os.Stdin, log)

	// O guarda-chuva cobre o encerramento INTEIRO, e nao so Shutdown.
	//
	// internal/boot/busca.go:209 ja registrava o buraco em letra: "serveEmProcesso
	// faz a espera das goroutines de fundo DEPOIS de lifecycle.Shutdown, entao
	// essa espera nao passa por orcamento nenhum -- e o harness de orfaos conta
	// como orfao o que nao morre em 8 s". c.Esperar(), abaixo, e essa espera.
	//
	// O daemon tinha o mesmo buraco e foi la que ele custou: PID 42856 vivo 20 h
	// depois de pedir encerramento (2026-09-07). A regra vale nos tres pontos de
	// saida do processo, inclusive nos que ainda nao falharam.
	defer lifecycle.ArmarGuardaChuva(ctx, log, lifecycle.OrcamentoDeEncerramento)()

	if prepararProcesso(log, "serve", cfg.VaultPath) {
		return nil
	}

	// boot.Montar monta o indice, o watcher e o servico de dominio -- a mesma
	// sequencia que o daemon (internal/daemon + cmd/gobsidian/daemon.go,
	// Task 92) usa para servir N conexoes em vez de uma. Extraida para as
	// duas nunca divergirem (ver o comentario do pacote em internal/boot).
	c, err := boot.Montar(ctx, cfg, service.ModoEmProcesso, log)
	if err != nil {
		// O irmao do ramo equivalente em runDaemon. Vale mesmo tendo aqui um
		// stderr com leitor: o fallback em processo e obrigatorio quando o
		// daemon nao sobe, e se os DOIS caminhos de boot morrerem calados um
		// cofre mal configurado nao produz mensagem acionavel em lugar
		// nenhum. Foi o que aconteceu com um dos cofres do dono por dois dias.
		log.Error("nao foi possivel montar o servico",
			"vault", cfg.VaultPath, "err", err)
		return err
	}

	srv := mcpsrv.New(ctx, c.Service, cfg, log)

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ctx, vig.Stdin, os.Stdout) }()

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
	// logo depois de vig.LC.Wait(). Um canal com buffer 1 carrega a garantia de
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

	lifecycle.Shutdown(ctx, log, lifecycle.OrcamentoDeEncerramento,
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
		vig.PassoFecharEspelho(),
		c.PassoWatcher(),
	)

	lifecycle.Esperar(ctx, log, "lifecycle", vig.LC.Wait)
	lifecycle.Esperar(ctx, log, "goroutines-de-fundo", c.Esperar)

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

	// Diferente do runServe de antes da ponte (Task 91), esta funcao AGORA
	// retorna em vez de terminar o processo: quem decide o codigo de saida e
	// runServe, um nivel acima, porque e o mesmo lugar que decide entre este
	// caminho e servePonteRemota — os dois precisam do mesmo shutdownExitCode
	// aplicado ao MESMO loopErr, e duplicar a chamada a os.Exit nos dois
	// caminhos teria dado a cada um seu proprio codigo de saida.
	return loopErr
}
