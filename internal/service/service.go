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

// StatsRequest sao os dois recortes opcionais de vault_stats. Ambos custam
// caro — saude percorre o grafo de links, runtime le contadores do watcher —
// e por isso ficam desligados por padrao.
type StatsRequest struct {
	IncludeHealth  bool `json:"include_health"`
	IncludeRuntime bool `json:"include_runtime"`
}

// StatsResult e a resposta de vault_stats. As tags json nao sao decorativas:
// esta struct vira o structuredContent que o cliente le primeiro, entao os
// nomes dos campos sao contrato publico.
type StatsResult struct {
	Notes     int   `json:"notes"`
	Assets    int   `json:"assets"`
	TotalSize int64 `json:"total_size"`
}

// VaultStats em M0 conta arquivos varrendo o disco. Em M1 passa a ler do
// indice; a assinatura nao muda, e o teste desta tarefa continua valendo.
//
// O segundo parametro esta como `_` porque M0 nao le nenhum campo dele. A
// assinatura ja o declara para que M1 passe a honrar IncludeHealth e
// IncludeRuntime sem quebrar chamador nenhum; ali ele volta a se chamar req.
func (s *Service) VaultStats(ctx context.Context, _ StatsRequest) (StatsResult, error) {
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
		return StatsResult{}, Wrap(CodeVaultUnavailable, err, "varrendo o cofre")
	}
	return out, nil
}
