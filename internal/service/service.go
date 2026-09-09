package service

import (
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/writer"
)

// WatchCounters reporta a saude e os contadores do watcher em tempo de execucao.
type WatchCounters struct {
	Active            bool             `json:"active"`
	EventsReceived    int64            `json:"events_received"`
	EventsDropped     int64            `json:"events_dropped"`
	DroppedByReason   map[string]int64 `json:"events_dropped_by_reason"`
	EventsCoalesced   int64            `json:"events_coalesced"`
	EventsProcessed   int64            `json:"events_processed"`
	EventsSkipped     int64            `json:"events_skipped"`
	Reconciliations   int64            `json:"reconciliations"`
	ReconciledUpdated int64            `json:"reconciled_updated"`
	ReconciledRemoved int64            `json:"reconciled_removed"`
}

// WatchStats representa o subsistema do watcher, capaz de reportar seus contadores.
type WatchStats interface {
	Stats() WatchCounters
}

// Options sao as decisoes de configuracao que o servico precisa conhecer.
type Options struct {
	ReadOnly   bool
	MaxResults int

	// Modo diz COMO este servico esta sendo servido: "daemon" (uma instancia
	// para N sessoes) ou "em-processo" (uma instancia por sessao).
	//
	// O pacote service nao tem como descobrir isso sozinho -- quem decide e
	// cmd/gobsidian --, e a diferenca importa para quem diagnostica: em
	// 2026-09-08 dois processos serviam o mesmo cofre e gravavam o MESMO cache
	// de busca, e isso so ficou visivel comparando milissegundos entre linhas
	// de log duplicadas. Vazio sai como "desconhecido".
	Modo string

	// CacheDir e o diretorio de cache deste cofre, reportado por vault_stats.
	// Duas instancias com o mesmo CacheDir sao duas instancias disputando os
	// mesmos arquivos.
	CacheDir string
	// CarregarBusca, quando nao nil, adia o carregamento do indice
	// invertido para a primeira chamada de Search. Quem monta o servico
	// (cmd/gobsidian) e quem sabe COMO carregar — cache em disco, cofre,
	// config — o pacote service so sabe QUANDO: uma vez, na primeira busca,
	// com retentativa se aquela tentativa falhar. Ver garanteIndiceDeBusca.
	//
	// nil preserva o comportamento antigo: o indice chega pronto (ou
	// marcado em construcao por outro caminho, em segundo plano) e Search
	// nunca dispara carga nenhuma.
	CarregarBusca CarregadorBusca

	// SnippetCacheEntries e o teto do cache de trechos de vault_search.
	//
	// E ponteiro pela mesma razao de config.ReadOnlySet existir: num int, zero
	// e "omitido" e "desligado" ao mesmo tempo, e a diferenca importa. nil usa
	// search.DefaultSnippetCacheEntries; um zero explicito DESLIGA o cache, e e
	// assim que o benchmark mede o caminho frio, que e o que o RNF-04 cobra.
	SnippetCacheEntries *int
}

// Service e a fachada das tools sobre o dominio. Recebe o indice concreto:
// a interface que existia aqui tinha uma implementacao e nenhum fake, e
// search.CalculateBM25 e search.GenerateSnippet exigem *index.Index, o que
// obrigava uma assercao de tipo em cada busca.
type Service struct {
	vault    *vault.Vault
	index    *index.Index
	inverted *search.Inverted
	watcher  WatchStats
	opts     Options
	locker   *writer.PathLocker
	trechos  *search.SnippetCache

	carregarBusca CarregadorBusca
	cargaBusca    cargaUnica
}

// New monta o servico.
func New(v *vault.Vault, idx *index.Index, inv *search.Inverted, w WatchStats, opts Options) *Service {
	entradasTrecho := search.DefaultSnippetCacheEntries
	if opts.SnippetCacheEntries != nil {
		entradasTrecho = *opts.SnippetCacheEntries
	}
	return &Service{
		vault:         v,
		index:         idx,
		inverted:      inv,
		watcher:       w,
		opts:          opts,
		locker:        writer.NewPathLocker(),
		trechos:       search.NewSnippetCache(entradasTrecho),
		carregarBusca: opts.CarregarBusca,
	}
}

// ModoDaemon e ModoEmProcesso sao os dois valores que Options.Modo aceita.
//
// Constantes, e nao literais nos pontos de uso: elas atravessam o retorno de
// vault_stats, que e contrato publico (docs/TOOLS.md), e um valor escrito a mao
// em dois lugares diverge no dia em que um deles muda.
const (
	ModoDaemon       = "daemon"
	ModoEmProcesso   = "em-processo"
	ModoDesconhecido = "desconhecido"
)

// modo devolve o modo declarado, ou "desconhecido".
//
// Nunca vazio: um campo que as vezes some faz quem le acreditar que a
// informacao nao existe, quando ela so nao foi preenchida -- a mesma razao pela
// qual Orphans e ponteiro em vez de int com omitempty.
func (s *Service) modo() string {
	if s.opts.Modo == "" {
		return ModoDesconhecido
	}
	return s.opts.Modo
}
