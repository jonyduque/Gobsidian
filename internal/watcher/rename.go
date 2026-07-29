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

// CorrelateRenames analisa um lote de caminhos coalescidos para encontrar renames em uma única passagem.
// A correlação ocorre quando há exatamente uma remoção e exatamente uma criação com o mesmo xxhash.
func CorrelateRenames(ctx context.Context, batch []vault.CanonicalPath, v *vault.Vault, idx *index.Index, log *slog.Logger) (renames []RenameCandidate, nonRenames []vault.CanonicalPath) {
	missingHashes := make(map[uint64][]vault.CanonicalPath)

	type addedInfo struct {
		path vault.CanonicalPath
		abs  string
	}
	var addedCandidates []addedInfo

	// 1. Laço único sobre o batch para classificar ausentes (do índice) e novos (do disco)
	for _, p := range batch {
		abs := v.Abs(p)
		info, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				n, ok := idx.Get(p)
				if ok && n.Hash != 0 && n.Size > 0 {
					missingHashes[n.Hash] = append(missingHashes[n.Hash], p)
				}
			}
			continue
		}

		if info.IsDir() {
			continue
		}

		if vault.Classify(p) == vault.ClassNote && !vault.IsCloudOnly(abs) {
			addedCandidates = append(addedCandidates, addedInfo{path: p, abs: abs})
		}
	}

	// Curto-circuito: se não há remoções com hash no lote, nenhum rename é possível
	if len(missingHashes) == 0 {
		return nil, batch
	}

	// 2. Ler conteúdo apenas das notas adicionadas elegíveis quando há remoções
	addedHashes := make(map[uint64][]vault.CanonicalPath)
	for _, cand := range addedCandidates {
		data, rErr := v.ReadAll(ctx, cand.path)
		if rErr == nil && len(data) > 0 {
			h := xxhash.Sum64(data)
			if h != 0 {
				addedHashes[h] = append(addedHashes[h], cand.path)
			}
		}
	}

	// 3. Correlacionar por cardinalidade exata 1-para-1
	matchedMissing := make(map[vault.CanonicalPath]bool)
	matchedAdded := make(map[vault.CanonicalPath]bool)

	for h, ms := range missingHashes {
		as, ok := addedHashes[h]
		if !ok {
			continue
		}

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
			if log != nil {
				log.Info("Rename detectado por hash de conteúdo", "from", oldPath, "to", newPath)
			}
		} else {
			if log != nil {
				log.Debug("Correlação recusada por ambiguidade", "hash", h, "missing_count", len(ms), "added_count", len(as))
			}
		}
	}

	// 4. Varredura final única sobre batch para montar nonRenames sem duplicatas
	seen := make(map[vault.CanonicalPath]bool)
	for _, p := range batch {
		if matchedMissing[p] || matchedAdded[p] {
			continue
		}
		if !seen[p] {
			seen[p] = true
			nonRenames = append(nonRenames, p)
		}
	}

	return renames, nonRenames
}
