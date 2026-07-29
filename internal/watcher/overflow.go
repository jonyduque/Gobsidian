package watcher

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Em macOS e BSD esta funcao nunca e chamada: o backend kqueue do fsnotify
// v1.10.1 nao emite ErrEventOverflow (so backend_inotify.go:398 e
// backend_windows.go:582 emitem). La o unico anteparo contra evento perdido
// e a reindexacao no boot. Lacuna registrada, nao resolvida por heuristica.
// Reconcile faz uma varredura completa do cofre e compara com o índice,
// aplicando index.Replace para arquivos modificados/novos e index.Remove para deletados.
func Reconcile(ctx context.Context, v *vault.Vault, idx *index.Index, log *slog.Logger) (updated, removed, skipped int) {
	log.Info("Iniciando reconciliação completa do cofre devido a overflow")

	visited := make(map[vault.CanonicalPath]struct{})

	err := v.Walk(ctx, func(e vault.Entry) error {
		visited[e.Path] = struct{}{}

		abs := v.Abs(e.Path)
		info, statErr := os.Stat(abs)
		if statErr != nil {
			skipped++
			return nil
		}

		// Verifica se já está no índice e se está atualizado
		if n, ok := idx.Get(e.Path); ok {
			if n.ModTime.Equal(info.ModTime()) && n.Size == info.Size() {
				return nil
			}
		}

		if replaceErr := idx.Replace(ctx, v, e.Path); replaceErr != nil {
			skipped++
		} else {
			updated++
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Debug("Reconciliação interrompida pelo shutdown", "updated", updated, "removed", removed, "skipped", skipped)
			return updated, removed, skipped
		}
		log.Error("Erro durante varredura de reconciliação", "err", err)
		return updated, removed, skipped
	}

	// Remove arquivos do índice que não foram visitados no disco
	for _, indexedPath := range idx.Paths() {
		if _, ok := visited[indexedPath]; !ok {
			idx.Remove(indexedPath)
			removed++
		}
	}

	log.Warn("Reconciliação concluída", "updated", updated, "removed", removed, "skipped", skipped)
	return updated, removed, skipped
}
