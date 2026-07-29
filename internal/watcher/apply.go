package watcher

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Apply processa o canal de caminhos coalescidos, aplicando as mudanças no índice.
// A comparação de mtime e tamanho não é uma prova de que o conteúdo não mudou.
// Uma reescrita dentro do mesmo tique de mtime, preservando o tamanho, passa despercebida.
// Isso é aceitável porque há dois anteparos: a reconciliação por overflow (Task 30)
// e a reindexação no boot.
func Apply(ctx context.Context, in <-chan []vault.CanonicalPath, reconcile <-chan struct{}, idx *index.Index, v *vault.Vault, log *slog.Logger, processed, skipped *atomic.Int64) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-reconcile:
			Reconcile(ctx, v, idx, log)
		case batch, ok := <-in:
			if !ok {
				return
			}

			// Task 31: Correlate renames within the batch
			renames, nonRenames := CorrelateRenames(batch, v, idx, log)

			// Process renames (report without rewriting content)
			for _, rename := range renames {
				if processed != nil {
					processed.Add(2)
				}
				log.Info("rename_correlated",
					"from", rename.From,
					"to", rename.To,
					"backlinks", len(rename.Backlinks),
				)
				idx.MoveNote(rename.From, rename.To)
			}

			for _, path := range nonRenames {
				if processed != nil {
					processed.Add(1)
				}
				abs := v.Abs(path)
				info, err := os.Stat(abs)

				if err != nil {
					if os.IsNotExist(err) {
						idx.Remove(path)
					} else {
						log.Warn("Erro ao ler os.Stat, pulando mudança para evitar deleção indevida do índice", "path", abs, "err", err)
					}
					continue
				}

				// Verifica se já está indexado e se mtime/tamanho bateram.
				// Só compara para notas (pois idx.Get resolve apenas notas).
				if n, ok := idx.Get(path); ok {
					if n.ModTime.Equal(info.ModTime()) && n.Size == info.Size() {
						if skipped != nil {
							skipped.Add(1)
						}
						continue
					}
				}

				err = idx.Replace(ctx, v, path)
				if err != nil {
					log.Warn("Falha ao reindexar arquivo", "path", path, "err", err)
				}
			}
		}
	}
}
