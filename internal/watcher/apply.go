package watcher

import (
	"context"
	"log/slog"
	"os"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Apply processa o canal de caminhos coalescidos, aplicando as mudanças no índice.
// A comparação de mtime e tamanho não é uma prova de que o conteúdo não mudou.
// Uma reescrita dentro do mesmo tique de mtime, preservando o tamanho, passa despercebida.
// Isso é aceitável porque há dois anteparos: a reconciliação por overflow (Task 30)
// e a reindexação no boot.
func Apply(ctx context.Context, in <-chan vault.CanonicalPath, idx *index.Index, v *vault.Vault, log *slog.Logger) (processed, skipped int64) {
	for {
		select {
		case <-ctx.Done():
			return processed, skipped
		case path, ok := <-in:
			if !ok {
				return processed, skipped
			}

			processed++
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
					skipped++
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
