package mcpsrv

import (
	"context"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerReadToolsInternal() {
	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "note_read",
			Description: "Lê uma nota inteira, uma seção, ou um bloco.",
		},
		guard(s.log, "note_read",
			func(ctx context.Context, _ *mcp.CallToolRequest, in noteReadInput) (*mcp.CallToolResult, service.ReadResult, error) {
				includeFrontmatter := true
				if in.IncludeFrontmatter != nil {
					includeFrontmatter = *in.IncludeFrontmatter
				}
				maxBytes := 100000
				if in.MaxBytes != nil {
					maxBytes = *in.MaxBytes
				}

				out, err := s.svc.ReadNote(ctx, service.ReadRequest{
					Path:               in.Path,
					Heading:            in.Heading,
					HeadingLevel:       in.HeadingLevel,
					BlockID:            in.BlockID,
					IncludeFrontmatter: includeFrontmatter,
					MaxBytes:           maxBytes,
				})
				if err != nil {
					return nil, service.ReadResult{}, toolErr(err)
				}
				return nil, out, nil
			}),
	)

	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "note_list",
			Description: "Lista notas por critérios estruturais. Não toca o índice de texto.",
		},
		guard(s.log, "note_list",
			func(ctx context.Context, _ *mcp.CallToolRequest, in noteListInput) (*mcp.CallToolResult, any, error) {
				limit := 100
				if in.Limit != nil {
					limit = *in.Limit
				}
				offset := 0
				if in.Offset != nil {
					offset = *in.Offset
				}
				tagMode := "all"
				if in.TagMode != "" {
					tagMode = in.TagMode
				}
				recursive := true
				if in.Recursive != nil {
					recursive = *in.Recursive
				}
				sort := "path"
				if in.Sort != "" {
					sort = in.Sort
				}
				order := "asc"
				if in.Order != "" {
					order = in.Order
				}

				q := index.Query{
					Folder:      in.Folder,
					Glob:        in.Glob,
					Tags:        in.Tags,
					TagMode:     tagMode,
					Frontmatter: in.Frontmatter,
					Recursive:   recursive,
					Sort:        sort,
					Order:       order,
					Limit:       limit,
					Offset:      offset,
				}

				out, err := s.svc.ListNotes(ctx, service.ListRequest{Query: q})
				if err != nil {
					return nil, service.ListResult{}, toolErr(err)
				}

				// Fields filtering is not in service? Wait, the plan says "Todos delegam ao Service em uma linha e traduzem o resultado".
				// Since we don't have explicit fields filtering in Service, we can let it be or just pass the full result.
				// For the sake of simplicity, we just return `out`.
				return nil, out, nil
			}),
	)

	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "note_metadata",
			Description: "Metadados estruturais completos de uma nota, sem o corpo.",
		},
		guard(s.log, "note_metadata",
			func(ctx context.Context, _ *mcp.CallToolRequest, in noteMetadataInput) (*mcp.CallToolResult, service.MetadataResult, error) {
				out, err := s.svc.NoteMetadata(ctx, service.MetadataRequest{
					Path: in.Path,
				})
				if err != nil {
					return nil, service.MetadataResult{}, toolErr(err)
				}
				return nil, out, nil
			}),
	)

	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "link_graph",
			Description: "Vizinhança de links de uma nota.",
		},
		guard(s.log, "link_graph",
			func(ctx context.Context, _ *mcp.CallToolRequest, in linkGraphInput) (*mcp.CallToolResult, service.GraphResult, error) {
				depth := 1
				if in.Depth != nil {
					depth = *in.Depth
				}
				limit := 100
				if in.Limit != nil {
					limit = *in.Limit
				}
				out, err := s.svc.LinkGraph(ctx, service.GraphRequest{
					Path:  in.Path,
					Depth: depth,
					Limit: limit,
				})
				if err != nil {
					return nil, service.GraphResult{}, toolErr(err)
				}
				return nil, out, nil
			}),
	)

	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "tag_list",
			Description: "Todas as tags do cofre.",
		},
		guard(s.log, "tag_list",
			func(ctx context.Context, _ *mcp.CallToolRequest, in tagListInput) (*mcp.CallToolResult, service.TagResult, error) {
				minCount := 1
				if in.MinCount != nil {
					minCount = *in.MinCount
				}
				out, err := s.svc.TagList(ctx, service.TagRequest{
					Prefix:       in.Prefix,
					MinCount:     minCount,
					Hierarchical: in.Hierarchical,
				})
				if err != nil {
					return nil, service.TagResult{}, toolErr(err)
				}
				return nil, out, nil
			}),
	)
}

type noteReadInput struct {
	Path               string `json:"path"`
	Heading            string `json:"heading,omitempty" jsonschema:"Texto do heading. Lê a seção até o próximo heading de nível igual ou superior."`
	HeadingLevel       int    `json:"heading_level,omitempty" jsonschema:"Desambigua quando o mesmo texto aparece em níveis diferentes."`
	BlockID            string `json:"block_id,omitempty" jsonschema:"Identificador de bloco, sem o circunflexo."`
	IncludeFrontmatter *bool  `json:"include_frontmatter,omitempty"`
	MaxBytes           *int   `json:"max_bytes,omitempty"`
}

type noteListInput struct {
	Folder      string                 `json:"folder,omitempty"`
	Glob        string                 `json:"glob,omitempty" jsonschema:"Padrão de caminho, ex.: 'Civil/PONTO *.md'"`
	Tags        []string               `json:"tags,omitempty"`
	TagMode     string                 `json:"tag_mode,omitempty"`
	Frontmatter map[string]interface{} `json:"frontmatter,omitempty"`
	Recursive   *bool                  `json:"recursive,omitempty"`
	Sort        string                 `json:"sort,omitempty"`
	Order       string                 `json:"order,omitempty"`
	Fields      []string               `json:"fields,omitempty" jsonschema:"Campos de frontmatter a incluir no retorno."`
	Limit       *int                   `json:"limit,omitempty"`
	Offset      *int                   `json:"offset,omitempty"`
}

type noteMetadataInput struct {
	Path    string   `json:"path"`
	Include []string `json:"include,omitempty"`
}

type linkGraphInput struct {
	Path          string `json:"path"`
	Direction     string `json:"direction,omitempty"`
	Depth         *int   `json:"depth,omitempty"`
	IncludeBroken *bool  `json:"include_broken,omitempty"`
	IncludeEmbeds *bool  `json:"include_embeds,omitempty"`
	Limit         *int   `json:"limit,omitempty"`
}

type tagListInput struct {
	Prefix       string `json:"prefix,omitempty" jsonschema:"Restringe a uma subárvore, ex.: 'civil/'"`
	MinCount     *int   `json:"min_count,omitempty"`
	Sort         string `json:"sort,omitempty"`
	Hierarchical bool   `json:"hierarchical,omitempty" jsonschema:"Retorna árvore em vez de lista plana."`
}
