package watcher

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Reconcile faz uma varredura completa do cofre e compara com o indice,
// aplicando index.Replace para arquivos modificados ou novos e index.Remove
// para os que sumiram. Devolve quantos de cada, para os contadores.
//
// Em macOS e BSD esta funcao nunca e chamada: o backend kqueue do fsnotify
// v1.10.1 nao emite ErrEventOverflow (so backend_inotify.go:398 e
// backend_windows.go:582 emitem). La o unico anteparo contra evento perdido
// e a reindexacao no boot. Lacuna registrada, nao resolvida por heuristica.
// A reconciliacao repara AS DUAS estruturas: o indice de metadados e o de
// busca.
//
// Reparar so os metadados deixava a busca permanentemente errada, e em
// silencio. service.Search descarta a posting cujo caminho nao existe nos
// metadados (`s.index.Get` -> `if !ok { continue }`), entao uma nota MOVIDA
// durante o overflow ficava com os metadados no caminho novo e a posting no
// antigo: `vault_search` devolvia ZERO resultados para uma nota que estava la,
// para sempre, sem log e sem erro. Uma nota criada durante o overflow ficava
// invisivel pelo mesmo motivo, ao contrario.
//
// Reproduzido de forma deterministica antes da correcao: renomeando no disco
// sem evento nenhum e chamando so esta funcao, a busca devolvia 0.
func Reconcile(ctx context.Context, v *vault.Vault, idx *index.Index, inv *search.Inverted, log *slog.Logger) (updated, removed, skipped int) {
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

		// Verifica se já está nos DOIS índices e se está atualizado.
		//
		// A checagem de `inv` não é redundante: mtime e tamanho falam do
		// índice de metadados, e é justamente quando as duas estruturas
		// divergem que esta função existe. Sem ela, uma nota presente nos
		// metadados e ausente da busca era pulada por "já está atualizada" —
		// e a reconciliação, que é o anteparo, confirmava o defeito em vez de
		// consertá-lo. HasDoc não lê disco.
		if n, ok := idx.Get(e.Path); ok && (inv == nil || inv.HasDoc(string(e.Path))) {
			if n.ModTime.Equal(info.ModTime()) && n.Size == info.Size() {
				return nil
			}
		}

		if replaceErr := idx.Replace(ctx, v, e.Path); replaceErr != nil {
			skipped++
		} else {
			updated++
			if inv != nil {
				if err := inv.Update(ctx, v, e.Path); err != nil {
					log.Warn("falha ao reconciliar indice de busca", "path", e.Path, "err", err)
				}
			}
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

	// O mesmo para o índice de busca, e por sua própria lista.
	//
	// Percorrer idx.Paths() não bastaria: as duas estruturas podem ter
	// divergido — é a premissa desta função — e um caminho que sobrou só aqui
	// não apareceria lá para ser removido.
	if inv != nil {
		for _, p := range inv.DocPaths() {
			if _, ok := visited[vault.CanonicalPath(p)]; !ok {
				inv.Remove(p)
			}
		}
	}

	log.Warn("Reconciliação concluída", "updated", updated, "removed", removed, "skipped", skipped)
	return updated, removed, skipped
}
