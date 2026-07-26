package service

import (
	"context"

	"github.com/jonyd/gobsidian/internal/vault"
)

// Index e a dependencia do servico sobre o indice, declarada como interface
// para que o servico seja testavel sem construir um indice completo, e para
// que M1 possa injetar a implementacao real sem tocar aqui.
type Index interface {
	NoteCount() int
	AssetCount() int
	TotalSize() int64
}

type Options struct {
	ReadOnly   bool
	MaxResults int
}

type Service struct {
	vault *vault.Vault
	index Index
	opts  Options
}

func New(v *vault.Vault, idx Index, opts Options) *Service {
	return &Service{vault: v, index: idx, opts: opts}
}

type StatsRequest struct {
	IncludeHealth  bool `json:"include_health"`
	IncludeRuntime bool `json:"include_runtime"`
}

type StatsResult struct {
	Notes     int   `json:"notes"`
	Assets    int   `json:"assets"`
	TotalSize int64 `json:"total_size"`
}

// VaultStats em M0 conta arquivos varrendo o disco. Em M1 passa a ler do
// indice; a assinatura nao muda, e o teste desta tarefa continua valendo.
func (s *Service) VaultStats(ctx context.Context, req StatsRequest) (StatsResult, error) {
	if s.index != nil {
		return StatsResult{
			Notes:     s.index.NoteCount(),
			Assets:    s.index.AssetCount(),
			TotalSize: s.index.TotalSize(),
		}, nil
	}

	var out StatsResult
	err := s.vault.Walk(ctx, func(e vault.Entry) error {
		if e.IsNote {
			out.Notes++
		} else {
			out.Assets++
		}
		out.TotalSize += e.Size
		return nil
	})
	if err != nil {
		return StatsResult{}, Wrap(CodeVaultUnavailable, err, "varrendo o cofre: %v", err)
	}
	return out, nil
}
