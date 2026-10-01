package boot

import (
	"context"
	"log/slog"

	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/search"
	"github.com/jonyduque/Gobsidian/internal/service"
	"github.com/jonyduque/Gobsidian/internal/vault"
)

// AbrirServicoDeCLI monta o servico que um comando de CLI usa para chamar uma
// tool: cofre, indice de metadados (do cache quando fresco, que e o que torna
// a chamada rapida) e busca PREGUICOSA -- a mesma montagem de Montar sem
// --eager-search, sem watcher.
//
// Sem watcher porque o comando vive o tempo de uma chamada: nao ha o que
// vigiar. A busca so carrega se a tool for vault_search, pelo CarregarBusca
// que o servico ja sabe usar; as outras treze nunca pagam por ela.
//
// fechar solta o indice invertido; quem chama o adia.
func AbrirServicoDeCLI(ctx context.Context, cfg config.Config, log *slog.Logger) (svc *service.Service, fechar func(), err error) {
	v, err := vault.New(cfg.VaultPath, vault.SeguirSymlinks(cfg.FollowSymlinks))
	if err != nil {
		return nil, nil, err
	}
	idx, origem, err := AbrirIndice(ctx, v, cfg, log)
	if err != nil {
		return nil, nil, err
	}
	// Em Info, que a CLI cala sem --log-level: e o que diz a quem mede se a
	// chamada pagou a varredura ou leu o cache.
	log.Info("indice de metadados aberto", "origem", origem, "notas", idx.NoteCount())

	// Vazio e marcado em construcao, como em Montar: quem consultar antes da
	// carga recebe INDEX_BUILDING, e nao zero resultados.
	inv := search.NewInverted()
	inv.MarkBuilding()

	svc = service.New(v, idx, inv, nil, service.Options{
		ReadOnly:   cfg.ReadOnly,
		MaxResults: cfg.MaxResults,
		CacheDir:   cfg.CacheDir,
		CarregarBusca: func(buscaCtx context.Context) error {
			PrepararBusca(buscaCtx, v, idx, inv, cfg, log)
			return buscaCtx.Err()
		},
	})
	return svc, func() { _ = inv.Close() }, nil
}
