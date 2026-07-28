package index

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/jonyd/gobsidian/internal/vault"
)

type Index struct {
	mu sync.RWMutex

	notes     map[vault.CanonicalPath]*Note
	assets    map[vault.CanonicalPath]*Asset
	lowerPath map[string]vault.CanonicalPath
	byName    map[string][]vault.CanonicalPath
	byAlias   map[string][]vault.CanonicalPath
	backlinks map[vault.CanonicalPath][]Backlink
	tags      map[string][]vault.CanonicalPath

	generation uint64
}

func New() *Index {
	return &Index{
		notes:     make(map[vault.CanonicalPath]*Note),
		assets:    make(map[vault.CanonicalPath]*Asset),
		lowerPath: make(map[string]vault.CanonicalPath),
		byName:    make(map[string][]vault.CanonicalPath),
		byAlias:   make(map[string][]vault.CanonicalPath),
		backlinks: make(map[vault.CanonicalPath][]Backlink),
		tags:      make(map[string][]vault.CanonicalPath),
	}
}

func (ix *Index) Get(path vault.CanonicalPath) (*Note, bool) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	n, ok := ix.notes[path]
	return n, ok
}

func (ix *Index) NoteCount() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return len(ix.notes)
}

func (ix *Index) AssetCount() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return len(ix.assets)
}

func (ix *Index) TotalSize() int64 {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	var total int64
	for _, n := range ix.notes {
		total += n.Size
	}
	for _, a := range ix.assets {
		total += a.Size
	}
	return total
}

func (ix *Index) Generation() uint64 {
	return atomic.LoadUint64(&ix.generation)
}

func (ix *Index) insert(r parsed) {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	atomic.AddUint64(&ix.generation, 1)

	if r.entry.IsNote {
		n := &Note{
			Path:        r.entry.Path,
			Title:       r.note.Title,
			Size:        r.entry.Size,
			ModTime:     r.entry.ModTime,
			Hash:        r.hash,
			EOL:         r.eol,
			BOM:         r.bom,
			CloudOnly:   r.entry.CloudOnly,
			Frontmatter: r.note.Frontmatter,
			Tags:        r.note.Tags,
			Aliases:     r.note.Aliases,
			Headings:    r.note.Headings,
			Blocks:      r.note.Blocks,
			Inline:      r.note.Inline,
		}

		for _, l := range r.note.Links {
			n.Links = append(n.Links, ResolvedLink{Link: l})
		}

		ix.notes[r.entry.Path] = n
	} else {
		a := &Asset{
			Path:    r.entry.Path,
			Size:    r.entry.Size,
			ModTime: r.entry.ModTime,
		}
		ix.assets[r.entry.Path] = a
	}

	lower := strings.ToLower(string(r.entry.Path))
	ix.lowerPath[lower] = r.entry.Path
}

func (ix *Index) buildAliasMap() {
	// Not specified in the instruction, leave empty for now
}

func (ix *Index) resolveAllLinks() {
	// Not specified in the instruction, leave empty for now
}

func (ix *Index) buildBacklinks() {
	// Not specified in the instruction, leave empty for now
}
