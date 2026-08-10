package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

// maxSnippetWorkers limita quantos trechos são recortados do disco ao mesmo
// tempo. O recorte é I/O: com `limit: 200` são 200 leituras, e em série elas
// dominavam a latência.
//
// O valor 8 é medido, não escolhido por gosto. Varredura na maquina de referencia (12 núcleos,
// Windows 11), p95 de `limit: 200`:
//
//	workers   500 notas   500 notas,        5.000 notas
//	          ociosa      4 cópias          ociosa
//	      1   69,2 ms      79,8 – 90,8 ms   561,8 ms
//	      4   35,9 ms      82,7 – 105,4 ms  226,8 ms
//	      8   30,7 ms      85,0 – 136,3 ms  177,2 ms
//	     16   35,2 ms     100,1 – 138,1 ms  217,7 ms
//
// Três coisas que a varredura mostra e que a intuição erra:
//
//  1. Mais trabalhadores não é monotonicamente melhor. 16 é PIOR que 8 nas duas
//     escalas — 12 núcleos não absorvem 16 leitores.
//  2. O ganho ocioso a 500 notas satura cedo: 16 não é mais rápido que 4.
//  3. Sob quatro cópias do binário de teste, QUALQUER pool fica pior que o
//     sequencial (80–91 ms). Mas quatro cópias multiplicam o pool por quatro, e
//     um servidor só nunca faz isso: depois desta otimização o harness de quatro
//     cópias deixou de ser proxy de "máquina ocupada" e virou proxy de "quatro
//     vezes a nossa própria concorrência". Quem escolhe o valor é a coluna de
//     processo único.
//
// A primeira versão desta otimização usava 16 e mediu só o caso ocioso a 500
// notas — o único regime em que a escolha não faz diferença.
const maxSnippetWorkers = 8

// SearchOptions representa os parâmetros de consulta e filtragem da tool vault_search.
type SearchOptions struct {
	Query          string
	Folder         string
	Tags           []string
	Frontmatter    map[string]any
	ModifiedAfter  *time.Time
	ModifiedBefore *time.Time
	SnippetChars   int
	Limit          int
	Offset         int
}

// SearchHit representa um resultado individual de busca.
type SearchHit struct {
	Path            string   `json:"path"`
	Title           string   `json:"title"`
	Score           float64  `json:"score"`
	Snippet         string   `json:"snippet"`
	MatchedHeadings []string `json:"matched_headings"`
	Modified        string   `json:"modified"`
}

// SearchResult representa o retorno consolidado da tool vault_search.
type SearchResult struct {
	Results   []SearchHit `json:"results"`
	Total     int         `json:"total"`
	Truncated bool        `json:"truncated"`
}

// Search executa a busca full-text com ranking BM25 e filtros de metadados.
func (s *Service) Search(ctx context.Context, opts SearchOptions) (SearchResult, error) {
	// garanteIndiceDeBusca dispara o carregamento do índice de busca na
	// primeira chamada (modo padrão, --eager-search desligado) e bloqueia até
	// ele terminar — é por isso que a carga entra na latência desta primeira
	// busca, e não da partida do servidor. Chamadas concorrentes esperam a
	// mesma carga; chamadas seguintes, com o índice já pronto, não pagam
	// nada aqui.
	if err := s.garanteIndiceDeBusca(ctx); err != nil {
		return SearchResult{}, Wrap(CodeIndexBuilding, err,
			"o indice de busca nao pode ser carregado; tente de novo em alguns segundos")
	}

	// O índice invertido pode estar sendo construído em segundo plano: o
	// servidor anuncia as tools assim que o índice de METADADOS fica pronto, o
	// que leva ~1 s, e não espera a tokenização do cofre, que num cofre de
	// 109 MB leva minutos. Buscar com ele pela metade acharia menos notas do
	// que existem, e essa resposta parcial chegaria como uma lista curta e
	// legítima.
	if s.inverted != nil && s.inverted.Building() {
		return SearchResult{}, Errorf(CodeIndexBuilding,
			"o indice de busca ainda esta sendo construido; tente de novo em alguns segundos")
	}

	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.Limit > 200 {
		opts.Limit = 200
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	if opts.SnippetChars <= 0 {
		opts.SnippetChars = search.DefaultSnippetChars
	}
	if opts.SnippetChars > search.MaxSnippetChars {
		opts.SnippetChars = search.MaxSnippetChars
	}

	trimmedQuery := strings.TrimSpace(opts.Query)

	// RF-25: Se a consulta for vazia mas houver filtros, redireciona para metadados
	if trimmedQuery == "" {
		return s.searchMetadataOnly(opts)
	}

	// RF-24: Aspas duplas delimitam frase exata
	isPhrase := strings.HasPrefix(trimmedQuery, `"`) && strings.HasSuffix(trimmedQuery, `"`) && len(trimmedQuery) >= 2
	var queryTokens []search.Token
	var rawTerms []string

	if isPhrase {
		phraseText := trimmedQuery[1 : len(trimmedQuery)-1]
		toks := search.Analyze(phraseText)
		for _, tok := range toks {
			queryTokens = append(queryTokens, search.Token{Raw: tok.Raw, Start: tok.Start, End: tok.End})
			rawTerms = append(rawTerms, tok.Raw)
		}
	} else {
		queryTokens = search.Analyze(trimmedQuery)
		for _, tok := range queryTokens {
			rawTerms = append(rawTerms, tok.Raw)
		}
	}

	if len(queryTokens) == 0 {
		return SearchResult{Results: []SearchHit{}, Total: 0, Truncated: false}, nil
	}

	var idxImpl *index.Index
	if realIdx, ok := s.index.(*index.Index); ok {
		idxImpl = realIdx
	}

	// Executa pontuação BM25 sobre o índice invertido
	rawHits := search.CalculateBM25(queryTokens, s.inverted, idxImpl)

	// Filtra por pasta, tags, frontmatter e data de modificação
	var filteredHits []search.Result
	for _, hit := range rawHits {
		note, ok := s.index.Get(vault.CanonicalPath(hit.Path))
		if !ok {
			continue
		}

		if !s.matchesSearchFilters(note, opts) {
			continue
		}

		if isPhrase && !s.matchPhraseInNote(hit.Path, queryTokens) {
			continue
		}

		filteredHits = append(filteredHits, hit)
	}

	total := len(filteredHits)
	if opts.Offset >= total {
		return SearchResult{Results: []SearchHit{}, Total: total, Truncated: false}, nil
	}

	end := opts.Offset + opts.Limit
	truncated := false
	if end < total {
		truncated = true
	} else {
		end = total
	}

	pagedHits := filteredHits[opts.Offset:end]

	// Um slot por hit, preenchido pelo índice: a ordem do resultado é a ordem
	// do ranking, e não a ordem em que as goroutines terminaram. `ok` diz se o
	// slot foi preenchido; usar Path == "" como sentinela confundiria "nota
	// sumiu do índice" com "goroutine não rodou".
	slots := make([]struct {
		hit SearchHit
		ok  bool
	}, len(pagedHits))

	// montaSlot é o ÚNICO lugar que constrói um SearchHit. O caminho sequencial
	// e o concorrente chamam a mesma função: dois corpos iguais divergem, e a
	// divergência aparece no caminho menos usado.
	montaSlot := func(i int, h search.Result) {
		// A nota pode ter saído do índice entre o filtro e aqui — o watcher roda
		// em paralelo. Slot não preenchido é descartado no final.
		note, ok := s.index.Get(vault.CanonicalPath(h.Path))
		if !ok {
			return
		}
		snip, _ := search.GenerateSnippet(ctx, s.vault, s.inverted, idxImpl, h.Path, rawTerms, opts.SnippetChars)
		var matchedHeadings []string
		if snip.MatchedHeading != "" {
			matchedHeadings = append(matchedHeadings, snip.MatchedHeading)
		}
		slots[i].hit = SearchHit{
			Path:            string(note.Path),
			Title:           note.Title,
			Score:           h.Score,
			Snippet:         snip.Text,
			MatchedHeadings: matchedHeadings,
			Modified:        note.ModTime.UTC().Format(time.RFC3339),
		}
		slots[i].ok = true
	}

	if workers := min(len(pagedHits), maxSnippetWorkers); workers <= 1 {
		for i, hit := range pagedHits {
			montaSlot(i, hit)
		}
	} else {
		sem := make(chan struct{}, workers)
		var wg sync.WaitGroup
		for i, hit := range pagedHits {
			wg.Add(1)
			go func(i int, h search.Result) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				// Cancelamento não pula a escrita em silêncio: quem decide o que
				// fazer com a página incompleta é o retorno de erro lá embaixo.
				// Aqui só evitamos pagar I/O por uma resposta que ninguém quer.
				if ctx.Err() != nil {
					return
				}
				montaSlot(i, h)
			}(i, hit)
		}
		wg.Wait()
	}

	// Página parcial NÃO pode sair como sucesso. Antes desta verificação, um ctx
	// cancelado devolvia entre 24 e 99 dos 200 resultados com err == nil e
	// Total intacto — o cliente não tinha como distinguir isso de uma busca que
	// achou pouco, e a contagem variava entre chamadas idênticas.
	if err := ctx.Err(); err != nil {
		return SearchResult{}, err
	}

	results := make([]SearchHit, 0, len(slots))
	for _, sl := range slots {
		if sl.ok {
			results = append(results, sl.hit)
		}
	}

	return SearchResult{
		Results:   results,
		Total:     total,
		Truncated: truncated,
	}, nil
}

func (s *Service) searchMetadataOnly(opts SearchOptions) (SearchResult, error) {
	q := index.Query{
		Folder:         opts.Folder,
		Tags:           opts.Tags,
		Frontmatter:    opts.Frontmatter,
		ModifiedAfter:  opts.ModifiedAfter,
		ModifiedBefore: opts.ModifiedBefore,
		Sort:           "path",
		Limit:          0,
	}

	notes, total := s.index.List(q)
	if opts.Offset >= total {
		return SearchResult{Results: []SearchHit{}, Total: total, Truncated: false}, nil
	}

	end := opts.Offset + opts.Limit
	truncated := false
	if end < total {
		truncated = true
	} else {
		end = total
	}

	paged := notes[opts.Offset:end]
	results := make([]SearchHit, 0, len(paged))

	for _, n := range paged {
		results = append(results, SearchHit{
			Path:            string(n.Path),
			Title:           n.Title,
			Score:           0.0,
			Snippet:         "",
			MatchedHeadings: nil,
			Modified:        n.ModTime.UTC().Format(time.RFC3339),
		})
	}

	return SearchResult{
		Results:   results,
		Total:     total,
		Truncated: truncated,
	}, nil
}

func (s *Service) matchesSearchFilters(note *index.Note, opts SearchOptions) bool {
	if opts.Folder != "" {
		canonFolder := string(vault.CanonicalPath(opts.Folder))
		if canonFolder != "" && !strings.HasPrefix(string(note.Path), canonFolder+"/") && string(note.Path) != canonFolder {
			return false
		}
	}

	if len(opts.Tags) > 0 {
		noteTags := make(map[string]bool)
		for _, t := range note.Tags {
			noteTags[strings.ToLower(t)] = true
		}
		for _, reqTag := range opts.Tags {
			cleanTag := strings.TrimPrefix(strings.ToLower(reqTag), "#")
			if !noteTags[cleanTag] {
				return false
			}
		}
	}

	if opts.ModifiedAfter != nil && note.ModTime.Before(*opts.ModifiedAfter) {
		return false
	}

	if opts.ModifiedBefore != nil && note.ModTime.After(*opts.ModifiedBefore) {
		return false
	}

	if len(opts.Frontmatter) > 0 {
		q := index.Query{
			Frontmatter: opts.Frontmatter,
		}
		notes, _ := s.index.List(q)
		matched := false
		for _, n := range notes {
			if n.Path == note.Path {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

func (s *Service) matchPhraseInNote(path string, phraseTokens []search.Token) bool {
	if s.inverted == nil || len(phraseTokens) <= 1 {
		return true
	}

	tokenPositions := make([][]search.TokenPosition, len(phraseTokens))
	for i, tok := range phraseTokens {
		positions := s.inverted.Positions(tok.Raw, path)
		if len(positions) == 0 {
			return false
		}
		tokenPositions[i] = positions
	}

	for _, pos1 := range tokenPositions[0] {
		matchedAll := true
		currEnd := pos1.End

		for i := 1; i < len(phraseTokens); i++ {
			foundNext := false
			for _, posN := range tokenPositions[i] {
				if posN.Start >= currEnd && posN.Start <= currEnd+1 {
					currEnd = posN.End
					foundNext = true
					break
				}
			}

			if !foundNext {
				matchedAll = false
				break
			}
		}

		if matchedAll {
			return true
		}
	}

	return false
}
