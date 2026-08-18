package service

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
)

// GraphRequest sao os parametros de link_graph. Depth e Limit existem porque
// o grafo de um cofre real cresce depressa demais para caber numa resposta.
type GraphRequest struct {
	Path          string `json:"path"`
	Direction     string `json:"direction"` // "outgoing", "incoming", "both"
	Depth         int    `json:"depth"`
	IncludeBroken bool   `json:"include_broken"`
	IncludeEmbeds bool   `json:"include_embeds"`
	Limit         int    `json:"limit"`
}

// GraphNode e uma nota no grafo de links.
type GraphNode struct {
	Path  string `json:"path"`
	Title string `json:"title,omitempty"`
}

// GraphEdge e um link resolvido, da origem para o alvo.
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind,omitempty"`
}

// GraphResult e o retorno de link_graph.
type GraphResult struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// LinkGraph percorre o grafo a partir de uma nota, ate a profundidade pedida.
func (s *Service) LinkGraph(_ context.Context, req GraphRequest) (GraphResult, error) {
	if s.index == nil {
		return GraphResult{}, fmt.Errorf("index not available")
	}

	depth := req.Depth
	if depth <= 0 {
		depth = 1
	}
	if depth > 3 {
		depth = 3
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}

	direction := req.Direction
	if direction == "" {
		direction = "both"
	}

	startPath, err := s.index.ResolvePath(req.Path)
	if err != nil {
		return GraphResult{}, Wrap(CodePathOutsideVault, err, "resolving root path")
	}

	nodesMap := make(map[vault.CanonicalPath]GraphNode)
	edgesMap := make(map[string]GraphEdge)

	type queueItem struct {
		Path  vault.CanonicalPath
		Depth int
	}

	queue := []queueItem{{Path: startPath, Depth: 0}}
	visited := make(map[vault.CanonicalPath]bool)

	for len(queue) > 0 && len(nodesMap) < limit {
		curr := queue[0]
		queue = queue[1:]

		if visited[curr.Path] {
			continue
		}
		visited[curr.Path] = true

		n, ok := s.index.Get(curr.Path)
		if !ok {
			nodesMap[curr.Path] = GraphNode{Path: string(curr.Path)}
			continue
		}

		nodesMap[curr.Path] = GraphNode{Path: string(curr.Path), Title: n.Title}

		if curr.Depth >= depth {
			continue
		}

		if direction == "outgoing" || direction == "both" {
			for _, link := range n.Links {
				if link.Kind == parser.LinkEmbed && !req.IncludeEmbeds {
					continue
				}

				if link.State == index.LinkOK || link.State == index.LinkAnchorMissing {
					if link.Resolved != "" {
						edgeID := string(curr.Path) + "->" + string(link.Resolved)
						edgesMap[edgeID] = GraphEdge{Source: string(curr.Path), Target: string(link.Resolved), Kind: link.Kind.String()}

						if !visited[link.Resolved] && len(nodesMap)+len(queue) < limit {
							queue = append(queue, queueItem{Path: link.Resolved, Depth: curr.Depth + 1})
						}
					}
				} else if link.State == index.LinkTargetMissing && req.IncludeBroken {
					targetPath := string(link.Resolved)
					if targetPath == "" {
						targetPath = link.Target
					}
					if targetPath != "" {
						edgeID := string(curr.Path) + "->" + targetPath
						edgesMap[edgeID] = GraphEdge{Source: string(curr.Path), Target: targetPath, Kind: link.Kind.String()}
						nodesMap[vault.CanonicalPath(targetPath)] = GraphNode{Path: targetPath}
					}
				}
			}
		}

		if direction == "incoming" || direction == "both" {
			for _, bl := range s.index.Backlinks(curr.Path) {
				if bl.Kind == parser.LinkEmbed && !req.IncludeEmbeds {
					continue
				}

				edgeID := string(bl.From) + "->" + string(curr.Path)
				edgesMap[edgeID] = GraphEdge{Source: string(bl.From), Target: string(curr.Path), Kind: bl.Kind.String()}

				if !visited[bl.From] && len(nodesMap)+len(queue) < limit {
					queue = append(queue, queueItem{Path: bl.From, Depth: curr.Depth + 1})
				}
			}
		}
	}

	res := GraphResult{
		Nodes: make([]GraphNode, 0, len(nodesMap)),
		Edges: make([]GraphEdge, 0, len(edgesMap)),
	}

	for _, v := range nodesMap {
		res.Nodes = append(res.Nodes, v)
	}
	for _, e := range edgesMap {
		res.Edges = append(res.Edges, e)
	}

	return res, nil
}

// TagRequest sao os parametros de tag_list.
type TagRequest struct {
	Prefix       string `json:"prefix"`
	MinCount     int    `json:"min_count"`
	Hierarchical bool   `json:"hierarchical"`
}

// TagResult e o retorno de tag_list, com a contagem por tag.
type TagResult struct {
	Tags []index.TagCount `json:"tags"`
}

// TagList devolve as tags do cofre com suas contagens.
//
// Nao recebe ctx util: le so o indice em memoria.
func (s *Service) TagList(_ context.Context, req TagRequest) (TagResult, error) {
	if s.index == nil {
		return TagResult{}, fmt.Errorf("index not available")
	}
	tags := s.index.Tags(req.Prefix, req.MinCount)
	return TagResult{Tags: tags}, nil
}

// ListRequest sao os parametros de note_list.
type ListRequest struct {
	Query index.Query `json:"query"`
	// Fields sao os campos de frontmatter a incluir em cada item. Vazio
	// devolve nenhum — note_list e a tool barata, e despejar o frontmatter
	// inteiro de cada nota derrota o proposito dela.
	Fields []string `json:"fields,omitempty"`
}

// ListItem e a projecao que note_list devolve, conforme docs/TOOLS.md.
//
// Nao e o *index.Note inteiro, e a diferenca importa: a Note carrega headings,
// blocos e todos os links, e devolver isso por nota transforma "que notas
// existem na pasta X" numa resposta de dezenas de milhares de tokens. A tool
// existe justamente para ser a barata.
type ListItem struct {
	Path     string         `json:"path"`
	Title    string         `json:"title"`
	Hash     string         `json:"hash"`
	Modified time.Time      `json:"modified"`
	Size     int64          `json:"size"`
	Tags     []string       `json:"tags,omitempty"`
	Fields   map[string]any `json:"fields,omitempty"`
}

// ListResult e o retorno de note_list. Total e a contagem ANTES do limite,
// para o cliente saber que existe mais do que ele recebeu.
type ListResult struct {
	Notes []ListItem `json:"notes"`
	Total int        `json:"total"`
}

// ListNotes filtra e ordena notas por metadados, devolvendo a projecao barata
// ListItem em vez da Note inteira.
//
// Nao recebe ctx util: le so o indice em memoria.
func (s *Service) ListNotes(_ context.Context, req ListRequest) (ListResult, error) {
	if s.index == nil {
		return ListResult{}, fmt.Errorf("index not available")
	}

	notes, total := s.index.List(req.Query)

	items := make([]ListItem, 0, len(notes))
	for _, n := range notes {
		items = append(items, ListItem{
			Path:     string(n.Path),
			Title:    n.Title,
			Hash:     fmt.Sprintf("%016x", n.Hash),
			Modified: n.ModTime,
			Size:     n.Size,
			Tags:     n.Tags,
			Fields:   selectFields(n.Frontmatter, req.Fields),
		})
	}

	return ListResult{Notes: items, Total: total}, nil
}

// selectFields devolve so os campos pedidos.
//
// Antes disto o parametro era declarado no schema da tool e ignorado: o modelo
// pedia tres campos, recebia tudo, e nao tinha como saber que o pedido nao
// fizera nada. Schema que promete e codigo que ignora e pior que parametro
// ausente, porque quem le o schema acredita.
//
// Campo pedido e inexistente simplesmente nao aparece — a ausencia ja e a
// resposta, e inventar null obrigaria o cliente a distinguir dois nadas.
func selectFields(fm map[string]any, want []string) map[string]any {
	if len(want) == 0 || len(fm) == 0 {
		return nil
	}

	out := make(map[string]any, len(want))
	for _, k := range want {
		if v, ok := fm[k]; ok {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// MetadataRequest sao os parametros de note_metadata.
type MetadataRequest struct {
	Path    string   `json:"path"`
	Include []string `json:"include,omitempty"`
}

// MetadataResult e tudo o que o indice sabe de uma nota sem ler o disco.
type MetadataResult struct {
	Path           string               `json:"path"`
	Title          string               `json:"title"`
	Hash           string               `json:"hash"`
	Frontmatter    map[string]any       `json:"frontmatter,omitempty"`
	FrontmatterErr string               `json:"frontmatter_err,omitempty"`
	Tags           []string             `json:"tags,omitempty"`
	Aliases        []string             `json:"aliases,omitempty"`
	Headings       []string             `json:"headings,omitempty"`
	Blocks         []string             `json:"blocks,omitempty"`
	Links          []index.ResolvedLink `json:"links,omitempty"`
	Backlinks      []index.Backlink     `json:"backlinks,omitempty"`
}

// NoteMetadata devolve tudo o que o indice sabe de uma nota sem abrir o
// arquivo.
func (s *Service) NoteMetadata(_ context.Context, req MetadataRequest) (MetadataResult, error) {
	if s.index == nil {
		return MetadataResult{}, fmt.Errorf("index not available")
	}
	cp, err := s.index.ResolvePath(req.Path)
	if err != nil {
		return MetadataResult{}, Wrap(CodePathOutsideVault, err, "resolving path")
	}
	n, ok := s.index.Get(cp)
	if !ok {
		return MetadataResult{}, Wrap(CodeNoteNotFound, nil, "note not found")
	}

	includeSet := make(map[string]bool)
	if len(req.Include) == 0 {
		includeSet["frontmatter"] = true
		includeSet["tags"] = true
		includeSet["headings"] = true
		includeSet["links"] = true
		includeSet["backlinks"] = true
	} else {
		for _, inc := range req.Include {
			includeSet[inc] = true
		}
	}

	res := MetadataResult{
		Path:  string(n.Path),
		Title: n.Title,
		Hash:  fmt.Sprintf("%016x", n.Hash),
	}

	if includeSet["frontmatter"] {
		res.Frontmatter = n.Frontmatter
		res.FrontmatterErr = n.FrontmatterErr
		res.Aliases = n.Aliases
	}
	if includeSet["tags"] {
		res.Tags = n.Tags
	}
	if includeSet["headings"] {
		headings := make([]string, len(n.Headings))
		for i, h := range n.Headings {
			headings[i] = h.Text
		}
		res.Headings = headings
	}
	if includeSet["blocks"] {
		blocks := make([]string, len(n.Blocks))
		for i, b := range n.Blocks {
			blocks[i] = b.ID
		}
		res.Blocks = blocks
	}
	if includeSet["links"] {
		res.Links = n.Links
	}
	if includeSet["backlinks"] {
		res.Backlinks = s.index.Backlinks(cp)
	}

	return res, nil
}

// RuntimeStats sao os numeros do runtime do Go, para diagnosticar consumo de
// memoria e vazamento de goroutine contra o orcamento de RNF-07.
type RuntimeStats struct {
	NumGoroutine int    `json:"num_goroutine"`
	Alloc        uint64 `json:"alloc"`
	TotalAlloc   uint64 `json:"total_alloc"`
	Sys          uint64 `json:"sys"`
	NumGC        uint32 `json:"num_gc"`
}

// StatsRequest sao os parametros de vault_stats. Os dois campos sao opcionais
// porque as contagens de saude percorrem o indice inteiro e o runtime so
// interessa quando se esta diagnosticando.
type StatsRequest struct {
	IncludeHealth  bool `json:"include_health"`
	IncludeRuntime bool `json:"include_runtime"`
}

// StatsResult e o retorno de vault_stats. Os campos aqui sao contrato
// publico: docs/TOOLS.md descreve cada um pelo nome JSON, e um campo declarado
// que ninguem preenche e pior que um campo ausente, porque quem le acredita.
type StatsResult struct {
	Notes     int   `json:"notes"`
	Assets    int   `json:"assets"`
	TotalSize int64 `json:"total_size"`
	// Ponteiro, e nao int com omitempty. Zero e contagem legitima: um cofre sem
	// orfas reporta orphans=0. Se include_health desligado tambem reportasse 0,
	// o chamador nao distinguiria "nao ha orfas" de "nao perguntei" — que e a
	// regra deste projeto sobre o que um zero significa, e o motivo de
	// ReadOnlySet e DebounceMSSet existirem. `omitempty` em int seria pior
	// ainda: sumiria com o zero legitimo quando a saude FOI pedida.
	//
	// nil  = nao pedido (include_health false)
	// &0   = pedido, e nao ha nenhum
	Orphans           *int           `json:"orphans,omitempty"`
	BrokenLinks       *int           `json:"broken_links,omitempty"`
	BrokenAnchor      *int           `json:"broken_anchors,omitempty"`
	FrontmatterErrors *int           `json:"frontmatter_errors,omitempty"`
	Collisions        int            `json:"alias_collisions"`
	Generation        uint64         `json:"generation"`
	Runtime           *RuntimeStats  `json:"runtime,omitempty"`
	Watcher           *WatchCounters `json:"watcher,omitempty"`
}

// VaultStats was relocated from service.go
func (s *Service) VaultStats(ctx context.Context, req StatsRequest) (StatsResult, error) {
	if s.index == nil {
		var out StatsResult
		err := s.vault.Walk(ctx, func(e vault.Entry) error {
			if e.IsNote {
				out.Notes++
			} else {
				out.Assets++
			}
			out.TotalSize += e.Size
			return nil
		})
		if err != nil {
			return StatsResult{}, Wrap(CodeVaultUnavailable, err, "varrendo o cofre")
		}
		return out, nil
	}

	// A varredura so acontece quando a saude foi pedida. Ate 2026-07-31 o campo
	// IncludeHealth era declarado aqui e em docs/TOOLS.md e NUNCA lido: a
	// varredura rodava sempre, e o parametro nao fazia nada. E o defeito do
	// note_list.fields — o schema e justamente o que o chamador le para decidir.
	var orphans, brokenLinks, brokenAnchors, frontmatterErrors int

	if req.IncludeHealth {
		// NotePaths, nao Paths: Paths inclui anexos, que Get nao resolve.
		for _, p := range s.index.NotePaths() {
			n, ok := s.index.Get(p)
			if ok {
				if n.FrontmatterErr != "" {
					frontmatterErrors++
				}
				bl := s.index.Backlinks(p)
				if len(bl) == 0 {
					orphans++
				}
				for _, l := range n.Links {
					// LinkExternal nao entra em nenhuma das duas contagens: uma
					// URL nunca foi para o cofre, e conta-la como quebrada afoga
					// o sinal de saude em falso positivo.
					switch l.State {
					case index.LinkTargetMissing:
						brokenLinks++
					case index.LinkAnchorMissing:
						brokenAnchors++
					}
				}
			}
		}

	}

	res := StatsResult{
		Notes:     s.index.NoteCount(),
		Assets:    s.index.AssetCount(),
		TotalSize: s.index.TotalSize(),

		Collisions: s.index.AliasCollisions(),
		Generation: s.index.Generation(),
	}

	if req.IncludeHealth {
		res.Orphans = &orphans
		res.BrokenLinks = &brokenLinks
		res.BrokenAnchor = &brokenAnchors
		res.FrontmatterErrors = &frontmatterErrors
	}

	if req.IncludeRuntime {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		res.Runtime = &RuntimeStats{
			NumGoroutine: runtime.NumGoroutine(),
			Alloc:        mem.Alloc,
			TotalAlloc:   mem.TotalAlloc,
			Sys:          mem.Sys,
			NumGC:        mem.NumGC,
		}
		if s.watcher != nil {
			stats := s.watcher.Stats()
			res.Watcher = &stats
		}
	}

	return res, nil
}
