package index

import (
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/jonyd/gobsidian/internal/vault"
)

// Index e o cofre indexado em memoria: metadados e OFFSETS DE BYTE, nunca
// conteudo. E o que sustenta o orcamento de 60 MB de RSS e o que faz ler uma
// secao de 2 KB numa nota de 500 KB custar 2 KB.
//
// Leituras sao concorrentes; escritas sao serializadas na thread do watcher.
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

// New devolve um indice vazio. Quem o popula e Build, a partir de um cofre.
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

// Get devolve a nota daquele caminho canonico.
//
// O booleano NAO e opcional: Get resolve NOTAS, e Paths devolve notas e anexos.
// Iterar Paths chamando Get e desreferenciar sem checar estoura em qualquer
// cofre que tenha um anexo. Use NotePaths quando quiser so as notas.
func (ix *Index) Get(path vault.CanonicalPath) (*Note, bool) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	n, ok := ix.notes[path]
	return n, ok
}

// NoteCount e a quantidade de notas indexadas.
func (ix *Index) NoteCount() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return len(ix.notes)
}

// AssetCount e a quantidade de anexos indexados. Anexo entra por nome e
// nunca e lido — sem isso todo embed de imagem viraria link quebrado.
func (ix *Index) AssetCount() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return len(ix.assets)
}

// TotalSize soma os tamanhos de notas e anexos, em bytes.
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

// Generation conta mutacoes do indice desde o boot. Um cliente que guardou
// um resultado pode comparar a geracao para saber se ele envelheceu.
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
			TitleNorm:   normalizeTitleForNote(r.note.Title),
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

		for _, t := range r.note.Tags {
			ix.tags[t] = append(ix.tags[t], r.entry.Path)
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

	// Populate byName for note and asset resolution
	base := vault.CanonicalPath(filepath.ToSlash(filepath.Base(string(r.entry.Path))))
	ix.byName[string(base)] = append(ix.byName[string(base)], r.entry.Path)
}

// Paths devolve todos os caminhos indexados, notas E anexos, ordenados.
//
// Get so resolve notas. Iterar isto chamando Get e desreferenciar sem
// checar o booleano estoura em qualquer cofre que tenha um anexo — use
// NotePaths quando quiser so as notas.
func (ix *Index) Paths() []vault.CanonicalPath {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	paths := make([]vault.CanonicalPath, 0, len(ix.notes)+len(ix.assets))
	for p := range ix.notes {
		paths = append(paths, p)
	}
	for p := range ix.assets {
		paths = append(paths, p)
	}

	slices.Sort(paths)
	return paths
}
