package boot

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/index"
	"github.com/jonyduque/Gobsidian/internal/vault"
)

// AbrirIndice devolve o indice de metadados: do cache, se existir e estiver
// fresco, ou construido do cofre e gravado. origem e "cache" ou "build" -- e o
// valor que sai no campo index_origin do log "servidor pronto".
func AbrirIndice(ctx context.Context, v *vault.Vault, cfg config.Config, log *slog.Logger) (*index.Index, string, error) {
	if idx, ok := carregarIndiceDoCache(ctx, v, cfg, log); ok {
		return idx, "cache", nil
	}
	idx := index.New()
	if err := idx.Build(ctx, v); err != nil {
		return nil, "", err
	}
	if err := index.SaveIndexCache(ctx, cfg.CacheDir, cfg.VaultPath, idx); err != nil {
		log.Warn("falha ao salvar cache de indice de metadados", "err", err)
	}
	return idx, "build", nil
}

// carregarIndiceDoCache tenta servir o indice de metadados do disco em vez
// de varrer e parsear o cofre inteiro — o que fecha docs/PRD.md Q3: o cache
// so existia para o indice invertido (busca), e RNF-02 nunca era atingido
// porque o indice de metadados sempre reconstruia do zero, mesmo com o
// cache de busca quente.
//
// So aceita o cache quando VerifyFreshness confirma que ele bate com o
// disco — mesma contagem de arquivos, mesmo tamanho e mtime por arquivo.
// Qualquer divergencia (nota editada offline entre partidas, nota nova,
// cache ausente, corrompido ou de versao incompativel) devolve ok=false, e
// quem chamou cai para idx.Build, que sempre produz o indice correto. Nao
// ha reparo parcial aqui de proposito: reparar so o que mudou e o trabalho
// da reconciliacao do watcher (RF-05), nao do boot.
func carregarIndiceDoCache(ctx context.Context, v *vault.Vault, cfg config.Config, log *slog.Logger) (*index.Index, bool) {
	doCache, hdr, err := index.LoadIndexCache(ctx, cfg.CacheDir, cfg.VaultPath)
	if err != nil {
		if !errors.Is(err, index.ErrIndexCacheNotFound) {
			log.Warn("cache de indice de metadados descartado", "err", err)
		}
		return nil, false
	}

	fresh, ferr := doCache.VerifyFreshness(ctx, v)
	if ferr != nil {
		log.Warn("verificacao de atualidade do cache de indice de metadados falhou", "err", ferr)
		return nil, false
	}
	if !fresh {
		log.Info("cache de indice de metadados desatualizado; reconstruindo", "no_cache", hdr.NoteCount)
		return nil, false
	}

	return doCache, true
}
