package watcher

import (
	"context"
	"log/slog"
	"os"

	"github.com/cespare/xxhash/v2"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// RenameCandidate representa um possível rename detectado.
// A tarefa exige reportar os candidatos de atualização sem alterar o arquivo no disco.
type RenameCandidate struct {
	From      vault.CanonicalPath
	To        vault.CanonicalPath
	Backlinks []index.Backlink
}

// CorrelateRenames analisa um lote de caminhos coalescidos para encontrar renames.
// A correlação ocorre quando há exatamente uma remoção e exatamente uma criação com o mesmo xxhash.
func CorrelateRenames(batch []vault.CanonicalPath, v *vault.Vault, idx *index.Index, log *slog.Logger) (renames []RenameCandidate, nonRenames []vault.CanonicalPath) {
	var missing []vault.CanonicalPath
	var added []vault.CanonicalPath

	// 1. Separar o batch
	for _, path := range batch {
		abs := v.Abs(path)
		_, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				missing = append(missing, path)
			} else {
				// Erros de permissão, etc., não devem entrar na correlação
				nonRenames = append(nonRenames, path)
			}
		} else {
			added = append(added, path)
		}
	}

	// 2. Extrair os hashes do index para os deletados
	missingHashes := make(map[uint64][]vault.CanonicalPath)
	for _, p := range missing {
		n, ok := idx.Get(p)
		if ok && n.Hash != 0 && n.Size > 0 {
			missingHashes[n.Hash] = append(missingHashes[n.Hash], p)
		} else {
			// Se não estava no índice (ex. anexo) ou o hash era zero, não podemos correlacionar
			nonRenames = append(nonRenames, p)
		}
	}

	// 3. Calcular hashes para os recém-adicionados
	addedHashes := make(map[uint64][]vault.CanonicalPath)
	for _, p := range added {
		data, err := v.ReadAll(context.Background(), p)
		if err == nil && len(data) > 0 {
			// xxhash calculado sobre bytes crus (raw bytes), antes de remover o BOM, conforme exigido.
			h := xxhash.Sum64(data)
			if h != 0 {
				addedHashes[h] = append(addedHashes[h], p)
			} else {
				nonRenames = append(nonRenames, p)
			}
		} else {
			nonRenames = append(nonRenames, p)
		}
	}

	// 4. Correlacionar e detectar ambiguidade
	matchedMissing := make(map[vault.CanonicalPath]bool)
	matchedAdded := make(map[vault.CanonicalPath]bool)

	for h, ms := range missingHashes {
		as, ok := addedHashes[h]
		if !ok {
			continue // Sem correspondente
		}

		// Regra 3: ambiguidade (arquivos idênticos, como vazios).
		// Exatamente 1 e 1.
		if len(ms) == 1 && len(as) == 1 {
			oldPath := ms[0]
			newPath := as[0]

			backlinks := idx.Backlinks(oldPath)

			renames = append(renames, RenameCandidate{
				From:      oldPath,
				To:        newPath,
				Backlinks: backlinks,
			})
			matchedMissing[oldPath] = true
			matchedAdded[newPath] = true
			log.Info("Rename detectado por hash de conteúdo", "from", oldPath, "to", newPath)
		} else {
			log.Debug("Correlação recusada por ambiguidade", "hash", h, "missing_count", len(ms), "added_count", len(as))
		}
	}

	// 5. Adicionar os não correlacionados em nonRenames
	for _, p := range missing {
		if !matchedMissing[p] {
			if n, ok := idx.Get(p); !ok || n.Hash == 0 {
				// Já foi adicionado acima
			} else {
				nonRenames = append(nonRenames, p)
			}
		}
	}
	for _, p := range added {
		if !matchedAdded[p] {
			data, err := v.ReadAll(context.Background(), p)
			if err != nil {
				// Já foi adicionado acima
			} else {
				h := xxhash.Sum64(data)
				if h == 0 {
					// Já foi
				} else {
					nonRenames = append(nonRenames, p)
				}
			}
		}
	}

	return renames, nonRenames
}
