// Package watcher observa o cofre e mantem o indice atualizado sem reindexar
// tudo. Sao tres camadas de filtragem antes de qualquer trabalho real —
// relevancia, coalescencia por janela, e comparacao de mtime e tamanho — mais
// a reconciliacao que cobre o caso em que eventos se perderam. Nenhum tipo do
// fsnotify cruza para fora daqui.
package watcher

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Apply processa o canal de caminhos coalescidos, aplicando as mudanças no índice.
// A comparação de mtime e tamanho não é uma prova de que o conteúdo não mudou.
// Uma reescrita dentro do mesmo tique de mtime, preservando o tamanho, passa despercebida.
// Isso é aceitável porque há dois anteparos: a reconciliação por overflow (Task 30)
// e a reindexação no boot.
func Apply(ctx context.Context, in <-chan []vault.CanonicalPath, reconcile <-chan struct{}, idx *index.Index, v *vault.Vault, log *slog.Logger, processed, skipped, reconciledUpdated, reconciledRemoved *atomic.Int64, searchInv *search.Inverted) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-reconcile:
			up, rem, _ := Reconcile(ctx, v, idx, searchInv, log)
			if reconciledUpdated != nil {
				reconciledUpdated.Add(int64(up))
			}
			if reconciledRemoved != nil {
				reconciledRemoved.Add(int64(rem))
			}
		case batch, ok := <-in:
			if !ok {
				return
			}

			// Task 31: Correlate renames within the batch
			renames, nonRenames := CorrelateRenames(ctx, batch, v, idx, log)

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
				idx.MoveNote(v, rename.From, rename.To)
				if searchInv != nil {
					searchInv.Remove(string(rename.From))
					if err := searchInv.Update(ctx, v, rename.To); err != nil {
						log.Warn("falha ao atualizar indice invertido no rename", "path", rename.To, "err", err)
					}
				}
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
						if searchInv != nil {
							searchInv.Remove(string(path))
						}
					} else {
						log.Warn("Erro ao ler os.Stat, pulando mudança para evitar deleção indevida do índice", "path", abs, "err", err)
					}
					continue
				}

				// Verifica se já está indexado e se mtime/tamanho bateram.
				// Só compara para notas (pois idx.Get resolve apenas notas).
				//
				// A condição inclui o índice de BUSCA, e não só o de metadados.
				// A reconciliação por overflow já exigia isso (overflow.go:58);
				// este atalho não a espelhava, e eram duas cópias de uma regra
				// com a errada no caminho mais usado.
				//
				// O que a assimetria custava: um único searchInv.Update falho
				// — que só produz log.Warn, logo abaixo — deixa os metadados em
				// dia e a posting ausente. A partir daí TODO evento com mtime e
				// tamanho iguais caía aqui, e o índice de busca nunca se
				// recompunha. O OneDrive re-emite evento de arquivo intocado
				// como rotina, então o gatilho não é raro.
				//
				// HasDoc não lê disco: o custo do atalho continua em memória.
				if n, ok := idx.Get(path); ok && (searchInv == nil || searchInv.HasDoc(string(path))) {
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
				} else if searchInv != nil {
					if err := searchInv.Update(ctx, v, path); err != nil {
						log.Warn("falha ao atualizar indice invertido", "path", path, "err", err)
					}
				}
			}
		}
	}
}
