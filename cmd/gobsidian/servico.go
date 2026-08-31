// servico.go extrai a montagem do indice, do watcher e do servico de
// dominio -- compartilhada entre serveEmProcesso (Task 91, serve.go) e o
// daemon (Task 92, daemon.go + internal/daemon). As duas precisam da MESMA
// sequencia de boot: cache do indice de metadados, varredura de
// temporarios, watcher.New ANTES da construcao do indice invertido, e o
// carregamento em segundo plano do cache de busca. Extrair para uma funcao
// so evita a classe de divergencia que o CLAUDE.md registra para "chave de
// mapa calculada em dois lugares": aqui seria "sequencia de boot construida
// em dois lugares", com o mesmo risco de um dos dois ficar para tras quando
// o outro mudar sozinho.
package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/watcher"
	"github.com/jonyd/gobsidian/internal/writer"
)

// servicoMontado agrupa o que construirServico produz: o servico de
// dominio pronto para um *mcpsrv.Server, o watcher (para Stats() e para
// ser fechado no encerramento) e o WaitGroup que acompanha as goroutines de
// fundo (construcao do indice de busca e o loop do watcher) -- quem chama
// tem de esperar por ele APOS o shutdown, do mesmo jeito que serveEmProcesso
// ja fazia antes desta extracao.
type servicoMontado struct {
	svc *service.Service
	w   *watcher.Watcher
	wg  *sync.WaitGroup
}

// construirServico monta o indice de metadados, o watcher e o indice de
// busca (em segundo plano) para cfg.VaultPath, e devolve o servico de
// dominio pronto para ser exposto por um *mcpsrv.Server -- uma sessao
// (serveEmProcesso) ou N sessoes sobre um socket (o daemon).
//
// ctx aqui e o context do CHAMADOR (pos-lifecycle): a goroutine de fundo
// que esta funcao lanca (construcao do indice de busca + watcher.Run) roda
// por baixo dele e para quando ele for cancelado, exatamente como antes
// desta extracao.
func construirServico(ctx context.Context, cfg config.Config, log *slog.Logger) (*servicoMontado, error) {
	v, err := vault.New(cfg.VaultPath, vault.SeguirSymlinks(cfg.FollowSymlinks))
	if err != nil {
		return nil, err
	}

	// A duracao da indexacao a frio e RNF-01, e ate aqui ninguem a media:
	// quem quisesse o numero cronometrava o processo inteiro por fora,
	// misturando boot do Go e handshake do MCP com o que o alvo cobre.
	// Logar aqui torna a medicao reproduzivel e recorta exatamente o
	// trecho que o requisito nomeia.
	// A varredura de temporarios roda EM PARALELO com a carga do indice, e nao
	// depois dela (achado P15).
	//
	// Ela percorre o cofre inteiro, e estava no caminho critico antes de montar
	// watcher e servico — recuperacao de crash que nao e pre-requisito para
	// responder o initialize. Numa partida simultanea de varias instancias eram
	// N varreduras seriais disputando disco antes do primeiro byte.
	//
	// A garantia documentada em writer/atomic.go continua valendo: "o unico
	// lugar sem escrita em voo e o boot". As duas metades do boot continuam
	// antes de o servidor servir, e o join abaixo acontece antes de watcher.New
	// — nada que escreva chegou a existir ainda.
	type varreduraFeita struct {
		res writer.SweepResult
		err error
	}
	varredura := make(chan varreduraFeita, 1)
	go func() {
		r, err := writer.SweepStaleTempFiles(ctx, cfg.VaultPath)
		varredura <- varreduraFeita{res: r, err: err}
	}()

	buildStart := time.Now()
	idx, usouCache := carregarIndiceDoCache(ctx, v, cfg, log)
	indexOrigin := "cache"
	if !usouCache {
		indexOrigin = "build"
		idx = index.New()
		if err := idx.Build(ctx, v); err != nil {
			return nil, err
		}
		if err := index.SaveIndexCache(ctx, cfg.CacheDir, cfg.VaultPath, idx); err != nil {
			log.Warn("falha ao salvar cache de indice de metadados", "err", err)
		}
	}
	indexMS := time.Since(buildStart).Milliseconds()

	// JOIN da varredura, antes de qualquer coisa que escreva.
	//
	// Recuperacao de crash: um processo morto no meio de uma escrita nao roda
	// defer, e deixa o temporario no cofre. O boot e o unico momento sem
	// escrita em voo, e por isso o unico em que varrer o diretorio nao corre
	// risco de apagar o temporario de outra escrita. Ver
	// writer.CleanStaleTempFiles.
	feita := <-varredura
	if varr, err := feita.res, feita.err; err != nil {
		log.Warn("varredura de temporarios interrompida", "err", err)
	} else if varr.Removidos > 0 || varr.NaoRemovidos > 0 || varr.Inacessiveis > 0 {
		// Os tres numeros, e nao so os removidos: "varri e nao achei nada" e
		// "varri e nao consegui entrar em trinta diretorios" davam a mesma
		// linha de log — nenhuma (achado P11).
		log.Warn("varredura de temporarios de escritas interrompidas",
			"removidos", varr.Removidos,
			"nao_removidos", varr.NaoRemovidos,
			"subarvores_inacessiveis", varr.Inacessiveis)
	}

	// Nem a construcao NEM o carregamento do cache bloqueiam o anuncio das
	// tools.
	//
	// A construcao custa proporcionalmente aos BYTES do cofre, nao a
	// contagem de notas: num cofre real de 109 MB a tokenizacao levou
	// 219 s, contra 1,3 s do indice de metadados. O host desiste do
	// handshake MCP em 30 s e mata o processo — antes de SaveInvertedCache
	// rodar, entao a tentativa seguinte recomecava do zero. Toda tentativa
	// falhava pelo mesmo motivo, para sempre, e nada no log dizia isso:
	// "servidor pronto" so aparecia depois.
	//
	// O carregamento do cache saiu do caminho de boot pelo mesmo
	// raciocinio, aplicado a uma escala menor: medido em 1,83 s no cofre
	// de referencia, contra 1,3 s do indice de metadados. Nao e o
	// suficiente para estourar o handshake, mas era mais da metade do
	// tempo ate o servidor responder, gasto numa estrutura que nenhuma
	// tool precisa para ser anunciada.
	//
	// O indice entregue aqui esta VAZIO e marcado como em construcao.
	// Quem consulta a busca nesse intervalo recebe INDEX_BUILDING, e nao
	// zero resultados: cofre sem a palavra e indice ainda sem a palavra
	// nao podem produzir a mesma resposta.
	//
	// Este paragrafo descrevia SO o modo eager ate 2026-08-26, e a
	// auditoria acusou a lacuna (A6). No modo preguicoso — o padrao — o
	// indice tambem chega vazio, mas a carga so dispara na primeira
	// vault_search. Ate entao quem chegava durante ela esperava num mutex
	// PURO, sem prazo: com cache frio a tokenizacao do cofre inteiro roda
	// por minutos sem resposta e sem erro. Hoje a espera e uma porta com
	// select em ctx.Done() (ver cargaUnica em internal/service), quem
	// desiste recebe INDEX_BUILDING, e a carga segue em segundo plano —
	// amarra-la ao primeiro chamador faria a busca seguinte recomecar do
	// zero.
	inv := search.NewInverted()
	inv.MarkBuilding()

	// watcher.New ANTES da construcao, e w.Run depois dela.
	//
	// New registra os watches, entao os eventos do periodo de construcao
	// ficam enfileirados no fsnotify em vez de se perderem; Run os
	// consome depois e reindexa com o conteudo corrente do disco, que e
	// no minimo tao novo quanto o que a construcao leu. Sem isso, os dois
	// escreveriam no mesmo indice ao mesmo tempo e a construcao poderia
	// sobrescrever uma nota que o watcher acabara de atualizar.
	//
	// Se a fila estourar, o fsnotify emite ErrEventOverflow e o caminho de
	// reconciliacao por varredura completa assume — que e exatamente o
	// que ele existe para fazer.
	w, err := watcher.New(v, idx, inv, time.Duration(cfg.DebounceMS)*time.Millisecond, log)
	if err != nil {
		return nil, err
	}

	opts := service.Options{
		ReadOnly:   cfg.ReadOnly,
		MaxResults: cfg.MaxResults,
	}
	if !cfg.EagerSearch {
		// Modo padrao: a carga so acontece na primeira vault_search, e
		// dentro DELA — e o Service quem decide quando chamar isto (uma
		// vez, com retentativa se falhar; ver Service.garanteIndiceDeBusca).
		// O context recebido aqui e o da chamada MCP que disparou a carga,
		// nao o de boot: se aquele cliente especifico desistir no meio de
		// uma construcao longa, a proxima busca — com um context novo —
		// tenta de novo a partir do que HasDoc ja cobriu.
		//
		// Como o watcher.Run comeca no boot (ver comentario abaixo) e nao
		// espera este carregamento, uma edicao no cofre entre a partida e
		// a primeira busca pode chegar primeiro e escrever no indice
		// invertido. Nesse caso search.Inverted.AdotarDe recusa o cache
		// (indice nao vazio) e prepararIndiceDeBusca cai para
		// buildInvertedIndex, que ja conta o que o watcher escreveu via
		// HasDoc e so le do disco o resto — mais lento que o caminho de
		// cache, nunca incorreto.
		opts.CarregarBusca = func(searchCtx context.Context) error {
			prepararIndiceDeBusca(searchCtx, v, idx, inv, cfg, log)
			if err := searchCtx.Err(); err != nil {
				return err
			}
			return nil
		}
	}

	svc := service.New(v, idx, inv, watcherStats{w: w}, opts)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if cfg.EagerSearch {
			// prepararIndiceDeBusca ANTES de w.Run, e nao em paralelo com
			// ele.
			//
			// A adocao do cache SUBSTITUI o conteudo do indice (ver
			// search.Inverted.AdotarDe). Um evento do watcher aplicado
			// antes dela seria descartado sem deixar rastro, e o sintoma
			// seria uma nota editada durante o boot sumir da busca ate o
			// proximo reinicio. Os eventos desse intervalo nao se perdem:
			// watcher.New ja registrou os watches, entao eles ficam
			// enfileirados no fsnotify e w.Run os consome em seguida.
			prepararIndiceDeBusca(ctx, v, idx, inv, cfg, log)
		}
		// Modo padrao: o watcher comeca a consumir a fila do fsnotify sem
		// esperar o indice de busca. Adiar w.Run ate a primeira busca
		// faria eventos se perderem enquanto ninguem busca — e o unico
		// anteparo seria a reindexacao completa no proximo reinicio.
		_ = w.Run(ctx)
	}()

	// index_ms cobre SO o indice de metadados, e e o que RNF-01 nomeia.
	//
	// search_ready some daqui de proposito. Com o cache carregado em
	// segundo plano, a busca NUNCA esta pronta neste ponto, e um campo
	// que so pode valer false nao informa nada — informa errado, porque
	// parece medir algo. Quando a busca fica pronta, "indice de busca
	// pronto" diz quando e por que caminho.
	//
	// A mensagem "servidor pronto" e o campo "index_ms" NAO podem mudar de
	// texto: scripts/measure.ps1 faz parsing desta linha (grep por
	// "servidor pronto" e pela regex "index_ms=(\d+)") para medir RNF-01 —
	// ver o comentario la.
	log.Info("servidor pronto",
		"vault", cfg.VaultPath,
		"read_only", cfg.ReadOnly,
		"notes", idx.NoteCount(),
		"assets", idx.AssetCount(),
		"index_ms", indexMS,
		"index_origin", indexOrigin)

	return &servicoMontado{svc: svc, w: w, wg: &wg}, nil
}
