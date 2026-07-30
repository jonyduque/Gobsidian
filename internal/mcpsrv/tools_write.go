package mcpsrv

import (
	"context"

	"github.com/jonyd/gobsidian/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type createInput struct {
	Path          string         `json:"path" jsonschema:"caminho relativo da nota no cofre"`
	Content       string         `json:"content" jsonschema:"conteudo textual da nota"`
	Frontmatter   map[string]any `json:"frontmatter,omitempty" jsonschema:"metadados frontmatter em mapa chave/valor"`
	CreateFolders bool           `json:"create_folders,omitempty" jsonschema:"cria diretorios intermediarios se nao existirem"`
	DryRun        bool           `json:"dry_run,omitempty" jsonschema:"se verdadeiro devolve apenas o diff sem alterar o disco"`
}

type appendInput struct {
	Path            string `json:"path" jsonschema:"caminho relativo da nota"`
	Content         string `json:"content" jsonschema:"conteudo a anexar"`
	Heading         string `json:"heading,omitempty" jsonschema:"heading onde anexar; ausente anexa ao fim da nota"`
	HeadingLevel    int    `json:"heading_level,omitempty" jsonschema:"nivel do heading (1-6) para desambiguar"`
	CreateIfMissing bool   `json:"create_if_missing,omitempty" jsonschema:"cria o heading se nao existir"`
	EnsureBlankLine bool   `json:"ensure_blank_line,omitempty" jsonschema:"garante linha em branco antes do conteudo anexado"`
	ExpectedHash    string `json:"expected_hash,omitempty" jsonschema:"hash xxhash para concorrencia otimista"`
	DryRun          bool   `json:"dry_run,omitempty" jsonschema:"se verdadeiro devolve apenas o diff sem alterar o disco"`
}

type patchInput struct {
	Path         string `json:"path" jsonschema:"caminho relativo da nota"`
	Content      string `json:"content" jsonschema:"conteudo de substituicao"`
	Heading      string `json:"heading,omitempty" jsonschema:"heading alvo da substituicao"`
	HeadingLevel int    `json:"heading_level,omitempty" jsonschema:"nivel do heading (1-6)"`
	BlockID      string `json:"block_id,omitempty" jsonschema:"id de bloco (sem o ^) alvo da substituicao"`
	Mode         string `json:"mode,omitempty" jsonschema:"replace_section, replace_heading_and_section ou replace_block"`
	ExpectedHash string `json:"expected_hash,omitempty" jsonschema:"hash xxhash para concorrencia otimista"`
	DryRun       bool   `json:"dry_run,omitempty" jsonschema:"se verdadeiro devolve apenas o diff sem alterar o disco"`
}

func (s *Server) registerWriteTools() {
	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "note_create",
			Description: "Cria uma nova nota no cofre. Falha se o caminho ja existir.",
		},
		guard(s.log, "note_create",
			func(ctx context.Context, _ *mcp.CallToolRequest, in createInput) (*mcp.CallToolResult, service.CreateNoteResult, error) {
				out, err := s.svc.CreateNote(ctx, service.CreateNoteRequest{
					Path:          in.Path,
					Content:       in.Content,
					Frontmatter:   in.Frontmatter,
					CreateFolders: in.CreateFolders,
					DryRun:        in.DryRun,
				})
				if err != nil {
					return nil, service.CreateNoteResult{}, toolErr(err)
				}
				return nil, out, nil
			}),
	)

	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "note_append",
			Description: "Anexa conteudo ao final de uma nota ou ao final de uma secao especifica.",
		},
		guard(s.log, "note_append",
			func(ctx context.Context, _ *mcp.CallToolRequest, in appendInput) (*mcp.CallToolResult, service.AppendNoteResult, error) {
				out, err := s.svc.AppendNote(ctx, service.AppendNoteRequest{
					Path:            in.Path,
					Content:         in.Content,
					Heading:         in.Heading,
					HeadingLevel:    in.HeadingLevel,
					CreateIfMissing: in.CreateIfMissing,
					EnsureBlankLine: in.EnsureBlankLine,
					ExpectedHash:    in.ExpectedHash,
					DryRun:          in.DryRun,
				})
				if err != nil {
					return nil, service.AppendNoteResult{}, toolErr(err)
				}
				return nil, out, nil
			}),
	)

	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "note_patch",
			Description: "Substitui uma secao, cabecalho ou bloco de uma nota.",
		},
		guard(s.log, "note_patch",
			func(ctx context.Context, _ *mcp.CallToolRequest, in patchInput) (*mcp.CallToolResult, service.PatchNoteResult, error) {
				out, err := s.svc.PatchNote(ctx, service.PatchNoteRequest{
					Path:         in.Path,
					Content:      in.Content,
					Heading:      in.Heading,
					HeadingLevel: in.HeadingLevel,
					BlockID:      in.BlockID,
					Mode:         in.Mode,
					ExpectedHash: in.ExpectedHash,
					DryRun:       in.DryRun,
				})
				if err != nil {
					return nil, service.PatchNoteResult{}, toolErr(err)
				}
				return nil, out, nil
			}),
	)
}
