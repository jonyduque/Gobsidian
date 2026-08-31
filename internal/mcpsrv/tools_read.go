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
					t, err := parseDateFilter(in.ModifiedAfter)
					if err != nil {
						return nil, service.SearchResult{}, toolErr(service.Errorf(service.CodeInvalidArgument, "modified_after invalido: %v", err))
					}
					modAfter = &t
				}
				if in.ModifiedBefore != "" {
					t, err := parseDateFilter(in.ModifiedBefore)
					if err != nil {
						return nil, service.SearchResult{}, toolErr(service.Errorf(service.CodeInvalidArgument, "modified_before invalido: %v", err))
					}
					modBefore = &t
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
			Name: "note_outline",
			Description: "Mapa de uma nota: os headings Markdown reais e os CANDIDATOS a título — parágrafo em negrito e setext — que uma nota convertida de PDF, DOCX ou EPUB usa no lugar de '#'. " +
				"Os offsets são absolutos: use 'start' em note_read(offset=start, max_bytes=end-start) para ler a seção.",
		},
		guard(s.log, "note_outline",
			func(ctx context.Context, _ *mcp.CallToolRequest, in noteOutlineInput) (*mcp.CallToolResult, service.OutlineResult, error) {
				maxCandidatos := 0
				if in.MaxCandidates != nil {
					maxCandidatos = *in.MaxCandidates
				}
				out, err := s.svc.Outline(ctx, service.OutlineRequest{
					Path:          in.Path,
					MaxCandidates: maxCandidatos,
				})
				if err != nil {
					return nil, service.OutlineResult{}, toolErr(err)
				}
				return nil, out, nil
			}),
	)

	// O schema de note_read e montado, e nao inferido, porque `paths` aceita
	// string OU objeto — ver schemaDoNoteRead. O SDK so infere quando
	// InputSchema e nil, entao preencher aqui e o que preserva o oneOf.
	//
	// Falha aqui e erro de PROGRAMACAO — a inferencia roda sobre um tipo
	// estatico e so falha se noteReadInput ganhar um campo irrepresentavel.
	// panic e o mesmo contrato de mcp.AddTool, que ja entra em panico por
	// schema invalido; devolver erro exigiria mudar a assinatura de New, que
	// nao tem como se recuperar disto de qualquer forma.
	esquemaNoteRead, err := schemaDoNoteRead()
	if err != nil {
		panic(fmt.Sprintf("note_read: %v", err))
	}
	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "note_read",
			Description: "Lê uma nota inteira, uma seção, ou um bloco. Aceita 'paths' para ler várias notas numa só chamada; cada item de 'paths' pode ser um caminho ou um objeto que sobrepõe os campos de topo só para ele.",
			InputSchema: esquemaNoteRead,
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
					alvos := make([]service.ReadAlvo, len(req.Paths))
					for i, a := range req.Paths {
						alvos[i] = service.ReadAlvo{
							Path:               a.Path,
							Heading:            a.Heading,
							HeadingLevel:       a.HeadingLevel,
							BlockID:            a.BlockID,
							Offset:             a.Offset,
							MaxBytes:           a.MaxBytes,
							IncludeFrontmatter: a.IncludeFrontmatter,
						}
					}
					out := s.svc.ReadNotes(ctx, service.ReadBatchRequest{
						Alvos:              alvos,
						Heading:            req.Heading,
						HeadingLevel:       req.HeadingLevel,
						BlockID:            req.BlockID,
						Offset:             req.Offset,
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
					Offset:             req.Offset,
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
					Sort:         in.Sort,
					Hierarchical: in.Hierarchical,
				})
				if err != nil {
					return nil, service.TagResult{}, toolErr(err)
				}
				return nil, out, nil
			}),
	)
}

func parseDateFilter(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("formato invalido: %q (esperado RFC3339 ou YYYY-MM-DD)", s)
}

type vaultSearchInput struct {
	Query          string                 `json:"query,omitempty" jsonschema:"Termos de busca. Aspas duplas delimitam frase exata."`
	Folder         string                 `json:"folder,omitempty" jsonschema:"Restringe a uma pasta e suas subpastas."`
	Tags           []string               `json:"tags,omitempty" jsonschema:"Notas que contenham TODAS as tags."`
	Frontmatter    map[string]interface{} `json:"frontmatter,omitempty" jsonschema:"Pares chave/valor que devem casar no frontmatter."`
	ModifiedAfter  string                 `json:"modified_after,omitempty" jsonschema:"Data mínima de modificação. Aceita RFC3339 ('2006-01-02T15:04:05Z07:00') ou data curta ('2006-01-02')."`
	ModifiedBefore string                 `json:"modified_before,omitempty" jsonschema:"Data máxima de modificação. Aceita RFC3339 ('2006-01-02T15:04:05Z07:00') ou data curta ('2006-01-02')."`
	SnippetChars   *int                   `json:"snippet_chars,omitempty" jsonschema:"Tamanho máximo do trecho em caracteres. Teto máximo: 1000."`
	Limit          *int                   `json:"limit,omitempty"`
	Offset         *int                   `json:"offset,omitempty"`
}

type noteReadInput struct {
	Path               string         `json:"path,omitempty" jsonschema:"Caminho de uma nota. Mutuamente exclusivo com paths."`
	Paths              []noteReadAlvo `json:"paths,omitempty" jsonschema:"Vários caminhos numa só chamada, até 50. Cada item é um caminho ou um objeto que sobrepõe os campos de topo só para ele. Mutuamente exclusivo com path; falha de um item não derruba os demais."`
	Heading            string         `json:"heading,omitempty" jsonschema:"Texto do heading. Lê a seção até o próximo heading de nível igual ou superior."`
	HeadingLevel       int            `json:"heading_level,omitempty" jsonschema:"Desambigua quando o mesmo texto aparece em níveis diferentes."`
	BlockID            string         `json:"block_id,omitempty" jsonschema:"Identificador de bloco, sem o circunflexo."`
	Offset             *int64         `json:"offset,omitempty" jsonschema:"Offset de byte a partir do inicio da nota (byte 0). Mutuamente exclusivo com heading e block_id. Ignora include_frontmatter."`
	IncludeFrontmatter *bool          `json:"include_frontmatter,omitempty"`
	MaxBytes           *int           `json:"max_bytes,omitempty" jsonschema:"Aplica-se por nota, não ao lote inteiro."`
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
	Sort         string `json:"sort,omitempty" jsonschema:"Ordenação: 'name' (crescente por nome) ou 'count' (decrescente por contagem, desempate por nome). Padrão: 'name'."`
	Hierarchical bool   `json:"hierarchical,omitempty" jsonschema:"Retorna árvore em vez de lista plana."`
}

type noteOutlineInput struct {
	Path string `json:"path" jsonschema:"Caminho da nota."`
	// Teto declarado, e nao silencioso: uma nota convertida de livro produz
	// centenas de candidatos, e cortar sem dizer e a mesma classe de defeito
	// que este projeto ja pagou em outros retornos paginados.
	MaxCandidates *int `json:"max_candidates,omitempty" jsonschema:"Máximo de candidatos devolvidos. Padrão 200, teto 1000. O retorno traz 'truncated' quando corta."`
}
