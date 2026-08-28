package search

import (
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/parser"
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
//
// # Espaço de IDs densos (Oportunidade 1)
//
// Até 2026-08-28 esta função trabalhava em caminhos: `docTermFreqs` era um
// `map[string]map[int]float64` — **um mapa alocado por documento candidato** —
// mais um `map[string]bool` por token de consulta e um `map[string]float64`
// para os scores. No perfil de `BenchmarkSearchLimit200Cache`, `CalculateBM25`
// respondia sozinha por **79% dos bytes alocados pela busca**, 96 MB só de
// alocação direta, e ~31% do tempo dela eram operações de mapa.
//
// Agora os candidatos recebem um ID compacto na PRIMEIRA passada — um índice
// 0..M-1, onde M é o número de documentos que casaram algum termo — e tudo
// depois disso é fatia indexada por esse ID: frequências, comprimentos, scores.
// Sobra um mapa só, `idPorCaminho`, com uma entrada por candidato em vez de uma
// por (candidato × termo).
//
// O `baseSoA` já trabalha em IDs int32 internamente; o que se perdia era na
// volta, porque `Postings` reconverte para string. Fechar isso de ponta a ponta
// exigiria mudar a API de `Inverted`, o que atinge trecho, serviço e testes —
// ficou de fora, e o ganho medido veio sem isso.
func CalculateBM25(queryTokens []Token, ix *Inverted, idx *index.Index) []Result {
	if ix == nil || ix.DocCount() == 0 || len(queryTokens) == 0 {
		return nil
	}

	totalDocs := ix.DocCount()
	N := float64(totalDocs)

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

	// Busca as postings uma vez por termo distinto e reusa — na pontuação e no
	// IDF (perfil anterior: 371,40 MB em Postings por refazer a busca).
	termPostings := make(map[string][]Posting)
	for _, m := range matches {
		if _, exists := termPostings[m.term]; !exists {
			termPostings[m.term] = ix.Postings(m.term)
		}
	}

	// PRIMEIRA PASSADA: atribui um ID compacto a cada documento candidato.
	//
	// Uma entrada de mapa por CANDIDATO, e não por (candidato × termo) como
	// antes. Daqui para baixo nada mais toca em mapa no caminho quente.
	idPorCaminho := make(map[string]int32)
	var caminhoPorID []string
	for _, m := range matches {
		for _, p := range termPostings[m.term] {
			if _, visto := idPorCaminho[p.Path]; !visto {
				idPorCaminho[p.Path] = int32(len(caminhoPorID))
				caminhoPorID = append(caminhoPorID, p.Path)
			}
		}
	}
	if len(caminhoPorID) == 0 {
		return nil
	}

	M := len(caminhoPorID)
	nq := len(queryTokens)

	// tf é uma matriz M×nq achatada: tf[doc*nq+termo]. Uma alocação, contígua,
	// contra os M mapas de antes.
	tf := make([]float64, M*nq)

	// docsPorTermo conta em quantos documentos cada token da consulta ocorre —
	// o `nq` do IDF. `marcado` faz o papel do `map[string]bool` por token, com
	// um carimbo por documento: o valor guardado é o índice do token que o
	// marcou por último, então não é preciso limpar a fatia entre tokens.
	docsPorTermo := make([]int, nq)
	marcado := make([]int32, M)
	for i := range marcado {
		marcado[i] = -1
	}

	for _, m := range matches {
		postings := termPostings[m.term]
		if len(postings) == 0 {
			continue
		}
		for _, p := range postings {
			id := idPorCaminho[p.Path]

			if marcado[id] != int32(m.queryIdx) {
				marcado[id] = int32(m.queryIdx)
				docsPorTermo[m.queryIdx]++
			}

			// Resolvido UMA vez por documento, e não por ocorrência (achado
			// P2): `idx.Get` toma RLock, e a decisão de título é a mesma para
			// todas as ocorrências do mesmo termo no mesmo documento — só o
			// heading depende da posição.
			var n *index.Note
			if idx != nil {
				if nota, ok := idx.Get(vault.CanonicalPath(p.Path)); ok {
					n = nota
				}
			}
			tituloCasa := n != nil && tituloContemTermo(n.TitleNorm, m.term)

			var soma float64
			for _, pos := range p.Positions {
				soma += pesoDeCampo(n, pos, tituloCasa)
			}
			tf[int(id)*nq+m.queryIdx] += soma * m.matchMult
		}
	}

	// avgdl vem memorizado do índice invertido, e não de um percurso do cofre
	// inteiro por consulta (achado P1).
	avgdl := float64(ix.SomaDocLen()) / N
	if avgdl == 0 {
		avgdl = 1.0
	}

	idfs := make([]float64, nq)
	for i := range idfs {
		d := float64(docsPorTermo[i])
		if d > 0 {
			idfs[i] = math.Log(1.0 + (N-d+0.5)/(d+0.5))
		}
	}

	// Comprimento de cada candidato, uma consulta por documento. Antes era uma
	// por documento POR ITERAÇÃO de pontuação, cada uma tomando o RLock.
	docLens := make([]float64, M)
	for id, caminho := range caminhoPorID {
		docLens[id] = float64(ix.DocLength(caminho))
	}

	results := make([]Result, 0, M)
	for id := 0; id < M; id++ {
		var score float64
		base := id * nq
		for q := 0; q < nq; q++ {
			f := tf[base+q]
			idf := idfs[q]
			if idf <= 0 || f <= 0 {
				continue
			}
			denom := f + ParamK1*(1.0-ParamB+ParamB*(docLens[id]/avgdl))
			if denom > 0 {
				score += idf * (f * (ParamK1 + 1.0)) / denom
			}
		}
		if !math.IsNaN(score) && score > 0 {
			results = append(results, Result{Path: caminhoPorID[id], Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Path < results[j].Path
	})

	return results
}

// pesoDeCampo decide o peso de UMA ocorrencia, dado o que ja foi resolvido uma
// vez por documento.
//
// Ate 2026-08-28 esta funcao recebia so o caminho e refazia `idx.Get` — um
// RLock e uma busca em mapa — a cada OCORRENCIA do termo, e varria os headings
// linearmente na mesma frequencia (achado P2). Um termo comum numa nota longa
// paga isso centenas de vezes para responder a mesma pergunta.
//
// `n` e `tituloCasa` agora chegam prontos, resolvidos uma vez por documento em
// CalculateBM25.
func pesoDeCampo(n *index.Note, pos TokenPosition, tituloCasa bool) float64 {
	if n == nil {
		return WeightBody
	}
	if tituloCasa {
		return WeightTitle
	}
	if dentroDeHeading(n.Headings, pos) {
		return WeightHeadings
	}
	return WeightBody
}

// dentroDeHeading responde se a ocorrencia cai na LINHA de algum heading.
//
// Busca binaria, e nao varredura: os headings vem em ordem de documento por
// construcao do parser, entao o candidato e o ultimo com Start <= pos.Start —
// e so ele precisa ser conferido, porque as linhas de heading nao se
// sobrepoem. A varredura linear anterior custava O(headings) por ocorrencia.
func dentroDeHeading(hs []parser.Heading, pos TokenPosition) bool {
	if len(hs) == 0 {
		return false
	}
	i := sort.Search(len(hs), func(k int) bool { return hs[k].Start > pos.Start })
	if i == 0 {
		return false
	}
	h := hs[i-1]
	nivel := h.Level
	if nivel < 1 {
		nivel = 1
	}
	// "#"*nivel + " " + texto
	fimDaLinha := h.Start + int64(nivel) + 1 + int64(len(h.Text))
	return pos.Start >= h.Start && pos.Start < fimDaLinha
}

// tituloContemTermo diz se o termo aparece no titulo COMO TERMO.
//
// Ate 2026-08-28 o teste era `strings.Contains(n.TitleNorm, term)` — substring
// crua (achado P2). Uma busca por "ar" recebia peso de TITULO, o maior do
// sistema, de uma nota chamada "Barragem". Nao e ineficiencia: e resultado
// errado, e ele aparece exatamente nas consultas curtas, que sao as mais comuns.
//
// # Por que fronteira de token, e nao o analisador
//
// A primeira correcao tokenizava o titulo com `Analyze` e comparava contra a
// forma crua e a reduzida de cada token. Estava certa e era CARA: `Analyze`
// roda por documento candidato por termo, e um termo amplo tem milhares de
// candidatos. Medido intercalado em 2026-08-28: `BenchmarkSearchLimit200Cache`
// foi de 20,3 para 27,9 ms de mediana (+38%), com faixas disjuntas.
//
// A varredura por fronteira roda sobre `TitleNorm`, que o indice ja calculou e
// guarda, sem alocar nada. Ela corrige o defeito real — pedaco de palavra deixa
// de valer titulo — e custa O(len(titulo)).
//
// # O que ela nao cobre, e por que isso e aceitavel
//
// Titulo no PLURAL com consulta no singular: `TitleNorm` "processos civis" com
// termo "processo" nao casa, porque "processo" ali nao termina em fronteira.
// O caminho inverso — consulta "processos", que `Analyze` reduz para
// "processo", contra titulo "Processo civil" — CASA, e e o mais comum, porque
// e a consulta que costuma vir no plural. Cobrir o outro lado exigiria os
// tokens do titulo pre-calculados no indice, e `internal/index` nao pode
// importar `internal/search` (seria ciclo: search ja importa index).
func tituloContemTermo(tituloNorm, termo string) bool {
	if tituloNorm == "" || termo == "" {
		return false
	}
	for i := 0; ; {
		j := strings.Index(tituloNorm[i:], termo)
		if j < 0 {
			return false
		}
		ini := i + j
		fim := ini + len(termo)
		if fronteiraAntes(tituloNorm, ini) && fronteiraDepois(tituloNorm, fim) {
			return true
		}
		// Avanca UM byte, e nao um termo: ocorrencias podem se sobrepor, e
		// pular o termo inteiro perderia a que comeca dentro da anterior.
		i = ini + 1
		if i >= len(tituloNorm) {
			return false
		}
	}
}

// fronteiraAntes e fronteiraDepois decidem se a posicao inicia/termina uma
// palavra. Sao duas funcoes porque decodificam a runa em direcoes opostas —
// para tras e para frente —, e uma so com um booleano esconderia isso.
func fronteiraAntes(s string, i int) bool {
	if i == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	return !parteDePalavra(r)
}

func fronteiraDepois(s string, i int) bool {
	if i >= len(s) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return !parteDePalavra(r)
}

func parteDePalavra(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
