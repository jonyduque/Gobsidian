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
	CreateFolders *bool          `json:"create_folders,omitempty" jsonschema:"cria diretorios intermediarios se nao existirem (padrao: true)"`
	DryRun        bool           `json:"dry_run,omitempty" jsonschema:"se verdadeiro devolve apenas o diff sem alterar o disco"`
}

type appendInput struct {
	Path            string `json:"path" jsonschema:"caminho relativo da nota"`
	Content         string `json:"content" jsonschema:"conteudo a anexar"`
	Heading         string `json:"heading,omitempty" jsonschema:"heading onde anexar; ausente anexa ao fim da nota"`
	HeadingLevel    int    `json:"heading_level,omitempty" jsonschema:"nivel do heading (1-6) para desambiguar"`
	CreateIfMissing bool   `json:"create_if_missing,omitempty" jsonschema:"cria o heading se nao existir"`
	EnsureBlankLine *bool  `json:"ensure_blank_line,omitempty" jsonschema:"garante linha em branco antes do conteudo anexado (padrao: true)"`
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

type moveInput struct {
	From          string `json:"from" jsonschema:"caminho relativo da nota de origem"`
	To            string `json:"to" jsonschema:"caminho relativo do destino"`
	UpdateLinks   *bool  `json:"update_links,omitempty" jsonschema:"se verdadeiro reescreve links apontando para o antigo caminho (padrao: true)"`
	CreateFolders *bool  `json:"create_folders,omitempty" jsonschema:"cria diretorios intermediarios se nao existirem (padrao: true)"`
	DryRun        bool   `json:"dry_run,omitempty" jsonschema:"se verdadeiro devolve apenas os diffs sem alterar o disco"`
}

type deleteInput struct {
	Path              string `json:"path" jsonschema:"caminho relativo da nota a excluir"`
	ToTrash           *bool  `json:"to_trash,omitempty" jsonschema:"se verdadeiro move para a lixeira .trash/ em vez de exclusao definitiva (padrao: true)"`
	ReportBrokenLinks *bool  `json:"report_broken_links,omitempty" jsonschema:"se verdadeiro lista notas cujos links ficarao quebrados (padrao: true)"`
	DryRun            bool   `json:"dry_run,omitempty" jsonschema:"se verdadeiro devolve apenas o relatorio sem excluir o arquivo"`
}

func (s *Server) registerWriteTools() {
	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "note_create",
			Description: "Cria uma nova nota no cofre. Falha se o caminho ja existir.",
		},
		guard(s.log, "note_create",
			func(ctx context.Context, _ *mcp.CallToolRequest, in createInput) (*mcp.CallToolResult, service.CreateNoteResult, error) {
				// docs/TOOLS.md declara "default": true. Com bool simples o
				// valor omitido chega como false e a promessa do schema quebra:
				// quem le o contrato omite o campo esperando true e recebe o
				// contrario. E a armadilha que ReadOnlySet e DebounceMSSet
				// existem para evitar, na camada da tool.
				createFolders := true
				if in.CreateFolders != nil {
					createFolders = *in.CreateFolders
				}

				out, err := s.svc.CreateNote(ctx, service.CreateNoteRequest{
					Path:          in.Path,
					Content:       in.Content,
					Frontmatter:   in.Frontmatter,
					CreateFolders: createFolders,
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
				// Mesmo motivo do create_folders: docs/TOOLS.md declara
				// "default": true para ensure_blank_line.
				ensureBlankLine := true
				if in.EnsureBlankLine != nil {
					ensureBlankLine = *in.EnsureBlankLine
				}

				out, err := s.svc.AppendNote(ctx, service.AppendNoteRequest{
					Path:            in.Path,
					Content:         in.Content,
					Heading:         in.Heading,
					HeadingLevel:    in.HeadingLevel,
					CreateIfMissing: in.CreateIfMissing,
					EnsureBlankLine: ensureBlankLine,
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

	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "note_move",
			Description: "Move ou renomeia uma nota no cofre e atualiza os links que apontam para ela.",
		},
		guard(s.log, "note_move",
			func(ctx context.Context, _ *mcp.CallToolRequest, in moveInput) (*mcp.CallToolResult, service.MoveNoteResult, error) {
				updateLinks := true
				if in.UpdateLinks != nil {
					updateLinks = *in.UpdateLinks
				}
				createFolders := true
				if in.CreateFolders != nil {
					createFolders = *in.CreateFolders
				}

				out, err := s.svc.MoveNote(ctx, service.MoveNoteRequest{
					From:          in.From,
					To:            in.To,
					UpdateLinks:   updateLinks,
					CreateFolders: createFolders,
					DryRun:        in.DryRun,
				})
				if err != nil {
					return nil, service.MoveNoteResult{}, toolErr(err)
				}
				return nil, out, nil
			}),
	)

	mcp.AddTool(s.mcp,
		&mcp.Tool{
			Name:        "note_delete",
			Description: "Exclui uma nota do cofre (por padrao movendo para a lixeira .trash/).",
		},
		guard(s.log, "note_delete",
			func(ctx context.Context, _ *mcp.CallToolRequest, in deleteInput) (*mcp.CallToolResult, service.DeleteNoteResult, error) {
				toTrash := true
				if in.ToTrash != nil {
					toTrash = *in.ToTrash
				}
				reportBrokenLinks := true
				if in.ReportBrokenLinks != nil {
					reportBrokenLinks = *in.ReportBrokenLinks
				}

				out, err := s.svc.DeleteNote(ctx, service.DeleteNoteRequest{
					Path:              in.Path,
					ToTrash:           toTrash,
					ReportBrokenLinks: reportBrokenLinks,
					DryRun:            in.DryRun,
				})
				if err != nil {
					return nil, service.DeleteNoteResult{}, toolErr(err)
				}
				return nil, out, nil
			}),
	)
}
