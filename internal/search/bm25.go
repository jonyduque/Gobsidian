package search

import (
	"math"
	"sort"
	"strings"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Constantes da fórmula BM25 e pesos de campo (D9, RF-20).
// ARCHITECTURE.md §6.2 exige que a forma crua (WeightRaw) pontue acima da forma reduzida (WeightReduced),
// garantindo que a busca por termo exato receba peso superior ao termo derivado morfológicamente.
const (
	ParamK1        = 1.2
	ParamB         = 0.75
	WeightTitle    = 3.0
	WeightHeadings = 2.0
	WeightBody     = 1.0
	WeightRaw      = 1.5
	WeightReduced  = 1.0
)

// Result representa uma nota pontuada no resultado da busca.
type Result struct {
	Path  string
	Score float64
}

// CalculateBM25 calcula os scores de relevância BM25 para um conjunto de tokens de consulta.
func CalculateBM25(queryTokens []Token, ix *Inverted, idx *index.Index) []Result {
	if ix == nil || ix.DocCount() == 0 || len(queryTokens) == 0 {
		return nil
	}

	totalDocs := ix.DocCount()
	N := float64(totalDocs)

	// Calcula o comprimento médio dos documentos (avgdl)
	var sumDocLen float64
	docTermFreqs := make(map[string]map[int]float64)

	type termMatch struct {
		term      string
		queryIdx  int
		matchMult float64
	}

	var matches []termMatch
	for i, qTok := range queryTokens {
		if qTok.Raw != "" {
			matches = append(matches, termMatch{term: qTok.Raw, queryIdx: i, matchMult: WeightRaw})
		}
		if qTok.Reduced != "" && qTok.Reduced != qTok.Raw {
			matches = append(matches, termMatch{term: qTok.Reduced, queryIdx: i, matchMult: WeightReduced})
		}
	}

	// Pre-fetch all postings to avoid duplicate allocations (profile: 371.40 MB in Postings).
	// Fetch once for each unique term and reuse throughout scoring and IDF calculation.
	termPostings := make(map[string][]Posting)
	for _, m := range matches {
		if _, exists := termPostings[m.term]; !exists {
			termPostings[m.term] = ix.Postings(m.term)
		}
	}

	for _, m := range matches {
		postings := termPostings[m.term]
		if len(postings) == 0 {
			continue
		}

		for _, p := range postings {
			if _, exists := docTermFreqs[p.Path]; !exists {
				docTermFreqs[p.Path] = make(map[int]float64)
			}

			for _, pos := range p.Positions {
				fw := getFieldWeight(p.Path, pos, m.term, idx)
				docTermFreqs[p.Path][m.queryIdx] += fw * m.matchMult
			}
		}
	}

	if len(docTermFreqs) == 0 {
		return nil
	}

	if idx != nil {
		for _, p := range idx.Paths() {
			if dl := ix.DocLength(string(p)); dl > 0 {
				sumDocLen += float64(dl)
			}
		}
	} else {
		for path := range docTermFreqs {
			sumDocLen += float64(ix.DocLength(path))
		}
	}

	avgdl := sumDocLen / N
	if avgdl == 0 {
		avgdl = 1.0
	}

	termIDFs := make(map[int]float64)
	for i, qTok := range queryTokens {
		docsWithTerm := make(map[string]bool)
		for _, p := range termPostings[qTok.Raw] {
			docsWithTerm[p.Path] = true
		}
		if qTok.Reduced != "" && qTok.Reduced != qTok.Raw {
			for _, p := range termPostings[qTok.Reduced] {
				docsWithTerm[p.Path] = true
			}
		}

		nq := float64(len(docsWithTerm))
		if nq > 0 {
			idf := math.Log(1.0 + (N-nq+0.5)/(nq+0.5))
			termIDFs[i] = idf
		}
	}

	docScores := make(map[string]float64)
	for path, termFreqs := range docTermFreqs {
		docLen := float64(ix.DocLength(path))
		var score float64

		for qIdx, tf := range termFreqs {
			idf := termIDFs[qIdx]
			if idf <= 0 || tf <= 0 {
				continue
			}

			denom := tf + ParamK1*(1.0-ParamB+ParamB*(docLen/avgdl))
			if denom > 0 {
				tfScore := (tf * (ParamK1 + 1.0)) / denom
				score += idf * tfScore
			}
		}

		if !math.IsNaN(score) && score > 0 {
			docScores[path] = score
		}
	}

	results := make([]Result, 0, len(docScores))
	for path, score := range docScores {
		results = append(results, Result{
			Path:  path,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Path < results[j].Path
	})

	return results
}

func getFieldWeight(path string, pos TokenPosition, term string, idx *index.Index) float64 {
	if idx == nil {
		return WeightBody
	}
	n, ok := idx.Get(vault.CanonicalPath(path))
	if !ok {
		return WeightBody
	}

	// 1. Título: se o título da nota contiver o termo normalizado
	if n.Title != "" {
		if strings.Contains(n.TitleNorm, term) {
			return WeightTitle
		}
	}

	// 2. Heading: se a posição estiver na linha exata do heading (# + nivel + espaço + texto)
	for _, h := range n.Headings {
		level := h.Level
		if level < 1 {
			level = 1
		}
		hLineEnd := h.Start + int64(level) + 1 + int64(len(h.Text))
		if pos.Start >= h.Start && pos.Start < hLineEnd {
			return WeightHeadings
		}
	}

	// 3. Corpo
	return WeightBody
}
