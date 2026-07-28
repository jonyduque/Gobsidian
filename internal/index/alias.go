package index

import "strings"

func (ix *Index) buildAliasMap() {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	for _, n := range ix.notes {
		for _, alias := range n.Aliases {
			lower := strings.ToLower(alias)
			ix.byAlias[lower] = append(ix.byAlias[lower], n.Path)
		}
	}
}
