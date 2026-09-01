package service

import (
	"context"

	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
)

// CandidatosPadrao e CandidatosTeto sao os limites publicados de note_outline.
//
// Uma nota de 255 KB convertida de livro produz lista longa, e uma lista longa
// devolvida inteira e o mesmo problema que a tool existe para resolver. O teto e
// DECLARADO no schema e acompanhado de Truncated no retorno: teto silencioso e
// achado proprio nesta base — o cliente pagina o que nao sabe que foi cortado.
const (
	CandidatosPadrao = 200
	CandidatosTeto   = 1000
)

// OutlineRequest e os parametros de note_outline.
type OutlineRequest struct {
	Path string
	// MaxCandidates zero significa "use o padrao", nao "nenhum": pedir zero
	// candidatos nao e um pedido que alguem faca, e tratar zero como ausencia
	// aqui e o que permite o campo ser omitido.
	MaxCandidates int
}

// OutlineResult e o mapa de uma nota: o que E estrutura, o que PARECE
// estrutura, e a diferenca dita em campos separados.
type OutlineResult struct {
	Path      string `json:"path"`
	TotalSize int64  `json:"total_size"`
	// Headings vem do indice, sem reler o disco. E estrutura Markdown de
	// verdade.
	Headings []parser.Heading `json:"headings"`
	// Candidates e calculado na chamada, sobre os bytes da nota, e NAO e
	// persistido: o parser nao muda e IndexCacheParserVersion nao sobe. Sao
	// palpites, e o nome do campo diz isso.
	Candidates []parser.Candidate `json:"candidates"`
	Truncated  bool               `json:"truncated"`
}

// Outline devolve o mapa de uma nota — headings reais e candidatos.
//
// Recebe ctx porque le o disco: os candidatos exigem os bytes da nota, ao
// contrario dos headings, que ja estao no indice.
func (s *Service) Outline(ctx context.Context, req OutlineRequest) (OutlineResult, error) {
	canonical, err := s.index.ResolvePath(req.Path)
	if err != nil {
		return OutlineResult{}, ErroDeResolucao(req.Path, err)
	}

	note, ok := s.index.Get(canonical)
	if !ok {
		return OutlineResult{}, Errorf(CodeNoteNotFound, "nota %q nao encontrada", req.Path)
	}

	// Somente-nuvem recusa ANTES de qualquer leitura: abrir dispara download
	// sincrono, e esta tool le o arquivo inteiro.
	if note.CloudOnly {
		return OutlineResult{}, Errorf(CodeCloudOnlyFile, "nota %q e apenas online (CloudOnly)", req.Path)
	}

	dados, err := s.vault.ReadAll(ctx, canonical)
	if err != nil {
		// Nota ilegivel e nota sem estrutura NAO podem dar a mesma resposta: um
		// outline vazio aqui diria "esta nota nao tem titulos" sobre um arquivo
		// que ninguem conseguiu abrir.
		return OutlineResult{}, Errorf(CodeInternal, "lendo nota %q: %v", req.Path, err)
	}

	corpo, tinhaBOM := vault.StripBOM(dados)
	var bomOffset int64
	if tinhaBOM {
		bomOffset = int64(vault.BOMLen)
	}

	// O CORPO, sem o frontmatter — a mesma entrada que ExtractHeadings recebe.
	//
	// Passar o arquivo inteiro fazia o `---` que FECHA o frontmatter virar
	// sublinhado setext, promovendo a ultima linha dele a titulo: uma nota com
	// `tags: [x]` na ultima linha do frontmatter devolvia "tags: [x]" como
	// candidato. Medido em 2026-09-01 nos cofres reais: um falso por nota com
	// frontmatter — 1.274 em 1.275 notas no cofre Revisao.
	//
	// E o defeito que esta tool existe para nao cometer: afirmar estrutura que o
	// arquivo nao tem. Publicado na v1.3.0 e v1.3.1.
	_, corpoSemFM, deslocamentoDoFM := parser.SplitFrontmatter(corpo)
	candidatos := parser.DetectCandidates(corpoSemFM, bomOffset+deslocamentoDoFM)

	teto := req.MaxCandidates
	if teto <= 0 {
		teto = CandidatosPadrao
	}
	if teto > CandidatosTeto {
		teto = CandidatosTeto
	}
	truncado := false
	if len(candidatos) > teto {
		candidatos = candidatos[:teto]
		truncado = true
	}

	// Listas VAZIAS, nunca nil: um slice nil vira `null` no JSON, e "esta nota
	// nao tem heading nenhum" passaria a ler igual a "nao sei dizer". A tool
	// inteira existe para separar essas duas respostas.
	headings := note.Headings
	if headings == nil {
		headings = []parser.Heading{}
	}
	if candidatos == nil {
		candidatos = []parser.Candidate{}
	}

	return OutlineResult{
		Path:       string(canonical),
		TotalSize:  note.Size,
		Headings:   headings,
		Candidates: candidatos,
		Truncated:  truncado,
	}, nil
}
