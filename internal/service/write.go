package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/writer"
	"gopkg.in/yaml.v3"
)

// CreateNoteRequest carrega os parametros para note_create.
type CreateNoteRequest struct {
	Path          string         `json:"path"`
	Content       string         `json:"content"`
	Frontmatter   map[string]any `json:"frontmatter,omitempty"`
	CreateFolders bool           `json:"create_folders"`
	DryRun        bool           `json:"dry_run"`
}

// CreateNoteResult e o retorno de note_create.
type CreateNoteResult struct {
	Path    string `json:"path"`
	Diff    string `json:"diff,omitempty"`
	Created bool   `json:"created"`
}

// AppendNoteRequest carrega os parametros para note_append.
type AppendNoteRequest struct {
	Path            string `json:"path"`
	Content         string `json:"content"`
	Heading         string `json:"heading,omitempty"`
	HeadingLevel    int    `json:"heading_level,omitempty"`
	CreateIfMissing bool   `json:"create_if_missing"`
	EnsureBlankLine bool   `json:"ensure_blank_line"`
	ExpectedHash    string `json:"expected_hash,omitempty"`
	DryRun          bool   `json:"dry_run"`
}

// AppendNoteResult e o retorno de note_append.
type AppendNoteResult struct {
	Path     string `json:"path"`
	Diff     string `json:"diff,omitempty"`
	Appended bool   `json:"appended"`
}

// PatchNoteRequest carrega os parametros para note_patch.
type PatchNoteRequest struct {
	Path         string `json:"path"`
	Content      string `json:"content"`
	Heading      string `json:"heading,omitempty"`
	HeadingLevel int    `json:"heading_level,omitempty"`
	BlockID      string `json:"block_id,omitempty"`
	Mode         string `json:"mode,omitempty"`
	ExpectedHash string `json:"expected_hash,omitempty"`
	DryRun       bool   `json:"dry_run"`
}

// PatchNoteResult e o retorno de note_patch.
type PatchNoteResult struct {
	Path    string `json:"path"`
	Diff    string `json:"diff,omitempty"`
	Patched bool   `json:"patched"`
}

func (s *Service) checkWriteAllowed(path string) error {
	if s.opts.ReadOnly {
		return Errorf(CodeReadOnlyMode, "servidor em modo somente leitura (--read-only)")
	}
	if strings.Contains(path, "../") || strings.HasPrefix(path, "/") || (len(path) > 1 && path[1] == ':') {
		return Errorf(CodePathOutsideVault, "caminho %q fora do cofre", path)
	}
	return nil
}

// CreateNote cria uma nova nota no cofre. Falha se a nota ja existir.
func (s *Service) CreateNote(_ context.Context, req CreateNoteRequest) (CreateNoteResult, error) {
	if err := s.checkWriteAllowed(req.Path); err != nil {
		return CreateNoteResult{}, err
	}

	cleanPath := filepath.ToSlash(filepath.Clean(req.Path))
	canonical := vault.CanonicalPath(cleanPath)

	if _, ok := s.index.Get(canonical); ok {
		return CreateNoteResult{}, Errorf(CodeNoteExists, "nota %q ja existe no cofre", req.Path)
	}

	absPath := s.vault.Abs(canonical)
	if _, err := os.Stat(absPath); err == nil {
		return CreateNoteResult{}, Errorf(CodeNoteExists, "nota %q ja existe no cofre", req.Path)
	}

	dir := filepath.Dir(absPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if !req.CreateFolders {
			return CreateNoteResult{}, Errorf(CodeFolderNotFound, "diretorio %q nao existe", filepath.Dir(req.Path))
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return CreateNoteResult{}, Errorf(CodeInternal, "criando diretorio %q: %v", dir, err)
		}
	}

	unlock := s.locker.Lock(canonical)
	defer unlock()

	var bodyBuilder strings.Builder
	if len(req.Frontmatter) > 0 {
		fmBytes, err := yaml.Marshal(req.Frontmatter)
		if err != nil {
			return CreateNoteResult{}, Errorf(CodeInternal, "formatando frontmatter YAML: %v", err)
		}
		bodyBuilder.WriteString("---\n")
		bodyBuilder.Write(fmBytes)
		bodyBuilder.WriteString("---\n\n")
	}
	bodyBuilder.WriteString(req.Content)
	fullContent := bodyBuilder.String()

	if req.DryRun {
		diff := writer.UnifiedDiff(req.Path, req.Path, "", fullContent, 3)
		return CreateNoteResult{Path: req.Path, Diff: diff, Created: false}, nil
	}

	if err := writer.WriteAtomic(absPath, []byte(fullContent)); err != nil {
		return CreateNoteResult{}, Errorf(CodeInternal, "escrevendo nota %q: %v", req.Path, err)
	}

	return CreateNoteResult{Path: req.Path, Diff: "", Created: true}, nil
}

// AppendNote anexa conteudo a uma nota ou secao existente.
func (s *Service) AppendNote(_ context.Context, req AppendNoteRequest) (AppendNoteResult, error) {
	if err := s.checkWriteAllowed(req.Path); err != nil {
		return AppendNoteResult{}, err
	}

	canonical, err := s.index.ResolvePath(req.Path)
	if err != nil {
		if errors.Is(err, index.ErrAmbiguousPath) {
			return AppendNoteResult{}, Errorf(CodeAmbiguousPath, "caminho %q e ambiguo", req.Path)
		}
		return AppendNoteResult{}, Errorf(CodeNoteNotFound, "nota %q nao encontrada", req.Path)
	}

	note, ok := s.index.Get(canonical)
	if !ok {
		return AppendNoteResult{}, Errorf(CodeNoteNotFound, "nota %q nao encontrada", req.Path)
	}
	if note.CloudOnly {
		return AppendNoteResult{}, Errorf(CodeCloudOnlyFile, "nota %q e apenas online (CloudOnly)", req.Path)
	}

	unlock := s.locker.Lock(canonical)
	defer unlock()

	absPath := s.vault.Abs(canonical)
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return AppendNoteResult{}, Errorf(CodeInternal, "lendo nota %q: %v", req.Path, err)
	}

	currentHash := fmt.Sprintf("%016x", note.Hash)
	if req.ExpectedHash != "" && currentHash != req.ExpectedHash {
		return AppendNoteResult{}, Errorf(CodeHashMismatch, "hash esperado %q nao confere com hash atual %q", req.ExpectedHash, currentHash)
	}

	cleaned, hadBOM := vault.StripBOM(raw)
	parsed, _ := parser.Parse(cleaned)
	if hadBOM {
		parsed.ShiftOffsets(int64(vault.BOMLen))
	}

	var targetH *parser.Heading
	if req.Heading != "" {
		h, err := writer.FindHeading(parsed.Headings, req.Heading)
		if err != nil {
			var hnf *writer.HeadingNotFoundError
			var amb *writer.AmbiguousHeadingError
			if errors.As(err, &amb) {
				return AppendNoteResult{}, Errorf(CodeAmbiguousHeading, "%v", err)
			}
			if errors.As(err, &hnf) {
				if !req.CreateIfMissing {
					return AppendNoteResult{}, Errorf(CodeHeadingNotFound, "%v", err)
				}
			}
		} else {
			targetH = h
		}
	}

	var proposed []byte
	if req.Heading != "" && targetH == nil && req.CreateIfMissing {
		eol := writer.DetectEOL(raw)
		level := req.HeadingLevel
		if level <= 0 {
			level = 2
		}
		headingLine := fmt.Sprintf("%s %s%s", strings.Repeat("#", level), req.Heading, eol)
		appended := writer.AppendSectionContent(raw, nil, headingLine+req.Content)
		proposed = appended
	} else {
		proposed = writer.AppendSectionContent(raw, targetH, req.Content)
	}

	if req.DryRun {
		diff := writer.UnifiedDiff(req.Path, req.Path, string(raw), string(proposed), 3)
		return AppendNoteResult{Path: req.Path, Diff: diff, Appended: false}, nil
	}

	if err := writer.WriteAtomic(absPath, proposed); err != nil {
		return AppendNoteResult{}, Errorf(CodeInternal, "escrevendo nota %q: %v", req.Path, err)
	}

	return AppendNoteResult{Path: req.Path, Diff: "", Appended: true}, nil
}

// PatchNote substitui uma secao, cabeçalho ou bloco de uma nota.
func (s *Service) PatchNote(_ context.Context, req PatchNoteRequest) (PatchNoteResult, error) {
	if err := s.checkWriteAllowed(req.Path); err != nil {
		return PatchNoteResult{}, err
	}

	if req.Heading != "" && req.BlockID != "" {
		return PatchNoteResult{}, Errorf(CodeInternal, "heading e block_id sao mutuamente exclusivos em note_patch")
	}

	canonical, err := s.index.ResolvePath(req.Path)
	if err != nil {
		if errors.Is(err, index.ErrAmbiguousPath) {
			return PatchNoteResult{}, Errorf(CodeAmbiguousPath, "caminho %q e ambiguo", req.Path)
		}
		return PatchNoteResult{}, Errorf(CodeNoteNotFound, "nota %q nao encontrada", req.Path)
	}

	note, ok := s.index.Get(canonical)
	if !ok {
		return PatchNoteResult{}, Errorf(CodeNoteNotFound, "nota %q nao encontrada", req.Path)
	}
	if note.CloudOnly {
		return PatchNoteResult{}, Errorf(CodeCloudOnlyFile, "nota %q e apenas online (CloudOnly)", req.Path)
	}

	unlock := s.locker.Lock(canonical)
	defer unlock()

	absPath := s.vault.Abs(canonical)
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return PatchNoteResult{}, Errorf(CodeInternal, "lendo nota %q: %v", req.Path, err)
	}

	currentHash := fmt.Sprintf("%016x", note.Hash)
	if req.ExpectedHash != "" && currentHash != req.ExpectedHash {
		return PatchNoteResult{}, Errorf(CodeHashMismatch, "hash esperado %q nao confere com hash atual %q", req.ExpectedHash, currentHash)
	}

	cleaned, hadBOM := vault.StripBOM(raw)
	parsed, _ := parser.Parse(cleaned)
	if hadBOM {
		parsed.ShiftOffsets(int64(vault.BOMLen))
	}

	var proposed []byte

	mode := req.Mode
	if mode == "" {
		if req.BlockID != "" {
			mode = "replace_block"
		} else {
			mode = "replace_section"
		}
	}

	switch mode {
	case "replace_block":
		b, err := writer.FindBlock(parsed.Blocks, req.BlockID)
		if err != nil {
			var bnf *writer.BlockNotFoundError
			var amb *writer.AmbiguousBlockError
			if errors.As(err, &amb) {
				return PatchNoteResult{}, Errorf(CodeAmbiguousBlock, "%v", err)
			}
			if errors.As(err, &bnf) {
				return PatchNoteResult{}, Errorf(CodeBlockNotFound, "%v", err)
			}
			return PatchNoteResult{}, Errorf(CodeBlockNotFound, "bloco %q nao encontrado", req.BlockID)
		}
		proposed = writer.ReplaceBlockContent(raw, *b, req.Content)

	case "replace_heading_and_section":
		h, err := writer.FindHeading(parsed.Headings, req.Heading)
		if err != nil {
			var hnf *writer.HeadingNotFoundError
			var amb *writer.AmbiguousHeadingError
			if errors.As(err, &amb) {
				return PatchNoteResult{}, Errorf(CodeAmbiguousHeading, "%v", err)
			}
			if errors.As(err, &hnf) {
				return PatchNoteResult{}, Errorf(CodeHeadingNotFound, "%v", err)
			}
			return PatchNoteResult{}, Errorf(CodeHeadingNotFound, "heading %q nao encontrado", req.Heading)
		}
		// Substitui a partir de h.Start (incluindo o titulo do heading) ate h.End
		eol := writer.DetectEOL(raw)
		normReplacement := writer.NormalizeEOL(req.Content, eol)
		if normReplacement != "" && !strings.HasSuffix(normReplacement, eol) {
			normReplacement += eol
		}
		var buf bytes.Buffer
		buf.Write(raw[:h.Start])
		buf.WriteString(normReplacement)
		buf.Write(raw[h.End:])
		proposed = buf.Bytes()

	case "replace_section":
		h, err := writer.FindHeading(parsed.Headings, req.Heading)
		if err != nil {
			var hnf *writer.HeadingNotFoundError
			var amb *writer.AmbiguousHeadingError
			if errors.As(err, &amb) {
				return PatchNoteResult{}, Errorf(CodeAmbiguousHeading, "%v", err)
			}
			if errors.As(err, &hnf) {
				return PatchNoteResult{}, Errorf(CodeHeadingNotFound, "%v", err)
			}
			return PatchNoteResult{}, Errorf(CodeHeadingNotFound, "heading %q nao encontrado", req.Heading)
		}
		proposed = writer.PatchSectionContent(raw, *h, req.Content)

	default:
		return PatchNoteResult{}, Errorf(CodeInternal, "modo de patch %q invalido", req.Mode)
	}

	if req.DryRun {
		diff := writer.UnifiedDiff(req.Path, req.Path, string(raw), string(proposed), 3)
		return PatchNoteResult{Path: req.Path, Diff: diff, Patched: false}, nil
	}

	if err := writer.WriteAtomic(absPath, proposed); err != nil {
		return PatchNoteResult{}, Errorf(CodeInternal, "escrevendo nota %q: %v", req.Path, err)
	}

	return PatchNoteResult{Path: req.Path, Diff: "", Patched: true}, nil
}
