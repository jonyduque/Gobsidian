package service

import (
	"cmp"
	"context"
	"fmt"
	"runtime"
	"slices"
	"strings"
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
	// Distance e a distancia em saltos ate a nota de origem da consulta.
	//
	// Prometida em docs/TOOLS.md e ausente ate 2026-08-27 (achado M5). Sem ela
	// o grafo devolve um conjunto de nos sem forma: quem recebe nao distingue
	// vizinho direto de no a tres saltos, que costuma ser a unica coisa que a
	// resposta precisava dizer.
	//
	// A travessia e em largura, entao a primeira vez que um no e retirado da
	// fila JA e pela rota mais curta — o valor nao precisa de relaxamento.
	Distance int `json:"distance"`
}

// GraphEdge e um link resolvido, da origem para o alvo.
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind,omitempty"`
	// Alias e Anchor sao a grafia da referencia. Prometidos em docs/TOOLS.md e
	// ausentes ate 2026-08-27 (achado M5).
	//
	// Eles tambem mudam a IDENTIDADE da aresta: A->B#Prescricao e
	// A->B#Honorarios sao referencias diferentes. Ate esta data a chave de
	// deduplicacao era so origem+destino, entao as duas colapsavam numa
	// aresta so — e publicar a ancora de uma delas seria escolher arbitraria-
	// mente qual das duas contar. A chave passou a incluir tipo, alias e
	// ancora.
	Alias  string `json:"alias,omitempty"`
	Anchor string `json:"anchor,omitempty"`
	// Resolved diz se a referencia encontrou alvo no cofre. Falso so aparece
	// com include_broken, que e o unico caminho que produz aresta nao resolvida.
	Resolved bool `json:"resolved"`
}

// chaveDaAresta e a identidade de uma aresta do grafo. Uma funcao so, porque os
// tres pontos que inserem em edgesMap tem de concordar: dois deles montavam a
// chave a mao e o terceiro tambem, e bastava um esquecer a ancora para arestas
// distintas colapsarem em silencio.
func chaveDaAresta(e GraphEdge) string {
	// %q em cada parte, e nao um separador escolhido a dedo: qualquer
	// separador literal pode aparecer DENTRO de um alias ou de uma ancora, e
	// ai duas arestas diferentes produzem a mesma chave. A citacao resolve
	// isso sem depender de suposicao sobre o conteudo.
	return fmt.Sprintf("%q|%q|%q|%q|%q", e.Source, e.Target, e.Kind, e.Alias, e.Anchor)
}

// GraphResult e o retorno de link_graph.
type GraphResult struct {
	Nodes        []GraphNode `json:"nodes"`
	Edges        []GraphEdge `json:"edges"`
	LimitEfetivo int         `json:"effective_limit"`
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

	// Mesma conta de teto que note_list. Estava inline aqui, com os números
	// mágicos repetidos, enquanto note_list não clampava nada — os dois
	// declaram `"maximum": 500` no schema (achado B4).
	limit := ComTeto(req.Limit)

	direction, err := ValidarEnum("direction", req.Direction, "both", "both", "outgoing", "incoming")
	if err != nil {
		return GraphResult{}, err
	}

	startPath, err := s.index.ResolvePath(req.Path)
	if err != nil {
		return GraphResult{}, ErroDeResolucao(req.Path, err)
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
			nodesMap[curr.Path] = GraphNode{Path: string(curr.Path), Distance: curr.Depth}
			continue
		}

		nodesMap[curr.Path] = GraphNode{Path: string(curr.Path), Title: n.Title, Distance: curr.Depth}

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
						a := GraphEdge{
							Source:   string(curr.Path),
							Target:   string(link.Resolved),
							Kind:     link.Kind.String(),
							Alias:    link.Alias,
							Anchor:   link.Anchor,
							Resolved: true,
						}
						edgesMap[chaveDaAresta(a)] = a

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
						a := GraphEdge{
							Source: string(curr.Path),
							Target: targetPath,
							Kind:   link.Kind.String(),
							Alias:  link.Alias,
							Anchor: link.Anchor,
							// Resolved falso de proposito: este ramo E o do
							// alvo inexistente. Um no criado aqui nunca entra
							// na fila, entao ele fica a curr.Depth+1 e nao
							// ganha distancia menor depois.
							Resolved: false,
						}
						edgesMap[chaveDaAresta(a)] = a
						if _, ja := nodesMap[vault.CanonicalPath(targetPath)]; !ja {
							nodesMap[vault.CanonicalPath(targetPath)] = GraphNode{Path: targetPath, Distance: curr.Depth + 1}
						}
					}
				}
			}
		}

		if direction == "incoming" || direction == "both" {
			for _, bl := range s.index.Backlinks(curr.Path) {
				if bl.Kind == parser.LinkEmbed && !req.IncludeEmbeds {
					continue
				}

				// Backlink so existe para link que resolveu, entao Resolved e
				// sempre verdadeiro neste ramo.
				a := GraphEdge{
					Source:   string(bl.From),
					Target:   string(curr.Path),
					Kind:     bl.Kind.String(),
					Alias:    bl.Alias,
					Anchor:   bl.Anchor,
					Resolved: true,
				}
				edgesMap[chaveDaAresta(a)] = a

				if !visited[bl.From] && len(nodesMap)+len(queue) < limit {
					queue = append(queue, queueItem{Path: bl.From, Depth: curr.Depth + 1})
				}
			}
		}
	}

	res := GraphResult{
		Nodes:        make([]GraphNode, 0, len(nodesMap)),
		Edges:        make([]GraphEdge, 0, len(edgesMap)),
		LimitEfetivo: limit,
	}

	for _, v := range nodesMap {
		res.Nodes = append(res.Nodes, v)
	}
	for _, e := range edgesMap {
		res.Edges = append(res.Edges, e)
	}

	slices.SortFunc(res.Nodes, func(a, b GraphNode) int {
		return cmp.Compare(a.Path, b.Path)
	})

	slices.SortFunc(res.Edges, func(a, b GraphEdge) int {
		if c := cmp.Compare(a.Source, b.Source); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Target, b.Target); c != 0 {
			return c
		}
		return cmp.Compare(a.Kind, b.Kind)
	})

	return res, nil
}

// TagRequest sao os parametros de tag_list.
type TagRequest struct {
	Prefix       string `json:"prefix"`
	MinCount     int    `json:"min_count"`
	Sort         string `json:"sort"`
	Hierarchical bool   `json:"hierarchical"`
}

// TagNode e um no na arvore de tags ou um item da lista plana.
type TagNode struct {
	Tag      string `json:"tag"`
	Count    int    `json:"count"`
	Children []any  `json:"children,omitempty"`
}

// TagResult e o retorno de tag_list, com a contagem por tag.
type TagResult struct {
	Tags []TagNode `json:"tags"`
}

func ordenarTags(tags []TagNode, sortMode string) {
	if sortMode == "count" {
		slices.SortStableFunc(tags, func(a, b TagNode) int {
			if c := cmp.Compare(b.Count, a.Count); c != 0 {
				return c
			}
			return cmp.Compare(a.Tag, b.Tag)
		})
	} else {
		slices.SortStableFunc(tags, func(a, b TagNode) int {
			return cmp.Compare(a.Tag, b.Tag)
		})
	}
	for i := range tags {
		if len(tags[i].Children) > 0 {
			childList := make([]TagNode, len(tags[i].Children))
			for j, ch := range tags[i].Children {
				if node, ok := ch.(TagNode); ok {
					childList[j] = node
				}
			}
			ordenarTags(childList, sortMode)
			for j, node := range childList {
				tags[i].Children[j] = node
			}
		}
	}
}

// TagList devolve as tags do cofre com suas contagens.
//
// Nao recebe ctx util: le so o indice em memoria.
func (s *Service) TagList(_ context.Context, req TagRequest) (TagResult, error) {
	if s.index == nil {
		return TagResult{}, fmt.Errorf("index not available")
	}
	sortTags, err := ValidarEnum("sort", req.Sort, "name", "name", "count")
	if err != nil {
		return TagResult{}, err
	}
	req.Sort = sortTags

	if req.Hierarchical {
		return s.tagListHierarchical(req), nil
	}
	tags := s.index.Tags(req.Prefix, req.MinCount)
	nodes := make([]TagNode, len(tags))
	for i, t := range tags {
		nodes[i] = TagNode{Tag: t.Tag, Count: t.Count}
	}
	ordenarTags(nodes, req.Sort)
	return TagResult{Tags: nodes}, nil
}

type tempTagNode struct {
	segment  string
	fullTag  string
	count    int
	children map[string]*tempTagNode
}

func (s *Service) tagListHierarchical(req TagRequest) TagResult {
	noteSets := make(map[string]map[vault.CanonicalPath]bool)

	for _, p := range s.index.NotePaths() {
		n, ok := s.index.Get(p)
		if !ok {
			continue
		}
		for _, tag := range n.Tags {
			tagClean := strings.TrimPrefix(tag, "#")
			parts := strings.Split(tagClean, "/")
			curr := ""
			for i, part := range parts {
				if i == 0 {
					curr = part
				} else {
					curr = curr + "/" + part
				}
				if noteSets[curr] == nil {
					noteSets[curr] = make(map[vault.CanonicalPath]bool)
				}
				noteSets[curr][p] = true
			}
		}
	}

	prefixLower := strings.ToLower(req.Prefix)
	rootNodes := make(map[string]*tempTagNode)

	for fullTag, paths := range noteSets {
		count := len(paths)
		if count < req.MinCount {
			continue
		}
		if prefixLower != "" && !strings.HasPrefix(strings.ToLower(fullTag), prefixLower) {
			continue
		}

		parts := strings.Split(fullTag, "/")
		currentMap := rootNodes
		currPath := ""
		for i, part := range parts {
			if i == 0 {
				currPath = part
			} else {
				currPath = currPath + "/" + part
			}
			node, exists := currentMap[part]
			if !exists {
				node = &tempTagNode{
					segment:  part,
					fullTag:  currPath,
					count:    len(noteSets[currPath]),
					children: make(map[string]*tempTagNode),
				}
				currentMap[part] = node
			}
			currentMap = node.children
		}
	}

	var convert func(m map[string]*tempTagNode) []TagNode
	convert = func(m map[string]*tempTagNode) []TagNode {
		res := make([]TagNode, 0, len(m))
		for _, tn := range m {
			childList := convert(tn.children)
			var anyChildren []any
			if len(childList) > 0 {
				anyChildren = make([]any, len(childList))
				for i, ch := range childList {
					anyChildren[i] = ch
				}
			}
			res = append(res, TagNode{
				Tag:      tn.fullTag,
				Count:    tn.count,
				Children: anyChildren,
			})
		}
		return res
	}

	resNodes := convert(rootNodes)
	ordenarTags(resNodes, req.Sort)
	return TagResult{Tags: resNodes}
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

	// O teto e os enums são conferidos AQUI, e não no boundary MCP, porque o
	// CLI (`gobsidian index`, `search`) chega pelo mesmo caminho. Validar só na
	// borda deixaria a segunda porta sem guarda (achado B4).
	q := req.Query
	q.Limit = ComTeto(q.Limit)
	var err error
	if q.TagMode, err = ValidarEnum("tag_mode", q.TagMode, "all", "all", "any"); err != nil {
		return ListResult{}, err
	}
	if q.Sort, err = ValidarEnum("sort", q.Sort, "path", "path", "modified", "size", "title"); err != nil {
		return ListResult{}, err
	}
	if q.Order, err = ValidarEnum("order", q.Order, "asc", "asc", "desc"); err != nil {
		return ListResult{}, err
	}

	notes, total := s.index.List(q)

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
	Path           string         `json:"path"`
	Title          string         `json:"title"`
	Hash           string         `json:"hash"`
	Frontmatter    map[string]any `json:"frontmatter,omitempty"`
	FrontmatterErr string         `json:"frontmatter_err,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	Aliases        []string       `json:"aliases,omitempty"`
	// Headings traz nivel, texto, slug e offsets — nao so o texto.
	//
	// Era []string ate 2026-08-27 (achado M3), enquanto docs/TOOLS.md prometia
	// "nivel, texto, slug e offsets — o que permite planejar uma leitura ou uma
	// escrita seletiva antes de faze-la". Sem o slug nao da para montar a
	// ancora; sem os offsets nao da para planejar leitura seletiva. O campo
	// existia e respondia menos do que o contrato dizia, que e a mesma classe do
	// achado A8.
	Headings  []parser.Heading     `json:"headings,omitempty"`
	Blocks    []string             `json:"blocks,omitempty"`
	Links     []index.ResolvedLink `json:"links,omitempty"`
	Backlinks []index.Backlink     `json:"backlinks,omitempty"`
	// InlineFields atende o valor "inline_fields" do enum de include, que era
	// aceito pelo schema e descartado pelo codigo (achado M4). Schema que
	// promete e codigo que ignora e pior que parametro ausente: o modelo do
	// outro lado le o schema para decidir o que pedir.
	InlineFields map[string][]string `json:"inline_fields,omitempty"`
}

// NoteMetadata devolve tudo o que o indice sabe de uma nota sem abrir o
// arquivo.
func (s *Service) NoteMetadata(_ context.Context, req MetadataRequest) (MetadataResult, error) {
	if s.index == nil {
		return MetadataResult{}, fmt.Errorf("index not available")
	}
	cp, err := s.index.ResolvePath(req.Path)
	if err != nil {
		return MetadataResult{}, ErroDeResolucao(req.Path, err)
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
		res.Headings = n.Headings
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
	if includeSet["inline_fields"] {
		res.InlineFields = n.Inline
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
