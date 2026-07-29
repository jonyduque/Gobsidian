package search

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/jonyd/gobsidian/internal/vault"
)

// TokenPosition representa o offset em bytes (início e fim) onde o termo ocorreu no arquivo.
type TokenPosition struct {
	Start int64
	End   int64
}

// Posting representa as ocorrências de um termo em uma nota específica.
type Posting struct {
	Path      string          // Caminho relativo da nota (ex: "b.md")
	Positions []TokenPosition // Posições (offsets em bytes) das ocorrências
	Frequency int             // Frequência do termo na nota
}

// Inverted é um índice invertido thread-safe com suporte a atualizações incrementais.
type Inverted struct {
	mu         sync.RWMutex
	terms      map[string]map[string][]TokenPosition // termo -> (path -> posList)
	docLengths map[string]int                        // path -> contagem total de tokens na nota
}

// NewInverted inicializa e devolve um novo índice invertido.
func NewInverted() *Inverted {
	return &Inverted{
		terms:      make(map[string]map[string][]TokenPosition),
		docLengths: make(map[string]int),
	}
}

// Add insere ou substitui as ocorrências dos tokens para o caminho especificado.
func (ix *Inverted) Add(path string, tokens []Token) {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	ix.removeLocked(path)

	if len(tokens) == 0 {
		return
	}

	ix.docLengths[path] = len(tokens)

	for _, tok := range tokens {
		pos := TokenPosition{Start: tok.Start, End: tok.End}

		if tok.Raw != "" {
			ix.addTermPositionLocked(tok.Raw, path, pos)
		}
		if tok.Reduced != "" && tok.Reduced != tok.Raw {
			ix.addTermPositionLocked(tok.Reduced, path, pos)
		}
	}
}

func (ix *Inverted) addTermPositionLocked(term, path string, pos TokenPosition) {
	docs, exists := ix.terms[term]
	if !exists {
		docs = make(map[string][]TokenPosition)
		ix.terms[term] = docs
	}
	docs[path] = append(docs[path], pos)
}

// Remove remove todas as ocorrências do caminho do índice.
// Se um termo ficar com zero documentos, ele é excluído do dicionário de termos.
func (ix *Inverted) Remove(path string) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.removeLocked(path)
}

func (ix *Inverted) removeLocked(path string) {
	delete(ix.docLengths, path)
	for term, docs := range ix.terms {
		delete(docs, path)
		if len(docs) == 0 {
			delete(ix.terms, term)
		}
	}
}

// Postings devolve as ocorrências de um termo (normalizado) no índice.
// Os resultados são ordenados deterministicamente por caminho de nota.
func (ix *Inverted) Postings(term string) []Posting {
	norm := Normalize(term)

	ix.mu.RLock()
	defer ix.mu.RUnlock()

	docs, ok := ix.terms[norm]
	if !ok || len(docs) == 0 {
		return nil
	}

	res := make([]Posting, 0, len(docs))
	for path, posList := range docs {
		res = append(res, Posting{
			Path:      path,
			Positions: posList,
			Frequency: len(posList),
		})
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Path < res[j].Path
	})

	return res
}

// HasTerm verifica se o termo (normalizado) existe no dicionário e possui ao menos um documento.
func (ix *Inverted) HasTerm(term string) bool {
	norm := Normalize(term)

	ix.mu.RLock()
	defer ix.mu.RUnlock()

	docs, ok := ix.terms[norm]
	return ok && len(docs) > 0
}

// TermCount devolve a quantidade de termos ativos no dicionário.
func (ix *Inverted) TermCount() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return len(ix.terms)
}

// DocCount devolve o número de notas indexadas.
func (ix *Inverted) DocCount() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return len(ix.docLengths)
}

// DocLength devolve o número total de tokens de uma nota indexada.
func (ix *Inverted) DocLength(path string) int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.docLengths[path]
}

// Update lê uma nota do vault, remove BOM, analisa os tokens e atualiza o índice invertido.
func (ix *Inverted) Update(ctx context.Context, v *vault.Vault, path vault.CanonicalPath) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	abs := v.Abs(path)
	data, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			ix.Remove(string(path))
			return nil
		}
		return fmt.Errorf("lendo arquivo para indice invertido: %w", err)
	}

	stripped, _ := vault.StripBOM(data)
	tokens := Analyze(string(stripped))
	ix.Add(string(path), tokens)
	return nil
}

// ExportTermsForCache faz uma cópia thread-safe dos termos do índice invertido para persistência.
func (ix *Inverted) ExportTermsForCache() map[string]map[string][]TokenPosition {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	out := make(map[string]map[string][]TokenPosition, len(ix.terms))
	for term, docMap := range ix.terms {
		dCopy := make(map[string][]TokenPosition, len(docMap))
		for path, posList := range docMap {
			pCopy := make([]TokenPosition, len(posList))
			copy(pCopy, posList)
			dCopy[path] = pCopy
		}
		out[term] = dCopy
	}
	return out
}

// NewInvertedFromCache reconstrói um índice invertido a partir de termos serializados.
func NewInvertedFromCache(terms map[string]map[string][]TokenPosition) *Inverted {
	inv := NewInverted()
	inv.mu.Lock()
	defer inv.mu.Unlock()

	docLengths := make(map[string]int)
	if terms != nil {
		inv.terms = terms
		for _, docMap := range terms {
			for path, posList := range docMap {
				docLengths[path] += len(posList)
			}
		}
	}
	inv.docLengths = docLengths
	return inv
}
