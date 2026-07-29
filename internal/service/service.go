package service

import (
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Index e a dependencia do servico sobre o indice, declarada como interface
// para que o servico seja testavel sem construir um indice completo, e para
// que M1 possa injetar a implementacao real sem tocar aqui.
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
// So o que muda o comportamento dele entra aqui; o resto da Config fica
// onde esta.
type Options struct {
	ReadOnly   bool
	MaxResults int
}

// Service e a fachada unica sobre os subsistemas: cada tool MCP corresponde
// a um metodo daqui. Nenhum tipo do SDK de MCP entra nesta struct, e e essa
// fronteira que torna uma quebra de protocolo mudanca de um pacote so.
type Service struct {
	vault   *vault.Vault
	index   Index
	watcher WatchStats
	opts    Options
}

// New monta o servico. idx pode ser nulo em M0, quando o indice ainda nao
// existe e as contagens saem de uma varredura do disco.
func New(v *vault.Vault, idx Index, w WatchStats, opts Options) *Service {
	return &Service{vault: v, index: idx, watcher: w, opts: opts}
}
