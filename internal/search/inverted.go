package search

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"

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

	// building diz que o índice NÃO cobre o cofre inteiro ainda.
	//
	// Existe porque o servidor passou a construí-lo em segundo plano: num cofre
	// de 109 MB a tokenização levava 220 s, e o host desiste do handshake MCP
	// em 30 s. Enquanto ele está incompleto, uma busca acharia menos notas do
	// que existem — e devolver isso como resultado faz "ainda não sei" chegar
	// ao outro lado com a mesma cara de "não achei nada".
	//
	// atomic.Bool, e não protegido por ix.mu, de propósito: quem lê é o caminho
	// de busca, que já pega RLock para o resto; empilhar a leitura desta flag
	// dentro do mesmo lock daria a impressão de que ela é parte do estado
	// consistente do índice, e ela não é — é um estado de ciclo de vida.
	building atomic.Bool
}

// NewInverted inicializa e devolve um novo índice invertido, PRONTO.
//
// Pronto por padrão porque a maioria dos chamadores — testes, watcher,
// reconstrução incremental — monta o índice inteiro antes de servir qualquer
// consulta. Quem constrói em segundo plano chama MarkBuilding logo em seguida.
func NewInverted() *Inverted {
	return &Inverted{
		terms:      make(map[string]map[string][]TokenPosition),
		docLengths: make(map[string]int),
	}
}

// MarkBuilding declara que o índice está incompleto.
func (ix *Inverted) MarkBuilding() { ix.building.Store(true) }

// MarkReady declara que o índice cobre o cofre inteiro.
func (ix *Inverted) MarkReady() { ix.building.Store(false) }

// Building diz se o índice ainda está sendo construído.
func (ix *Inverted) Building() bool { return ix.building.Load() }

// Add insere ou substitui as ocorrências dos tokens para o caminho especificado.
func (ix *Inverted) Add(path string, tokens []Token) {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	ix.removeLocked(path)

	// Nota sem token nenhum entra em docLengths com zero, e nao fica de fora.
	//
	// Ficando de fora ela nunca contava como coberta: o cabecalho do cache
	// gravava DocCount, o boot comparava com a contagem do indice de metadados,
	// e um cofre com qualquer nota vazia dava "cache parcial" em TODO boot —
	// medido no cofre de referencia, 4 notas vazias em 3.152 custavam uma
	// reconstrucao e uma regravacao do cache inteiro a cada partida.
	ix.docLengths[path] = len(tokens)
	if len(tokens) == 0 {
		return
	}

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
	if !ok {
		return nil
	}

	result := make([]Posting, 0, len(docs))
	for path, posList := range docs {
		result = append(result, Posting{
			Path:      path,
			Positions: posList,
			Frequency: len(posList),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result
}

// Positions devolve as posicoes de um termo em uma nota especifica.
func (ix *Inverted) Positions(term, path string) []TokenPosition {
	norm := Normalize(term)

	ix.mu.RLock()
	defer ix.mu.RUnlock()

	docs, ok := ix.terms[norm]
	if !ok {
		return nil
	}
	posList := docs[path]
	if len(posList) == 0 {
		return nil
	}
	res := make([]TokenPosition, len(posList))
	copy(res, posList)
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

// HasDoc diz se a nota está no índice, INCLUSIVE quando ela não tem token
// nenhum.
//
// Existe separada de DocLength porque `DocLength(p) > 0` responde outra
// pergunta: uma nota vazia devolve 0 e seria lida de novo a cada retomada,
// para sempre, sem nunca passar a contar como coberta.
func (ix *Inverted) HasDoc(path string) bool {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	_, ok := ix.docLengths[path]
	return ok
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

// ExportForCache faz uma cópia thread-safe do que vai para o disco.
//
// Devolve os dois mapas de uma vez, sob o MESMO RLock. Buscá-los em duas
// chamadas deixaria uma janela em que o watcher indexa uma nota entre elas, e o
// cache sairia com uma nota nas postings e ausente em docLengths — o bastante
// para o boot seguinte declarar o cache incompleto e reconstruir tudo.
func (ix *Inverted) ExportForCache() (map[string]map[string][]TokenPosition, map[string]int) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	comp := make(map[string]int, len(ix.docLengths))
	for path, n := range ix.docLengths {
		comp[path] = n
	}

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
	return out, comp
}

// ErrIndiceNaoVazio recusa a adoção de um cache sobre um índice que já recebeu
// escritas. Ver AdotarDe.
var ErrIndiceNaoVazio = errors.New("indice invertido nao esta vazio")

// AdotarDe move o conteúdo de `outro` para dentro deste índice, sem copiar.
//
// Existe para que o cache possa ser carregado FORA do caminho de boot. O
// watcher e o serviço recebem o ponteiro do índice antes de o cache estar lido;
// trocar o ponteiro depois não é opção, então o conteúdo é que entra no índice
// que eles já têm.
//
// Quem chama precisa garantir duas coisas:
//
//   - `outro` não pode estar acessível a mais ninguém. Ele é esvaziado.
//   - nada pode ter escrito neste índice desde que ele foi criado.
//
// A segunda condição é a perigosa, e por isso ela NÃO fica só no comentário: a
// adoção substitui o conteúdo, então um Add que tenha chegado antes dela seria
// perdido sem deixar rastro, e o sintoma seria uma nota editada durante o boot
// sumir da busca até o próximo reinício — um dado errado servido com confiança,
// que é o modo de falha mais caro desta área. Índice não vazio devolve
// ErrIndiceNaoVazio e nada é tocado; cabe a quem chama construir do zero, que é
// lento e correto.
func (ix *Inverted) AdotarDe(outro *Inverted) error {
	if outro == nil {
		return nil
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	outro.mu.Lock()
	defer outro.mu.Unlock()

	if len(ix.terms) > 0 || len(ix.docLengths) > 0 {
		return fmt.Errorf("%w: %d termos e %d documentos ja no indice",
			ErrIndiceNaoVazio, len(ix.terms), len(ix.docLengths))
	}

	ix.terms = outro.terms
	ix.docLengths = outro.docLengths

	// `outro` fica vazio e utilizável, e não com mapas nil: um índice zerado por
	// engano deve responder "sem resultados", não entrar em pânico.
	outro.terms = make(map[string]map[string][]TokenPosition)
	outro.docLengths = make(map[string]int)

	return nil
}

// NewInvertedFromCache reconstrói um índice invertido a partir de termos
// serializados e dos comprimentos por documento.
//
// `docLengths` vem pronto de quem decodificou. Antes esta função o recalculava
// varrendo os termos inteiros — 3 milhões de postings espalhadas pelo heap,
// medidas em 0,71 s no cofre de referência, gastos só para somar um número que
// o decodificador já tinha em mãos ao ler cada posting. Nada aqui pode
// depender dessa varredura de novo.
//
// `docLengths` nil é aceito e vira mapa vazio: cofre sem nota nenhuma é estado
// legítimo, e distinguir isso de cache ausente é responsabilidade de
// LoadInvertedCache, não desta função.
func NewInvertedFromCache(terms map[string]map[string][]TokenPosition, docLengths map[string]int) *Inverted {
	inv := NewInverted()
	inv.mu.Lock()
	defer inv.mu.Unlock()

	if terms != nil {
		inv.terms = terms
	}
	if docLengths == nil {
		docLengths = make(map[string]int)
	}
	inv.docLengths = docLengths
	return inv
}
