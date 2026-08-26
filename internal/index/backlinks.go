package index

import "github.com/jonyd/gobsidian/internal/vault"

// Backlinks devolve as referencias que chegam naquela nota.
func (ix *Index) Backlinks(path vault.CanonicalPath) []Backlink {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return append([]Backlink(nil), ix.backlinks[path]...)
}

func (ix *Index) buildBacklinks() {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	ix.backlinks = make(map[vault.CanonicalPath][]Backlink)
	for from, note := range ix.notes {
		for _, l := range note.Links {
			if l.Resolved != "" {
				ix.backlinks[l.Resolved] = append(ix.backlinks[l.Resolved], backlinkDe(from, l))
			}
		}
	}
}
