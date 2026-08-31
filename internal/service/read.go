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

// ReadAlvo e um item do lote de note_read: o caminho, e o que sobrepoe os
// padroes de topo SO para ele.
//
// Todo campo sobreponivel e PONTEIRO, e nao valor. Um item que traz
// max_bytes=0 esta pedindo explicitamente "sem teto", e isso e um pedido
// DIFERENTE de um item que nao trouxe max_bytes nenhum e herda o teto do topo.
// Campo nao-ponteiro nao distingue os dois — e a armadilha de ReadOnlySet e
// DebounceMSSet, um nivel abaixo.
type ReadAlvo struct {
	Path               string
	Heading            *string
	HeadingLevel       *int
	BlockID            *string
	Offset             *int64
	MaxBytes           *int
	IncludeFrontmatter *bool
}

// ReadBatchRequest e os parametros de note_read em modo lote. Heading,
// HeadingLevel, BlockID, Offset, MaxBytes e IncludeFrontmatter sao o PADRAO
// aplicado a cada alvo; cada ReadAlvo sobrepoe campo a campo o que trouxer.
//
// Ate 2026-08-31 os campos de topo eram a unica forma de entrada, e o comentario
// daqui registrava a escolha: "nao ha campo por-item porque o schema de entrada
// e plano". O custo caiu inteiro sobre o caso mais comum — seis capitulos de uma
// obra, com seis headings diferentes, exigiam seis chamadas.
type ReadBatchRequest struct {
	Alvos              []ReadAlvo
	Heading            string
	HeadingLevel       int
	BlockID            string
	Offset             *int64
	MaxBytes           int
	IncludeFrontmatter bool
}

// pedidoDoAlvo funde os padroes de topo com o que o alvo sobrepoe.
//
// E a UNICA conta da heranca. Fundir no lugar de uso, campo a campo, e como o
// erro previsivel desta superficie entra: sobrepor o REGISTRO INTEIRO em vez de
// campo a campo, e ai o max_bytes do topo some no item que so pediu heading.
func pedidoDoAlvo(padrao ReadBatchRequest, alvo ReadAlvo) ReadRequest {
	req := ReadRequest{
		Path:               alvo.Path,
		Heading:            padrao.Heading,
		HeadingLevel:       padrao.HeadingLevel,
		BlockID:            padrao.BlockID,
		Offset:             padrao.Offset,
		MaxBytes:           padrao.MaxBytes,
		IncludeFrontmatter: padrao.IncludeFrontmatter,
	}
	if alvo.Heading != nil {
		req.Heading = *alvo.Heading
		// Heading e offset sao mutuamente exclusivos (D-R-3). Um alvo que pede
		// heading sem pedir offset nao pode herdar o offset do topo e virar
		// INVALID_ARGUMENT por um campo que ele nao mandou.
		if alvo.Offset == nil {
			req.Offset = nil
		}
	}
	if alvo.HeadingLevel != nil {
		req.HeadingLevel = *alvo.HeadingLevel
	}
	if alvo.BlockID != nil {
		req.BlockID = *alvo.BlockID
		if alvo.Offset == nil {
			req.Offset = nil
		}
	}
	if alvo.Offset != nil {
		req.Offset = alvo.Offset
		// Simetrico: offset explicito no item apaga o heading/block herdados.
		if alvo.Heading == nil {
			req.Heading = ""
		}
		if alvo.BlockID == nil {
			req.BlockID = ""
		}
	}
	if alvo.MaxBytes != nil {
		req.MaxBytes = *alvo.MaxBytes
	}
	if alvo.IncludeFrontmatter != nil {
		req.IncludeFrontmatter = *alvo.IncludeFrontmatter
	}
	return req
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
	out := make([]ReadNoteItem, len(req.Alvos))
	for i, alvo := range req.Alvos {
		p := alvo.Path
		res, err := s.ReadNote(ctx, pedidoDoAlvo(req, alvo))
		if err != nil {
			// O indice do item entra na mensagem: "paths invalido" nao ajuda
			// quem mandou seis capitulos numa chamada so (D-R-3).
			out[i] = ReadNoteItem{Path: p, Err: Errorf(CodeOf(err), "item %d de paths (%s): %s", i, p, err.Error())}
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
			// Nota SEM heading nenhum e nota COM headings que nao casam sao
			// perguntas diferentes, e a resposta tem de dizer qual e.
			//
			// A nota convertida de livro cai sempre no primeiro caso: ela marca
			// titulo com paragrafo em negrito, e a lista de "disponiveis" sai
			// vazia sem explicar por que. E o incidente de campo de 2026-08-15.
			if len(alternatives) == 0 {
				return ReadResult{}, Errorf(CodeHeadingNotFound,
					"nota %q nao tem heading Markdown nenhum. Se ela veio de PDF, DOCX ou EPUB, os titulos provavelmente sao paragrafo em negrito: chame note_outline para ver os candidatos e leia por offset.",
					req.Path)
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
