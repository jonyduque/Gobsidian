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
	Generation() uint64
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
	vault *vault.Vault
	index Index
	opts  Options
}

// New monta o servico. idx pode ser nulo em M0, quando o indice ainda nao
// existe e as contagens saem de uma varredura do disco.
func New(v *vault.Vault, idx Index, opts Options) *Service {
	return &Service{vault: v, index: idx, opts: opts}
}
