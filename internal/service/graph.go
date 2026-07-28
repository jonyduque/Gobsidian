package service

import (
	"context"
	"fmt"
	"runtime"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

type GraphRequest struct {
	Path  string `json:"path"`
	Depth int    `json:"depth"`
	Limit int    `json:"limit"`
}

type GraphNode struct {
	Path  string `json:"path"`
	Title string `json:"title,omitempty"`
}

type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind,omitempty"`
}

type GraphResult struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

func (s *Service) LinkGraph(ctx context.Context, req GraphRequest) (GraphResult, error) {
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

		for _, link := range n.Links {
			if link.State == index.LinkOK && link.Resolved != "" {
				edgeId := string(curr.Path) + "->" + string(link.Resolved)
				edgesMap[edgeId] = GraphEdge{Source: string(curr.Path), Target: string(link.Resolved)}

				if !visited[link.Resolved] && len(nodesMap)+len(queue) < limit {
					queue = append(queue, queueItem{Path: link.Resolved, Depth: curr.Depth + 1})
				}
			}
		}

		for _, bl := range s.index.Backlinks(curr.Path) {
			edgeId := string(bl.From) + "->" + string(curr.Path)
			edgesMap[edgeId] = GraphEdge{Source: string(bl.From), Target: string(curr.Path), Kind: fmt.Sprint(bl.Kind)}

			if !visited[bl.From] && len(nodesMap)+len(queue) < limit {
				queue = append(queue, queueItem{Path: bl.From, Depth: curr.Depth + 1})
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

type TagRequest struct {
	Prefix       string `json:"prefix"`
	MinCount     int    `json:"min_count"`
	Hierarchical bool   `json:"hierarchical"`
}

type TagResult struct {
	Tags []index.TagCount `json:"tags"`
}

func (s *Service) TagList(ctx context.Context, req TagRequest) (TagResult, error) {
	if s.index == nil {
		return TagResult{}, fmt.Errorf("index not available")
	}
	tags := s.index.Tags(req.Prefix, req.MinCount)
	return TagResult{Tags: tags}, nil
}

type ListRequest struct {
	Query index.Query `json:"query"`
}

type ListResult struct {
	Notes []*index.Note `json:"notes"`
	Total int           `json:"total"`
}

func (s *Service) ListNotes(ctx context.Context, req ListRequest) (ListResult, error) {
	if s.index == nil {
		return ListResult{}, fmt.Errorf("index not available")
	}
	notes, total := s.index.List(req.Query)
	return ListResult{
		Notes: notes,
		Total: total,
	}, nil
}

type MetadataRequest struct {
	Path string `json:"path"`
}

type MetadataResult struct {
	Path        string               `json:"path"`
	Frontmatter map[string]any       `json:"frontmatter,omitempty"`
	Tags        []string             `json:"tags,omitempty"`
	Aliases     []string             `json:"aliases,omitempty"`
	Headings    []string             `json:"headings,omitempty"`
	Blocks      []string             `json:"blocks,omitempty"`
	Links       []index.ResolvedLink `json:"links,omitempty"`
	Backlinks   []index.Backlink     `json:"backlinks,omitempty"`
}

func (s *Service) NoteMetadata(ctx context.Context, req MetadataRequest) (MetadataResult, error) {
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

	headings := make([]string, len(n.Headings))
	for i, h := range n.Headings {
		headings[i] = h.Text
	}
	blocks := make([]string, len(n.Blocks))
	for i, b := range n.Blocks {
		blocks[i] = b.ID
	}

	return MetadataResult{
		Path:        string(n.Path),
		Frontmatter: n.Frontmatter,
		Tags:        n.Tags,
		Aliases:     n.Aliases,
		Headings:    headings,
		Blocks:      blocks,
		Links:       n.Links,
		Backlinks:   s.index.Backlinks(cp),
	}, nil
}

type RuntimeStats struct {
	NumGoroutine int    `json:"num_goroutine"`
	Alloc        uint64 `json:"alloc"`
	TotalAlloc   uint64 `json:"total_alloc"`
	Sys          uint64 `json:"sys"`
	NumGC        uint32 `json:"num_gc"`
}

type StatsRequest struct {
	IncludeHealth  bool `json:"include_health"`
	IncludeRuntime bool `json:"include_runtime"`
}

type StatsResult struct {
	Notes        int           `json:"notes"`
	Assets       int           `json:"assets"`
	TotalSize    int64         `json:"total_size"`
	Orphans      int           `json:"orphans"`
	BrokenLinks  int           `json:"broken_links"`
	BrokenAnchor int           `json:"broken_anchors"`
	Collisions   int           `json:"alias_collisions"`
	Generation   uint64        `json:"generation"`
	Runtime      *RuntimeStats `json:"runtime,omitempty"`
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

	var orphans, brokenLinks, brokenAnchors int

	// NotePaths, nao Paths: Paths inclui anexos, que Get nao resolve.
	for _, p := range s.index.NotePaths() {
		n, ok := s.index.Get(p)
		if ok {
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

	res := StatsResult{
		Notes:        s.index.NoteCount(),
		Assets:       s.index.AssetCount(),
		TotalSize:    s.index.TotalSize(),
		Orphans:      orphans,
		BrokenLinks:  brokenLinks,
		BrokenAnchor: brokenAnchors,
		Collisions:   s.index.AliasCollisions(),
		Generation:   s.index.Generation(),
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
	}

	return res, nil
}
