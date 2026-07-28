package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/parser"
)

// ReadRequest sao os parametros de note_read. Heading e BlockID sao
// mutuamente exclusivos: os dois juntos descrevem dois recortes diferentes da
// mesma nota, e adivinhar qual vale seria decidir pelo cliente.
type ReadRequest struct {
	Path               string
	Heading            string
	HeadingLevel       int
	BlockID            string
	MaxBytes           int
	IncludeFrontmatter bool
}

// ReadResult e o retorno de note_read. Section vem preenchido so quando a
// leitura foi recortada por heading, e Truncated diz que MaxBytes cortou o
// resultado — sem ele o cliente nao distingue nota curta de nota cortada.
type ReadResult struct {
	Content   string
	Hash      string
	Section   *parser.Heading
	Truncated bool
}

// ReadNote le uma nota, ou um recorte dela por heading ou por bloco.
//
// Le apenas o intervalo de bytes pedido: e o offset guardado no indice que faz
// uma secao de 2 KB numa nota de 500 KB custar 2 KB.
func (s *Service) ReadNote(ctx context.Context, req ReadRequest) (ReadResult, error) {
	if req.Heading != "" && req.BlockID != "" {
		return ReadResult{}, Errorf(CodeInternal, "heading e block_id sao mutuamente exclusivos")
	}

	canonical, err := s.index.ResolvePath(req.Path)
	if err != nil {
		if errors.Is(err, index.ErrAmbiguousPath) {
			return ReadResult{}, Errorf(CodeAmbiguousPath, "caminho %q resolve para mais de um arquivo", req.Path)
		}
		if strings.Contains(req.Path, "../") {
			return ReadResult{}, Errorf(CodePathOutsideVault, "caminho %q fora do cofre", req.Path)
		}
		return ReadResult{}, Errorf(CodeNoteNotFound, "nota %q nao encontrada no indice", req.Path)
	}

	note, ok := s.index.Get(canonical)
	if !ok {
		return ReadResult{}, Errorf(CodeNoteNotFound, "nota %q nao encontrada", req.Path)
	}

	if note.CloudOnly {
		return ReadResult{}, Errorf(CodeCloudOnlyFile, "nota %q e apenas online (CloudOnly)", req.Path)
	}

	start := int64(0)
	end := note.Size
	var matchedHeading *parser.Heading

	switch {
	case req.Heading != "":
		targetSlug := parser.Slug(req.Heading)
		var matches []parser.Heading

		for _, h := range note.Headings {
			if parser.Slug(h.Text) == targetSlug {
				if req.HeadingLevel == 0 || h.Level == req.HeadingLevel {
					matches = append(matches, h)
				}
			}
		}

		if len(matches) == 0 {
			var alternatives []string
			for _, h := range note.Headings {
				if req.HeadingLevel == 0 || h.Level == req.HeadingLevel {
					alternatives = append(alternatives, h.Text)
				}
			}
			return ReadResult{}, Errorf(CodeHeadingNotFound, "heading %q nao encontrado. Disponiveis: %s", req.Heading, strings.Join(alternatives, ", "))
		}

		if len(matches) > 1 {
			return ReadResult{}, Errorf(CodeAmbiguousHeading, "heading %q ambiguo (%d ocorrencias)", req.Heading, len(matches))
		}

		matchedHeading = &matches[0]
		start = int64(matchedHeading.Start)
		end = int64(matchedHeading.End)

	case req.BlockID != "":
		found := false
		for _, b := range note.Blocks {
			if b.ID == req.BlockID {
				start = int64(b.Start)
				end = int64(b.End)
				found = true
				break
			}
		}
		if !found {
			return ReadResult{}, Errorf(CodeBlockNotFound, "bloco %q nao encontrado", req.BlockID)
		}

	case !req.IncludeFrontmatter:
		// O indice guarda offsets, nao conteudo, e o offset do corpo nao esta
		// entre eles — descobri-lo exige olhar os bytes.
		//
		// O erro de leitura NAO pode ser engolido. Antes ele era, e a
		// consequencia era pior que uma falha: start ficava em 0, a nota
		// voltava COM o frontmatter, e o cliente que pediu para nao receber
		// frontmatter recebia um resultado de sucesso contendo exatamente o
		// que ele excluiu.
		data, err := s.vault.ReadAll(ctx, canonical)
		if err != nil {
			return ReadResult{}, Wrap(CodeInternal, err, "lendo %q para localizar o inicio do corpo", req.Path)
		}
		_, _, bodyOffset := parser.SplitFrontmatter(data)
		start = bodyOffset
	}

	if req.MaxBytes > 0 && (end-start) > int64(req.MaxBytes) {
		end = start + int64(req.MaxBytes)
	}

	data, err := s.vault.ReadRange(ctx, canonical, start, end)
	if err != nil {
		return ReadResult{}, Errorf(CodeVaultUnavailable, "falha lendo faixa da nota: %v", err)
	}

	res := ReadResult{
		Content: string(data),
		Hash:    fmt.Sprintf("%016x", note.Hash),
		Section: matchedHeading,
	}

	if req.MaxBytes > 0 && (end-start) == int64(req.MaxBytes) { // better condition
		res.Truncated = true
	}

	return res, nil
}
