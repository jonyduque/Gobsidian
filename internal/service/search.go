package service

import (
	"context"
	"strings"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

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
	results := make([]SearchHit, 0, len(pagedHits))

	for _, hit := range pagedHits {
		note, ok := s.index.Get(vault.CanonicalPath(hit.Path))
		if !ok {
			continue
		}

		snip, _ := search.GenerateSnippet(ctx, s.vault, s.inverted, idxImpl, hit.Path, rawTerms, opts.SnippetChars)

		var matchedHeadings []string
		if snip.MatchedHeading != "" {
			matchedHeadings = append(matchedHeadings, snip.MatchedHeading)
		}

		results = append(results, SearchHit{
			Path:            string(note.Path),
			Title:           note.Title,
			Score:           hit.Score,
			Snippet:         snip.Text,
			MatchedHeadings: matchedHeadings,
			Modified:        note.ModTime.UTC().Format(time.RFC3339),
		})
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

	firstPostings := s.inverted.Postings(phraseTokens[0].Raw)
	var firstPositions []search.TokenPosition
	for _, p := range firstPostings {
		if p.Path == path {
			firstPositions = p.Positions
			break
		}
	}
	if len(firstPositions) == 0 {
		return false
	}

	for _, pos1 := range firstPositions {
		matchedAll := true
		currEnd := pos1.End

		for i := 1; i < len(phraseTokens); i++ {
			nextPostings := s.inverted.Postings(phraseTokens[i].Raw)
			var nextPositions []search.TokenPosition
			for _, p := range nextPostings {
				if p.Path == path {
					nextPositions = p.Positions
					break
				}
			}

			foundNext := false
			for _, posN := range nextPositions {
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
