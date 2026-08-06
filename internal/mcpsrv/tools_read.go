package mcpsrv

import (
	"context"
	"fmt"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxPathsPorLote e o teto de itens que note_read aceita em req.Paths numa
// unica chamada. Sem ele, um lote pede o cofre inteiro e o servidor
// materializa centenas de MB em memoria para uma resposta que o cliente MCP
// nao consegue ler de uma vez.
const maxPathsPorLote = 50

func (s *Server) registerReadToolsInternal() {
	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "vault_search",
			Description: "Busca full-text com ranking, combinável com filtros de metadados.",
		},
		guard(s.log, "vault_search",
			func(ctx context.Context, _ *mcp.CallToolRequest, in vaultSearchInput) (*mcp.CallToolResult, service.SearchResult, error) {
				snippetChars := 240
				if in.SnippetChars != nil {
					snippetChars = *in.SnippetChars
				}
				limit := 20
				if in.Limit != nil {
					limit = *in.Limit
				}
				offset := 0
				if in.Offset != nil {
					offset = *in.Offset
				}

				var modAfter, modBefore *time.Time
				if in.ModifiedAfter != "" {
					if t, err := time.Parse(time.RFC3339, in.ModifiedAfter); err == nil {
						modAfter = &t
					}
				}
				if in.ModifiedBefore != "" {
					if t, err := time.Parse(time.RFC3339, in.ModifiedBefore); err == nil {
						modBefore = &t
					}
				}

				out, err := s.svc.Search(ctx, service.SearchOptions{
					Query:          in.Query,
					Folder:         in.Folder,
					Tags:           in.Tags,
					Frontmatter:    in.Frontmatter,
					ModifiedAfter:  modAfter,
					ModifiedBefore: modBefore,
					SnippetChars:   snippetChars,
					Limit:          limit,
					Offset:         offset,
				})
				if err != nil {
					return nil, service.SearchResult{}, toolErr(err)
				}
				return nil, out, nil
			}),
	)

	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "note_read",
			Description: "Lê uma nota inteira, uma seção, ou um bloco. Aceita 'paths' para ler várias notas numa só chamada.",
		},
		guard(s.log, "note_read",
			func(ctx context.Context, _ *mcp.CallToolRequest, req noteReadInput) (*mcp.CallToolResult, any, error) {
				// path e paths sao mutuamente exclusivos por decisao fechada do
				// brief: os dois preenchidos e erro de validacao, nao precedencia
				// silenciosa que decidiria pelo cliente qual valer.
				if req.Path != "" && len(req.Paths) > 0 {
					return noteReadValidationError("path e paths sao mutuamente exclusivos; preencha apenas um dos dois")
				}
				if len(req.Paths) > maxPathsPorLote {
					return noteReadValidationError(fmt.Sprintf("paths tem %d itens; o maximo por chamada e %d", len(req.Paths), maxPathsPorLote))
				}

				includeFrontmatter := true
				if req.IncludeFrontmatter != nil {
					includeFrontmatter = *req.IncludeFrontmatter
				}
				maxBytes := 100000
				if req.MaxBytes != nil {
					maxBytes = *req.MaxBytes
				}

				if len(req.Paths) > 0 {
					out := s.svc.ReadNotes(ctx, service.ReadBatchRequest{
						Paths:              req.Paths,
						Heading:            req.Heading,
						HeadingLevel:       req.HeadingLevel,
						BlockID:            req.BlockID,
						IncludeFrontmatter: includeFrontmatter,
						MaxBytes:           maxBytes,
					})
					return nil, out, nil
				}

				out, err := s.svc.ReadNote(ctx, service.ReadRequest{
					Path:               req.Path,
					Heading:            req.Heading,
					HeadingLevel:       req.HeadingLevel,
					BlockID:            req.BlockID,
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

				out, err := s.svc.ListNotes(ctx, service.ListRequest{Query: q, Fields: in.Fields})
				if err != nil {
					return nil, service.ListResult{}, toolErr(err)
				}

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
					Path:    in.Path,
					Include: in.Include,
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
				direction := "both"
				if in.Direction != "" {
					direction = in.Direction
				}
				includeBroken := true
				if in.IncludeBroken != nil {
					includeBroken = *in.IncludeBroken
				}
				includeEmbeds := true
				if in.IncludeEmbeds != nil {
					includeEmbeds = *in.IncludeEmbeds
				}

				out, err := s.svc.LinkGraph(ctx, service.GraphRequest{
					Path:          in.Path,
					Direction:     direction,
					Depth:         depth,
					IncludeBroken: includeBroken,
					IncludeEmbeds: includeEmbeds,
					Limit:         limit,
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

type vaultSearchInput struct {
	Query          string                 `json:"query,omitempty" jsonschema:"Termos de busca. Aspas duplas delimitam frase exata."`
	Folder         string                 `json:"folder,omitempty" jsonschema:"Restringe a uma pasta e suas subpastas."`
	Tags           []string               `json:"tags,omitempty" jsonschema:"Notas que contenham TODAS as tags."`
	Frontmatter    map[string]interface{} `json:"frontmatter,omitempty" jsonschema:"Pares chave/valor que devem casar no frontmatter."`
	ModifiedAfter  string                 `json:"modified_after,omitempty"`
	ModifiedBefore string                 `json:"modified_before,omitempty"`
	SnippetChars   *int                   `json:"snippet_chars,omitempty"`
	Limit          *int                   `json:"limit,omitempty"`
	Offset         *int                   `json:"offset,omitempty"`
}

type noteReadInput struct {
	Path               string   `json:"path,omitempty" jsonschema:"Caminho de uma nota. Mutuamente exclusivo com paths."`
	Paths              []string `json:"paths,omitempty" jsonschema:"Vários caminhos numa só chamada, até 50. Mutuamente exclusivo com path; falha de um item não derruba os demais."`
	Heading            string   `json:"heading,omitempty" jsonschema:"Texto do heading. Lê a seção até o próximo heading de nível igual ou superior."`
	HeadingLevel       int      `json:"heading_level,omitempty" jsonschema:"Desambigua quando o mesmo texto aparece em níveis diferentes."`
	BlockID            string   `json:"block_id,omitempty" jsonschema:"Identificador de bloco, sem o circunflexo."`
	IncludeFrontmatter *bool    `json:"include_frontmatter,omitempty"`
	MaxBytes           *int     `json:"max_bytes,omitempty" jsonschema:"Aplica-se por nota, não ao lote inteiro."`
}

// noteReadValidationError monta o CallToolResult de erro a mao, em vez de
// devolver err para o SDK. Handler que devolve error Go faz o SDK montar
// IsError sem StructuredContent (ver toolErr) — mas um erro de validacao de
// lote ainda se beneficia de Out estruturado, porque o cliente pode
// inspecionar o codigo do erro por campo em vez de reparsear texto.
func noteReadValidationError(msg string) (*mcp.CallToolResult, any, error) {
	valErr := service.Errorf(service.CodeInvalidArgument, "%s", msg)
	res := &mcp.CallToolResult{}
	res.SetError(fmt.Errorf("%s: %w", service.CodeInvalidArgument, valErr))
	out := service.ReadBatchResult{Items: []service.ReadNoteItem{{Err: valErr}}}
	return res, out, nil
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
