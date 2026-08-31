package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/writer"
	"gopkg.in/yaml.v3"
)

func hashDoConteudo(data []byte) string {
	return fmt.Sprintf("%016x", xxhash.Sum64(data))
}

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
	Hash    string `json:"hash,omitempty"`
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
	Hash     string `json:"hash,omitempty"`
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
	Hash    string `json:"hash,omitempty"`
}

func mapVaultErr(err error) *Error {
	if errors.Is(err, vault.ErrOutsideVault) || errors.Is(err, vault.ErrAbsolutePath) {
		return Errorf(CodePathOutsideVault, "%v", err)
	}
	if errors.Is(err, vault.ErrEmptyPath) || errors.Is(err, vault.ErrInvalidPath) {
		return Errorf(CodeInvalidArgument, "%v", err)
	}
	return Errorf(CodeInternal, "%v", err)
}

func (s *Service) checkWritable() error {
	if s.opts.ReadOnly {
		return Errorf(CodeReadOnlyMode, "servidor em modo somente leitura (--read-only)")
	}
	return nil
}

// CreateNote cria uma nova nota no cofre. Falha se a nota ja existir.
func (s *Service) CreateNote(ctx context.Context, req CreateNoteRequest) (CreateNoteResult, error) {
	if err := s.checkWritable(); err != nil {
		return CreateNoteResult{}, err
	}

	absPath, canonical, err := vault.Resolve(s.vault.Root(), req.Path)
	if err != nil {
		return CreateNoteResult{}, mapVaultErr(err)
	}

	if _, ok := s.index.Get(canonical); ok {
		return CreateNoteResult{}, Errorf(CodeNoteExists, "nota %q ja existe no cofre", req.Path)
	}
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
		return CreateNoteResult{Path: req.Path, Diff: diff, Created: false, Hash: hashDoConteudo([]byte(fullContent))}, nil
	}

	if err := writer.WriteAtomic(ctx, absPath, []byte(fullContent)); err != nil {
		return CreateNoteResult{}, Errorf(CodeInternal, "escrevendo nota %q: %v", req.Path, err)
	}

	return CreateNoteResult{Path: req.Path, Diff: "", Created: true, Hash: hashDoConteudo([]byte(fullContent))}, nil
}

// AppendNote anexa conteudo a uma nota ou secao existente.
func (s *Service) AppendNote(ctx context.Context, req AppendNoteRequest) (AppendNoteResult, error) {
	if err := s.checkWritable(); err != nil {
		return AppendNoteResult{}, err
	}

	canonical, err := s.index.ResolvePath(req.Path)
	if err != nil {
		return AppendNoteResult{}, ErroDeResolucao(req.Path, err)
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

	currentHash := hashDoConteudo(raw)
	if req.ExpectedHash != "" && currentHash != req.ExpectedHash {
		return AppendNoteResult{}, Errorf(CodeHashMismatch, "hash esperado %q nao confere com hash atual %q", req.ExpectedHash, currentHash)
	}

	cleaned, hadBOM := vault.StripBOM(raw)
	parsed := parser.Parse(cleaned)
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

	// ensure_blank_line separa o conteudo anexado do que ja estava la. Ate
	// 2026-07-30 o campo era declarado em docs/TOOLS.md com "default": true, na
	// struct de entrada da tool e nesta request — e nunca era lido. Fazia nada.
	// E o defeito que note_list.fields ja tinha custado: o modelo do outro lado
	// le o schema para decidir, pede o comportamento, e nao tem como saber que
	// o pedido nao fez efeito. A regra deste projeto e "ou implemente, ou tire
	// do schema e da documentacao"; implementado.
	//
	// O EOL vem do arquivo, nao de uma constante: escrever LF numa nota CRLF
	// produz diff de arquivo inteiro em cofre versionado por Git (RF-38).
	eol := writer.DetectEOL(raw)
	conteudo := req.Content
	if req.EnsureBlankLine && len(raw) > 0 && !strings.HasPrefix(conteudo, eol) {
		conteudo = eol + conteudo
	}

	var proposed []byte
	if req.Heading != "" && targetH == nil && req.CreateIfMissing {
		level := req.HeadingLevel
		if level <= 0 {
			level = 2
		}
		headingLine := fmt.Sprintf("%s %s%s", strings.Repeat("#", level), req.Heading, eol)
		appended := writer.AppendSectionContent(raw, nil, headingLine+conteudo)
		proposed = appended
	} else {
		proposed = writer.AppendSectionContent(raw, targetH, conteudo)
	}

	if req.DryRun {
		diff := writer.UnifiedDiff(req.Path, req.Path, string(raw), string(proposed), 3)
		return AppendNoteResult{Path: req.Path, Diff: diff, Appended: false, Hash: hashDoConteudo(proposed)}, nil
	}

	if err := writer.WriteAtomic(ctx, absPath, proposed); err != nil {
		return AppendNoteResult{}, Errorf(CodeInternal, "escrevendo nota %q: %v", req.Path, err)
	}

	return AppendNoteResult{Path: req.Path, Diff: "", Appended: true, Hash: hashDoConteudo(proposed)}, nil
}

// PatchNote substitui uma secao, cabeçalho ou bloco de uma nota.
func (s *Service) PatchNote(ctx context.Context, req PatchNoteRequest) (PatchNoteResult, error) {
	if err := s.checkWritable(); err != nil {
		return PatchNoteResult{}, err
	}

	if req.Heading != "" && req.BlockID != "" {
		return PatchNoteResult{}, Errorf(CodeInternal, "heading e block_id sao mutuamente exclusivos em note_patch")
	}

	canonical, err := s.index.ResolvePath(req.Path)
	if err != nil {
		return PatchNoteResult{}, ErroDeResolucao(req.Path, err)
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

	currentHash := hashDoConteudo(raw)
	if req.ExpectedHash != "" && currentHash != req.ExpectedHash {
		return PatchNoteResult{}, Errorf(CodeHashMismatch, "hash esperado %q nao confere com hash atual %q", req.ExpectedHash, currentHash)
	}

	cleaned, hadBOM := vault.StripBOM(raw)
	parsed := parser.Parse(cleaned)
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
		// INVALID_ARGUMENT, e nao INTERNAL: o modo veio do cliente, e INTERNAL
		// diz a ele que o servidor quebrou — o que faz o host tentar de novo
		// em vez de corrigir o pedido (achado B4).
		return PatchNoteResult{}, Errorf(CodeInvalidArgument,
			"mode = %q invalido; aceitos: replace_heading_and_section, append_to_heading, replace_block, append_to_note", req.Mode)
	}

	if req.DryRun {
		diff := writer.UnifiedDiff(req.Path, req.Path, string(raw), string(proposed), 3)
		return PatchNoteResult{Path: req.Path, Diff: diff, Patched: false, Hash: hashDoConteudo(proposed)}, nil
	}

	if err := writer.WriteAtomic(ctx, absPath, proposed); err != nil {
		return PatchNoteResult{}, Errorf(CodeInternal, "escrevendo nota %q: %v", req.Path, err)
	}

	return PatchNoteResult{Path: req.Path, Diff: "", Patched: true, Hash: hashDoConteudo(proposed)}, nil
}

// MoveNoteRequest carrega os parametros para note_move.
type MoveNoteRequest struct {
	From          string `json:"from"`
	To            string `json:"to"`
	UpdateLinks   bool   `json:"update_links"`
	CreateFolders bool   `json:"create_folders"`
	DryRun        bool   `json:"dry_run"`
}

// BrokenAnchor descreve um link que aponta para um heading ou bloco inexistente.
type BrokenAnchor struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Anchor string `json:"anchor"`
}

// MoveNoteResult e o retorno de note_move.
type MoveNoteResult struct {
	From          string            `json:"from"`
	To            string            `json:"to"`
	Rewritten     []string          `json:"rewritten"`
	LinksUpdated  int               `json:"links_updated"`
	BrokenAnchors []BrokenAnchor    `json:"broken_anchors,omitempty"`
	DryRun        bool              `json:"dry_run,omitempty"`
	Diffs         map[string]string `json:"diffs,omitempty"`
}

// MoveNote renomeia ou move uma nota e reescreve os links que apontam para ela.
func (s *Service) MoveNote(ctx context.Context, req MoveNoteRequest) (MoveNoteResult, error) {
	if err := s.checkWritable(); err != nil {
		return MoveNoteResult{}, err
	}

	canonicalFrom, err := s.index.ResolvePath(req.From)
	if err != nil {
		return MoveNoteResult{}, ErroDeResolucao(req.From, err)
	}

	_, _, err = vault.Resolve(s.vault.Root(), req.To)
	if err != nil {
		return MoveNoteResult{}, mapVaultErr(err)
	}

	toInput := req.To
	if !strings.HasSuffix(toInput, ".md") {
		toInput += ".md"
	}
	absTo, canonicalTo, err := vault.Resolve(s.vault.Root(), toInput)
	if err != nil {
		return MoveNoteResult{}, mapVaultErr(err)
	}

	if _, ok := s.index.Get(canonicalTo); ok {
		return MoveNoteResult{}, Errorf(CodeNoteExists, "destino %q ja existe no cofre", req.To)
	}
	if _, err := os.Stat(absTo); err == nil {
		return MoveNoteResult{}, Errorf(CodeNoteExists, "destino %q ja existe no cofre", req.To)
	}

	dirTo := filepath.Dir(absTo)
	if _, err := os.Stat(dirTo); os.IsNotExist(err) {
		if !req.CreateFolders {
			return MoveNoteResult{}, Errorf(CodeFolderNotFound, "diretorio %q nao existe", filepath.Dir(req.To))
		}
	}

	affectedNotes := make(map[vault.CanonicalPath][]writer.LinkReplacement)
	var totalLinks int
	var brokenAnchors []BrokenAnchor

	if req.UpdateLinks {
		backlinks := s.index.Backlinks(canonicalFrom)
		toBaseName := strings.TrimSuffix(filepath.Base(string(canonicalTo)), ".md")
		seenFrom := make(map[vault.CanonicalPath]bool)

		for _, bl := range backlinks {
			if seenFrom[bl.From] {
				continue
			}
			seenFrom[bl.From] = true

			refNote, ok := s.index.Get(bl.From)
			if !ok {
				continue
			}

			var replacements []writer.LinkReplacement
			for _, rl := range refNote.Links {
				if rl.Resolved == canonicalFrom {
					if rl.Anchor != "" && rl.State == index.LinkAnchorMissing {
						brokenAnchors = append(brokenAnchors, BrokenAnchor{
							From:   string(bl.From),
							To:     string(canonicalTo),
							Anchor: rl.Anchor,
						})
					}

					var newTarget string
					if rl.Kind == parser.LinkWiki || rl.Kind == parser.LinkEmbed {
						if strings.Contains(rl.Target, "/") {
							newTarget = strings.TrimSuffix(string(canonicalTo), ".md")
						} else {
							newTarget = toBaseName
						}
					} else {
						newTarget = string(canonicalTo)
					}
					replacements = append(replacements, writer.LinkReplacement{
						Link:      rl.Link,
						NewTarget: newTarget,
					})
				}
			}

			if len(replacements) > 0 {
				affectedNotes[bl.From] = replacements
				totalLinks += len(replacements)
			}
		}
	}

	if req.DryRun {
		diffs := make(map[string]string)
		absFrom := s.vault.Abs(canonicalFrom)
		// O erro de leitura SOBE. Ate 2026-08-26 era `fromRaw, _ :=`, e o
		// dry-run seguia produzindo um diff de "" contra "" — vazio, mas
		// apresentado como resultado legitimo. Quem le um dry-run vazio conclui
		// que a operacao nao muda nada, que e o oposto do que aconteceria.
		fromRaw, err := os.ReadFile(absFrom)
		if err != nil {
			return MoveNoteResult{}, Errorf(CodeInternal,
				"lendo nota de origem %q para o dry-run: %v", canonicalFrom, err)
		}
		diffs[string(canonicalFrom)] = writer.UnifiedDiff(string(canonicalFrom), string(canonicalTo), string(fromRaw), string(fromRaw), 3)

		for refPath, replacements := range affectedNotes {
			absRef := s.vault.Abs(refPath)
			raw, err := os.ReadFile(absRef)
			if err != nil {
				continue
			}
			rewritten, err := writer.RewriteLinks(raw, replacements)
			if err != nil {
				continue
			}
			diffs[string(refPath)] = writer.UnifiedDiff(string(refPath), string(refPath), string(raw), string(rewritten), 3)
		}

		return MoveNoteResult{
			From:          string(canonicalFrom),
			To:            string(canonicalTo),
			Rewritten:     nil,
			LinksUpdated:  totalLinks,
			BrokenAnchors: brokenAnchors,
			DryRun:        true,
			Diffs:         diffs,
		}, nil
	}

	// Sort keys of affectedNotes alphabetically for deterministic processing order
	affectedKeys := make([]vault.CanonicalPath, 0, len(affectedNotes))
	for k := range affectedNotes {
		affectedKeys = append(affectedKeys, k)
	}
	sort.Slice(affectedKeys, func(i, j int) bool {
		return affectedKeys[i] < affectedKeys[j]
	})

	// O diretorio de destino precisa existir ANTES do move do corpo.
	//
	// Ele era criado depois, o que funcionava enquanto o corpo se movia por
	// ultimo. Com a nova ordem, criar depois faz WriteAtomic falhar em
	// "criando temporario em <dir>: The system cannot find the path
	// specified" — pego pelos testes de move existentes, nao por leitura.
	if _, err := os.Stat(dirTo); os.IsNotExist(err) {
		if err := os.MkdirAll(dirTo, 0755); err != nil {
			return MoveNoteResult{
				From:          string(canonicalFrom),
				To:            string(canonicalTo),
				BrokenAnchors: brokenAnchors,
			}, Errorf(CodeInternal, "criando diretorio %q: %v", dirTo, err)
		}
	}

	// O CORPO se move ANTES de qualquer citante ser reescrito.
	//
	// A ordem era a inversa, e custava duas coisas ao mesmo tempo. Medido em
	// teste: com a origem travada para leitura, o citante ja tinha sido gravado
	// como `ver [[destino]] aqui.` e a nota nunca se movia — links persistidos
	// em disco apontando para um destino inexistente, sem compensacao.
	//
	// A inversao troca quem fica inconsistente quando a segunda etapa falha:
	// agora sobra link apontando para o caminho antigo, que e VISIVEL e
	// recuperavel, em vez de nota duplicada, que e silenciosa. Nao e nada de
	// graca — e menos grave, e esta escrito.
	if err := s.moverCorpo(ctx, canonicalFrom, canonicalTo, absTo); err != nil {
		return MoveNoteResult{
			From:          string(canonicalFrom),
			To:            string(canonicalTo),
			BrokenAnchors: brokenAnchors,
		}, err
	}

	var rewrittenList []string
	var linksUpdatedCount int

	for _, refPath := range affectedKeys {
		replacements := affectedNotes[refPath]
		unlock := s.locker.Lock(refPath)
		absRef := s.vault.Abs(refPath)
		raw, err := os.ReadFile(absRef)
		if err != nil {
			unlock()
			return MoveNoteResult{
				From:          string(canonicalFrom),
				To:            string(canonicalTo),
				Rewritten:     rewrittenList,
				LinksUpdated:  linksUpdatedCount,
				BrokenAnchors: brokenAnchors,
			}, Errorf(CodeInternal, "lendo nota %q: %v", refPath, err)
		}

		rewritten, err := writer.RewriteLinks(raw, replacements)
		if err != nil {
			unlock()
			return MoveNoteResult{
				From:          string(canonicalFrom),
				To:            string(canonicalTo),
				Rewritten:     rewrittenList,
				LinksUpdated:  linksUpdatedCount,
				BrokenAnchors: brokenAnchors,
			}, Errorf(CodeInternal, "reescrevendo links em %q: %v", refPath, err)
		}

		if err := writer.WriteAtomic(ctx, absRef, rewritten); err != nil {
			unlock()
			return MoveNoteResult{
				From:          string(canonicalFrom),
				To:            string(canonicalTo),
				Rewritten:     rewrittenList,
				LinksUpdated:  linksUpdatedCount,
				BrokenAnchors: brokenAnchors,
			}, Errorf(CodeInternal, "escrevendo nota %q: %v", refPath, err)
		}

		unlock()
		rewrittenList = append(rewrittenList, string(refPath))
		linksUpdatedCount += len(replacements)
	}

	return MoveNoteResult{
		From:          string(canonicalFrom),
		To:            string(canonicalTo),
		Rewritten:     rewrittenList,
		LinksUpdated:  linksUpdatedCount,
		BrokenAnchors: brokenAnchors,
	}, nil
}

// DeleteNoteRequest carrega os parametros para note_delete.
type DeleteNoteRequest struct {
	Path              string `json:"path"`
	ToTrash           bool   `json:"to_trash"`
	ReportBrokenLinks bool   `json:"report_broken_links"`
	DryRun            bool   `json:"dry_run"`
}

// DeleteNoteResult e o retorno de note_delete.
type DeleteNoteResult struct {
	Path          string         `json:"path"`
	Deleted       bool           `json:"deleted"`
	MovedToTrash  bool           `json:"moved_to_trash"`
	TrashPath     string         `json:"trash_path,omitempty"`
	BrokenLinks   []string       `json:"broken_links,omitempty"`
	BrokenAnchors []BrokenAnchor `json:"broken_anchors,omitempty"`
	DryRun        bool           `json:"dry_run,omitempty"`
}

// DeleteNote exclui uma nota do cofre (por padrao movendo para a lixeira .trash/).
func (s *Service) DeleteNote(ctx context.Context, req DeleteNoteRequest) (DeleteNoteResult, error) {
	if err := s.checkWritable(); err != nil {
		return DeleteNoteResult{}, err
	}

	canonical, err := s.index.ResolvePath(req.Path)
	if err != nil {
		return DeleteNoteResult{}, ErroDeResolucao(req.Path, err)
	}

	// 1. Calcula o relatorio de links quebrados e ancoras quebradas ANTES de excluir
	var brokenLinks []string
	var brokenAnchors []BrokenAnchor
	if req.ReportBrokenLinks {
		backlinks := s.index.Backlinks(canonical)
		seen := make(map[string]bool)
		for _, bl := range backlinks {
			pathStr := string(bl.From)
			if !seen[pathStr] {
				seen[pathStr] = true
				brokenLinks = append(brokenLinks, pathStr)
			}

			refNote, ok := s.index.Get(bl.From)
			if ok {
				for _, rl := range refNote.Links {
					if rl.Resolved == canonical && rl.Anchor != "" {
						brokenAnchors = append(brokenAnchors, BrokenAnchor{
							From:   string(bl.From),
							To:     string(canonical),
							Anchor: rl.Anchor,
						})
					}
				}
			}
		}
	}

	if req.DryRun {
		return DeleteNoteResult{
			Path:          string(canonical),
			Deleted:       false,
			MovedToTrash:  req.ToTrash,
			BrokenLinks:   brokenLinks,
			BrokenAnchors: brokenAnchors,
			DryRun:        true,
		}, nil
	}

	unlock := s.locker.Lock(canonical)
	defer unlock()

	absPath := s.vault.Abs(canonical)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return DeleteNoteResult{}, Errorf(CodeNoteNotFound, "nota %q nao encontrada no disco", req.Path)
	}

	if req.ToTrash {
		baseName := filepath.Base(string(canonical))
		trashRel := filepath.ToSlash(filepath.Join(".trash", baseName))
		absTrash, _, err := vault.Resolve(s.vault.Root(), trashRel)
		if err != nil {
			return DeleteNoteResult{}, mapVaultErr(err)
		}

		// Resolve colisoes de nome na lixeira adicionando timestamp
		if _, err := os.Stat(absTrash); err == nil {
			ext := filepath.Ext(baseName)
			stem := strings.TrimSuffix(baseName, ext)
			uniqueName := fmt.Sprintf("%s_%d%s", stem, time.Now().UnixNano(), ext)
			trashRel = filepath.ToSlash(filepath.Join(".trash", uniqueName))
			absTrash, _, err = vault.Resolve(s.vault.Root(), trashRel)
			if err != nil {
				return DeleteNoteResult{}, mapVaultErr(err)
			}
		}

		trashDir := filepath.Dir(absTrash)
		if err := os.MkdirAll(trashDir, 0755); err != nil {
			return DeleteNoteResult{}, Errorf(CodeInternal, "criando diretorio lixeira: %v", err)
		}

		raw, err := os.ReadFile(absPath)
		if err != nil {
			return DeleteNoteResult{}, Errorf(CodeInternal, "lendo nota %q: %v", req.Path, err)
		}

		if err := writer.WriteAtomic(ctx, absTrash, raw); err != nil {
			return DeleteNoteResult{}, Errorf(CodeInternal, "movendo nota para lixeira: %v", err)
		}

		// O erro do remove e CONFERIDO. Ate 2026-08-26 era `_ = os.Remove`, e
		// uma nota travada pelo Obsidian produzia Deleted=true com o arquivo
		// existindo em DOIS lugares: o caminho original e a lixeira. Medido em
		// teste. E a mesma familia do A1, no outro caminho de escrita.
		//
		// A mensagem diz que a copia na lixeira existe, porque sem isso quem
		// recebe o erro nao sabe se precisa limpar alguma coisa.
		if err := os.Remove(absPath); err != nil {
			return DeleteNoteResult{}, Errorf(CodeFileLocked,
				"a nota foi copiada para a lixeira em %q, mas %q nao pode ser removida (%v); "+
					"a nota existe nos dois caminhos ate a origem ser liberada",
				trashRel, canonical, err)
		}

		return DeleteNoteResult{
			Path:          string(canonical),
			Deleted:       true,
			MovedToTrash:  true,
			TrashPath:     trashRel,
			BrokenLinks:   brokenLinks,
			BrokenAnchors: brokenAnchors,
		}, nil
	}

	// Exclusao definitiva (to_trash == false)
	if err := os.Remove(absPath); err != nil {
		return DeleteNoteResult{}, Errorf(CodeInternal, "excluindo nota %q: %v", req.Path, err)
	}

	return DeleteNoteResult{
		Path:          string(canonical),
		Deleted:       true,
		MovedToTrash:  false,
		BrokenLinks:   brokenLinks,
		BrokenAnchors: brokenAnchors,
	}, nil
}

// moverCorpo move o arquivo da nota, conferindo TODOS os erros.
//
// `_ = os.Remove(absFrom)` descartava o erro do remove. No Windows, arquivo com
// handle aberto por outro processo recusa remocao — e o Obsidian segurando a
// nota aberta e rotina, nao caso de borda. O resultado era sucesso reportado
// com a nota existindo nos DOIS caminhos, em silencio.
//
// os.Rename primeiro: no mesmo volume ele e atomico, entao "duplicada" deixa de
// ser um estado possivel. O fallback copia-e-remove existe para volume
// diferente, e ali o remove e conferido.
// Recebe ctx porque a copia de fallback passa por WriteAtomic, que espera de
// verdade — laco de rename com recuo (achado M13).
func (s *Service) moverCorpo(ctx context.Context, de, para vault.CanonicalPath, absTo string) error {
	// Travas em ordem determinada pela chave, nao pela direcao do move.
	//
	// Adquirir sempre from->to permite o deadlock AB-BA entre dois moves
	// opostos simultaneos (A->B e B->A). Ordenar por chave da a ordem global
	// que o torna impossivel, e custa uma comparacao de string.
	primeira, segunda := de, para
	if string(segunda) < string(primeira) {
		primeira, segunda = segunda, primeira
	}
	destravaPrimeira := s.locker.Lock(primeira)
	defer destravaPrimeira()
	destravaSegunda := s.locker.Lock(segunda)
	defer destravaSegunda()

	absFrom := s.vault.Abs(de)

	if err := os.Rename(absFrom, absTo); err == nil {
		return nil
	}

	// Volume diferente, ou rename recusado: copia e remove, com o erro do
	// remove CONFERIDO.
	fromRaw, err := os.ReadFile(absFrom)
	if err != nil {
		return Errorf(CodeInternal, "lendo nota de origem %q: %v", de, err)
	}
	if err := writer.WriteAtomic(ctx, absTo, fromRaw); err != nil {
		return Errorf(CodeInternal, "escrevendo destino %q: %v", absTo, err)
	}
	if err := os.Remove(absFrom); err != nil {
		return Errorf(CodeFileLocked,
			"a nota foi copiada para %q mas a origem %q nao pode ser removida (%v); "+
				"a nota existe nos dois caminhos ate a origem ser liberada", para, de, err)
	}
	return nil
}
