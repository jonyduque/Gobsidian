package search

import (
	"context"
	"unicode/utf8"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Constantes de recorte de trecho (RF-22, TOOLS.md schema).
const (
	DefaultSnippetChars = 240
	MaxSnippetChars     = 1000
)

// Snippet representa um trecho recortado e destacado de uma nota.
type Snippet struct {
	Text           string
	HighlightStart int
	HighlightEnd   int
	MatchedHeading string
}

// TermosDeTrecho e a lista de termos da consulta JA analisada e expandida.
//
// E um tipo, e nao um []string, de proposito. GenerateSnippet analisava os
// termos por HIT: com limit=200 e tres palavras, 600 chamadas a Analyze por
// busca produzindo sempre o mesmo resultado (achado P10). Tirar o Analyze de
// dentro resolveria isso, mas passar a esperar um []string ja expandido faria
// um chamador desavisado — que passasse os termos crus — perder a expansao por
// forma reduzida em SILENCIO, trocando custo por resultado pior.
//
// O tipo obriga a construcao a passar por NovosTermosDeTrecho, que e a unica
// conta da expansao.
type TermosDeTrecho struct {
	termos []string
}

// NovosTermosDeTrecho analisa e expande os termos uma vez so.
//
// Cada termo vira a forma crua e, quando diferente, a forma reduzida — a mesma
// expansao que acontecia dentro do laco por hit.
func NovosTermosDeTrecho(queryTerms []string) TermosDeTrecho {
	var out []string
	for _, termStr := range queryTerms {
		for _, tok := range Analyze(termStr) {
			out = append(out, tok.Raw)
			if tok.Reduced != "" && tok.Reduced != tok.Raw {
				out = append(out, tok.Reduced)
			}
		}
	}
	return TermosDeTrecho{termos: out}
}

// Vazio diz se nao ha termo nenhum para procurar.
func (t TermosDeTrecho) Vazio() bool { return len(t.termos) == 0 }

// GenerateSnippet recorta do disco o trecho em volta das ocorrências dos termos de busca.
//
// cache pode ser nil, e nil é o caminho sem cache — é o que o benchmark frio
// mede e o que o RNF-04 cobra. Só é cacheada a nota que está no índice de
// metadados: sem o hash dela não existe chave de invalidação, e servir bytes
// velhos com confiança é pior que reler o disco.
func GenerateSnippet(ctx context.Context, v *vault.Vault, ix *Inverted, idx *index.Index, path string, queryTerms TermosDeTrecho, maxChars int, cache *SnippetCache) (Snippet, error) {
	if maxChars <= 0 {
		maxChars = DefaultSnippetChars
	}
	if maxChars > MaxSnippetChars {
		maxChars = MaxSnippetChars
	}

	cPath := vault.CanonicalPath(path)

	// 1. Arquivo somente-nuvem NUNCA é aberto para evitar downloads síncronos.
	// O cache entra DEPOIS desta saída: um placeholder não pode ganhar entrada
	// no cache, senão o caminho que existe para não abrir o arquivo passaria a
	// depender de nunca ter aberto antes.
	var noteHadBOM bool
	var headings []parser.Heading
	var noteHash uint64
	var temHash bool
	if idx != nil {
		note, ok := idx.Get(cPath)
		if ok {
			if note.CloudOnly {
				return Snippet{}, nil
			}
			noteHadBOM = note.BOM
			headings = note.Headings
			// idx.Get já foi pago acima: o hash sai sem syscall de stat.
			noteHash = note.Hash
			temHash = true
		}
	}

	if ix == nil || queryTerms.Vazio() {
		return Snippet{}, nil
	}

	// 2. Coleta posições do primeiro termo correspondente no índice invertido
	type matchPos struct {
		term  string
		start int64
		end   int64
	}

	var bestMatch *matchPos

	// A analise e a expansao ja aconteceram uma vez, em NovosTermosDeTrecho.
	// Aqui sobra so a consulta ao indice.
	for _, t := range queryTerms.termos {
		posicoes := ix.Positions(t, string(cPath))
		if len(posicoes) > 0 {
			bestMatch = &matchPos{
				term:  t,
				start: posicoes[0].Start,
				end:   posicoes[0].End,
			}
			break
		}
	}

	// Nenhum termo ocorre na nota: nada foi lido do disco, e um trecho vazio não
	// vale entrada no cache — despejaria trecho útil para guardar o que custa zero.
	if bestMatch == nil {
		return Snippet{}, nil
	}

	// 2.1. Consulta o cache. É aqui e não antes porque a chave inclui a
	// ocorrência escolhida, e achá-la custa uma busca binária (Positions), não
	// uma ida ao disco. O que o acerto pula é o par caro: ReadRange, que abre o
	// arquivo, e adjustUTF8Highlight.
	var chave chaveTrecho
	cacheavel := cache != nil && temHash
	if cacheavel {
		chave = chaveTrecho{
			path:     string(cPath),
			hash:     noteHash,
			start:    bestMatch.start,
			end:      bestMatch.end,
			maxChars: maxChars,
		}
		if trecho, ok := cache.Get(chave); ok {
			return trecho, nil
		}
	}

	// 3. Offset do BOM:
	// Posições no Inverted são relativas ao corpo sem BOM.
	// Se a nota no disco tem BOM (3 bytes \xEF\xBB\xBF), os offsets no disco são deslocados em +3 bytes.
	var bomOffset int64
	if noteHadBOM {
		bomOffset = int64(vault.BOMLen)
	}

	diskMatchStart := bestMatch.start + bomOffset
	diskMatchEnd := bestMatch.end + bomOffset

	var matchedHeadingText string
	for _, h := range headings {
		if diskMatchStart >= h.Start && diskMatchEnd <= h.End {
			matchedHeadingText = h.Text
			break
		}
	}

	// 4. Calcula a janela de recorte ao redor da ocorrência
	half := int64(maxChars / 2)
	winStart := diskMatchStart - half
	if winStart < 0 {
		winStart = 0
	}
	winEnd := winStart + int64(maxChars)

	buf, err := v.ReadRange(ctx, cPath, winStart, winEnd)
	if err != nil {
		data, readErr := v.ReadAll(ctx, cPath)
		if readErr != nil {
			// Falha de leitura é transitória — cofre desmontado, arquivo em uso.
			// Guardar o vazio aqui congelaria a falha até a nota mudar de hash,
			// então NÃO entra no cache.
			//
			// O erro é DEVOLVIDO, e não engolido (achado B3). Até 2026-08-28
			// todos os ramos desta função devolviam nil e o chamador escrevia
			// `snip, _ :=`: uma nota ilegível produzia trecho vazio,
			// indistinguível de "o termo não aparece no corpo". "Cofre
			// inacessível e cofre vazio não podem dar a mesma resposta" é regra
			// desta base, e ela vale para uma nota também.
			//
			// Quem chama NÃO deve derrubar a página por causa disto: uma nota
			// travada pelo Obsidian não pode apagar as outras 199. Ver
			// Service.Search, que conta as falhas e as publica.
			return Snippet{}, readErr
		}
		buf = data
		winStart = 0
	}

	// Trima sequências UTF-8 parciais nas extremidades e calcula destaques relativos ao trecho
	snippetText, relStart, relEnd := adjustUTF8Highlight(buf, winStart, diskMatchStart, diskMatchEnd)

	trecho := Snippet{
		Text:           snippetText,
		HighlightStart: relStart,
		HighlightEnd:   relEnd,
		MatchedHeading: matchedHeadingText,
	}

	// Guarda o que foi REALMENTE produzido, inclusive quando o recorte veio da
	// recuperação por ReadAll com winStart = 0: o conteúdo é o mesmo da chave, e
	// gravar o valor do caminho feliz aqui guardaria um trecho que não existiu.
	if cacheavel {
		cache.Put(chave, trecho)
	}

	return trecho, nil
}

func adjustUTF8Highlight(buf []byte, winStart, matchStart, matchEnd int64) (string, int, int) {
	if len(buf) == 0 {
		return "", 0, 0
	}

	cropStart := 0
	for cropStart < len(buf) && !utf8.RuneStart(buf[cropStart]) {
		cropStart++
	}

	cropEnd := len(buf)
	for cropEnd > cropStart && !utf8.Valid(buf[cropStart:cropEnd]) {
		cropEnd--
	}

	slice := buf[cropStart:cropEnd]
	text := string(slice)

	relStart := int(matchStart - (winStart + int64(cropStart)))
	relEnd := int(matchEnd - (winStart + int64(cropStart)))

	if relStart < 0 {
		relStart = 0
	}
	if relEnd > len(text) {
		relEnd = len(text)
	}
	if relStart > relEnd {
		relStart = relEnd
	}

	return text, relStart, relEnd
}
