package service

import (
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/writer"
)

// Index e a dependencia do servico sobre o indice.
type Index interface {
	NoteCount() int
	AssetCount() int
	TotalSize() int64
	ResolvePath(input string) (vault.CanonicalPath, error)
	Get(path vault.CanonicalPath) (*index.Note, bool)
	List(q index.Query) ([]*index.Note, int)
	Tags(prefix string, minCount int) []index.TagCount
	Backlinks(path vault.CanonicalPath) []index.Backlink
	Paths() []vault.CanonicalPath
	NotePaths() []vault.CanonicalPath
	AliasCollisions() int
	Generation() uint64
}

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

// Service e a fachada unica sobre os subsistemas: cada tool MCP corresponde a um metodo daqui.
type Service struct {
	vault    *vault.Vault
	index    Index
	inverted *search.Inverted
	watcher  WatchStats
	opts     Options
	locker   *writer.PathLocker
	trechos  *search.SnippetCache

	carregarBusca CarregadorBusca
	cargaBusca    cargaUnica
}

// New monta o servico.
func New(v *vault.Vault, idx Index, inv *search.Inverted, w WatchStats, opts Options) *Service {
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

// Inverted devolve a instância do índice invertido.
func (s *Service) Inverted() *search.Inverted {
	return s.inverted
}
