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

	// citantesPorNome e o indice reverso que permite reindexar um arquivo
	// sem varrer o cofre inteiro (RNF-06, Task 86): para cada chave derivada
	// por nomeChave, a lista de notas que tem ao menos um link cujo alvo
	// normaliza para aquela chave. Cobre link RESOLVIDO e QUEBRADO — um
	// [[foo]] sem arquivo tem de estar aqui sob "foo", senao criar foo.md
	// nunca conserta o link, porque nada dispara o reprocessamento dele.
	citantesPorNome map[string][]vault.CanonicalPath

	generation uint64
}

// New devolve um indice vazio. Quem o popula e Build, a partir de um cofre.
func New() *Index {
	return &Index{
		notes:           make(map[vault.CanonicalPath]*Note),
		assets:          make(map[vault.CanonicalPath]*Asset),
		lowerPath:       make(map[string]vault.CanonicalPath),
		byName:          make(map[string][]vault.CanonicalPath),
		byAlias:         make(map[string][]vault.CanonicalPath),
		backlinks:       make(map[vault.CanonicalPath][]Backlink),
		tags:            make(map[string][]vault.CanonicalPath),
		citantesPorNome: make(map[string][]vault.CanonicalPath),
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

	if classificar(r.entry) == classeAnexo {
		ix.publishAssetLocked(anexoDaEntrada(r.entry))
		return
	}

	n := notaSemLeitura(r.entry)

	// r.note e nil quando build.go NAO leu o arquivo, e ele nao le de
	// proposito no placeholder de nuvem: abrir dispara download sincrono. A
	// nota entra so com o que a varredura de diretorio ja sabia, que e o que
	// notaSemLeitura acabou de montar.
	if r.note != nil {
		n.Hash = r.hash
		n.EOL = r.eol
		n.BOM = r.bom
		n.Title = r.note.Title
		n.TitleNorm = normalizeTitleForNote(r.note.Title)
		n.Frontmatter = r.note.Frontmatter
		n.Tags = r.note.Tags
		n.Aliases = r.note.Aliases
		n.Headings = r.note.Headings
		n.Blocks = r.note.Blocks
		n.Inline = r.note.Inline

		for _, l := range r.note.Links {
			n.Links = append(n.Links, ResolvedLink{Link: l})
		}
	}

	ix.publishNoteLocked(n)
}

// publishNoteLocked registra uma nota ja construida nos indices derivados:
// notes, lowerPath, byName e tags. E o UNICO lugar que faz isso — Build (via
// insert) e o carregamento do cache de disco (persist.go) passam os dois
// pelo mesmo caminho, o que impede as duas fontes de povoar os indices
// derivados de jeitos que divergem. Exige ix.mu ja travado.
func (ix *Index) publishNoteLocked(n *Note) {
	ix.notes[n.Path] = n
	ix.publishNameLocked(n.Path)
	for _, t := range n.Tags {
		ix.tags[t] = append(ix.tags[t], n.Path)
	}
	ix.registrarCitantesLocked(n.Path, n.Links)
}

// publishAssetLocked e o par de publishNoteLocked para anexos. Exige ix.mu
// ja travado.
func (ix *Index) publishAssetLocked(a *Asset) {
	ix.assets[a.Path] = a
	ix.publishNameLocked(a.Path)
}

// publishNameLocked povoa lowerPath e byName, comuns a nota e anexo. Exige
// ix.mu ja travado.
func (ix *Index) publishNameLocked(path vault.CanonicalPath) {
	lower := strings.ToLower(string(path))
	ix.lowerPath[lower] = path

	base := vault.CanonicalPath(filepath.ToSlash(filepath.Base(string(path))))
	ix.byName[string(base)] = append(ix.byName[string(base)], path)
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
