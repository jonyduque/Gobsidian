package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	Offset             *int64
	MaxBytes           int
	IncludeFrontmatter bool
}

// ReadResult e o retorno de note_read. Section vem preenchido so quando a
// leitura foi recortada por heading, e Truncated diz que MaxBytes cortou o
// resultado — sem ele o cliente nao distingue nota curta de nota cortada.
type ReadResult struct {
	Content    string          `json:"content"`
	Hash       string          `json:"hash"`
	Section    *parser.Heading `json:"section,omitempty"`
	Truncated  bool            `json:"truncated,omitempty"`
	TotalSize  int64           `json:"total_size"`
	NextOffset *int64          `json:"next_offset,omitempty"`
}

// ReadBatchRequest e os parametros de note_read em modo lote. Heading,
// HeadingLevel, BlockID, MaxBytes e IncludeFrontmatter valem para CADA
// caminho de Paths, do mesmo jeito que valeriam para um ReadRequest unico —
// nao ha campo por-item porque o schema de entrada e plano.
type ReadBatchRequest struct {
	Paths              []string
	Heading            string
	HeadingLevel       int
	BlockID            string
	Offset             *int64
	MaxBytes           int
	IncludeFrontmatter bool
}

// ReadNoteItem e um item do lote de note_read. Path identifica de qual
// entrada de ReadBatchRequest.Paths o item veio, na mesma posicao — e o que
// permite ao chamador casar erro com pedido sem depender de ordem estavel.
// Err carrega a falha daquele item especificamente: uma nota ausente no meio
// de dez nao pode custar as outras nove, mas tambem nao pode desaparecer da
// lista sem dizer nada.
type ReadNoteItem struct {
	Path       string          `json:"path"`
	Content    string          `json:"content,omitempty"`
	Hash       string          `json:"hash,omitempty"`
	Section    *parser.Heading `json:"section,omitempty"`
	Truncated  bool            `json:"truncated,omitempty"`
	TotalSize  int64           `json:"total_size,omitempty"`
	NextOffset *int64          `json:"next_offset,omitempty"`
	Err        error           `json:"-"`
}

// readNoteItemWire e a forma serializada de ReadNoteItem. Err e um error Go
// sem campos exportados: sem este tipo auxiliar, json.Marshal produziria
// "error":{} e o codigo/mensagem se perderiam no canal que o cliente le.
type readNoteItemWire struct {
	Path       string          `json:"path"`
	Content    string          `json:"content,omitempty"`
	Hash       string          `json:"hash,omitempty"`
	Section    *parser.Heading `json:"section,omitempty"`
	Truncated  bool            `json:"truncated,omitempty"`
	TotalSize  int64           `json:"total_size,omitempty"`
	NextOffset *int64          `json:"next_offset,omitempty"`
	Error      *itemErrorWire  `json:"error,omitempty"`
}

type itemErrorWire struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// MarshalJSON traduz Err (um error Go) para {"code","message"} — a mesma
// forma que toolErr usa no canal de texto, so que endereçavel por campo em
// vez de exigir parsear a mensagem.
func (i ReadNoteItem) MarshalJSON() ([]byte, error) {
	wire := readNoteItemWire{
		Path:       i.Path,
		Content:    i.Content,
		Hash:       i.Hash,
		Section:    i.Section,
		Truncated:  i.Truncated,
		TotalSize:  i.TotalSize,
		NextOffset: i.NextOffset,
	}
	if i.Err != nil {
		wire.Error = &itemErrorWire{Code: string(CodeOf(i.Err)), Message: i.Err.Error()}
	}
	return json.Marshal(wire)
}

// ReadBatchResult e o retorno de note_read em modo lote: um item por
// caminho pedido, na mesma ordem e no mesmo comprimento de Paths.
type ReadBatchResult struct {
	Items []ReadNoteItem `json:"items"`
}

// ReadNotes le varias notas numa chamada so. Cada caminho de req.Paths vira
// um ReadNoteItem na MESMA posicao do slice de saida — um erro num item nao
// remove os demais, e nao desloca os que vem depois dele. E o índice, não
// o comprimento da lista, que amarra pedido e resposta.
func (s *Service) ReadNotes(ctx context.Context, req ReadBatchRequest) ReadBatchResult {
	out := make([]ReadNoteItem, len(req.Paths))
	for i, p := range req.Paths {
		res, err := s.ReadNote(ctx, ReadRequest{
			Path:               p,
			Heading:            req.Heading,
			HeadingLevel:       req.HeadingLevel,
			BlockID:            req.BlockID,
			Offset:             req.Offset,
			MaxBytes:           req.MaxBytes,
			IncludeFrontmatter: req.IncludeFrontmatter,
		})
		if err != nil {
			out[i] = ReadNoteItem{Path: p, Err: err}
			continue
		}
		out[i] = ReadNoteItem{
			Path:       p,
			Content:    res.Content,
			Hash:       res.Hash,
			Section:    res.Section,
			Truncated:  res.Truncated,
			TotalSize:  res.TotalSize,
			NextOffset: res.NextOffset,
		}
	}
	return ReadBatchResult{Items: out}
}

// ReadNote le uma nota, ou um recorte dela por heading ou por bloco.
//
// Le apenas o intervalo de bytes pedido: e o offset guardado no indice que faz
// uma secao de 2 KB numa nota de 500 KB custar 2 KB.
func (s *Service) ReadNote(ctx context.Context, req ReadRequest) (ReadResult, error) {
	if req.Heading != "" && req.BlockID != "" {
		return ReadResult{}, Errorf(CodeInvalidArgument, "heading e block_id sao mutuamente exclusivos")
	}
	if req.Offset != nil && (req.Heading != "" || req.BlockID != "") {
		return ReadResult{}, Errorf(CodeInvalidArgument, "offset e mutuamente exclusivo com heading e block_id")
	}

	canonical, err := s.index.ResolvePath(req.Path)
	if err != nil {
		return ReadResult{}, ErroDeResolucao(req.Path, err)
	}

	note, ok := s.index.Get(canonical)
	if !ok {
		return ReadResult{}, Errorf(CodeNoteNotFound, "nota %q nao encontrada", req.Path)
	}

	if note.CloudOnly {
		return ReadResult{}, Errorf(CodeCloudOnlyFile, "nota %q e apenas online (CloudOnly)", req.Path)
	}

	if req.Offset != nil && (*req.Offset < 0 || *req.Offset > note.Size) {
		return ReadResult{}, Errorf(CodeInvalidArgument, "offset %d fora dos limites da nota (tamanho %d)", *req.Offset, note.Size)
	}

	start := int64(0)
	end := note.Size
	var matchedHeading *parser.Heading

	switch {
	case req.Heading != "":
		targetSlug := parser.Slug(req.Heading)
		var matches []parser.Heading

		for _, h := range note.Headings {
			if h.Slug == targetSlug {
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

	case req.Offset != nil:
		start = *req.Offset
		end = note.Size

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

	truncou := false
	if req.MaxBytes > 0 && (end-start) > int64(req.MaxBytes) {
		end = start + int64(req.MaxBytes)
		truncou = true
	}

	data, err := s.vault.ReadRange(ctx, canonical, start, end)
	if err != nil {
		return ReadResult{}, Errorf(CodeVaultUnavailable, "falha lendo faixa da nota: %v", err)
	}

	res := ReadResult{
		Content:   string(data),
		Hash:      fmt.Sprintf("%016x", note.Hash),
		Section:   matchedHeading,
		Truncated: truncou,
		TotalSize: note.Size,
	}
	if truncou {
		res.NextOffset = &end
	}

	return res, nil
}
