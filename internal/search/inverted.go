package search

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/jonyd/gobsidian/internal/index"
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

// Inverted é um índice invertido thread-safe com suporte a atualizações
// incrementais, em duas camadas.
//
// # Por que duas camadas
//
// O que vem do cache é grande e não muda: 126.342 termos e 3 milhões de
// postings no cofre de referência. O que muda é minúsculo: as notas que o
// usuário editou desde a partida. Guardar as duas coisas na mesma estrutura
// obrigava a estrutura grande a ser mutável, e mutável significava um mapa por
// termo — o que custava 35% do tempo de carga e a maior parte da memória.
//
//	base   arrays achatados, IMUTÁVEIS, vindos do cache (ver soa.go)
//	delta  mapas, pequenos, com o que mudou desde a carga
//
// Toda leitura consulta as duas e o delta ganha. Toda escrita vai só para o
// delta, e marca o caminho como substituído no base (`sombra`).
//
// Quando o índice é construído do zero — sem cache, ou cache incompleto — base
// é nil e o delta é o índice inteiro. Esse caminho é exatamente o de antes, o
// que importa porque é o caminho que a construção em segundo plano usa.
//
// # O que a sombra garante
//
// Um caminho em `sombra` está OBSOLETO no base. Nenhuma leitura pode devolver
// posting, posição ou comprimento do base para ele — se devolvesse, uma nota
// editada apareceria na busca com o conteúdo que tinha na partida, que é o tipo
// de erro que ninguém percebe até confiar nele.
type Inverted struct {
	mu sync.RWMutex

	// base é a metade imutável. nil quando o índice foi construído do zero.
	base *baseSoA

	// sombra guarda os caminhos do base já substituídos ou removidos. Só é
	// mantida quando base != nil; sem base não há o que sombrear.
	sombra map[string]bool

	// sombraNoBase conta quantas entradas de `sombra` existem no base. É o que
	// permite DocCount em O(1) — sem ele seria uma varredura por chamada, e
	// DocCount é chamado uma vez por consulta pelo BM25.
	sombraNoBase int

	// postsVivasPorTermo[t] é quantas postings do termo t do base ainda não
	// foram sombreadas. termosVivosBase conta quantos ainda têm alguma.
	//
	// Existem para TermCount continuar exato. A versão em mapa apagava o termo
	// quando ele ficava sem documento, e `len(ix.terms)` respondia sozinho;
	// aqui o base não pode ser alterado, então a contagem é mantida à parte.
	postsVivasPorTermo []int32
	termosVivosBase    int

	terms      map[string]map[string][]TokenPosition // delta: termo -> (path -> posList)
	docLengths map[string]int                        // delta: path -> total de tokens

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

	// avgdlCache guarda (geracao, valor) para evitar N chamadas a DocLength a
	// cada busca só para calcular uma constante do corpus. Uma geração
	// obsoleta sinaliza que o índice mudou e o valor precisa recalcular.
	avgdlGen    uint64
	avgdlCached float64
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

	ix.sombrearLocked(path)
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
	ix.sombrearLocked(path)
	ix.removeLocked(path)
}

// sombrearLocked declara que o base está obsoleto para este caminho.
//
// O custo desta função é o motivo de o base carregar um índice direto
// (documento -> termos). Na representação em mapa, o equivalente varria os
// 126.342 termos a CADA arquivo mudado, só para apagar o caminho de cada um.
// Aqui percorre os termos daquela nota — algumas centenas.
func (ix *Inverted) sombrearLocked(path string) {
	if ix.base == nil {
		return
	}
	if ix.sombra[path] {
		return
	}
	id, ok := ix.base.idPorCaminho[path]
	if !ok {
		// Caminho que não existe no base. Não há o que sombrear, e registrá-lo
		// faria sombraNoBase contar o que não está lá.
		return
	}
	ix.sombra[path] = true
	ix.sombraNoBase++
	for _, t := range ix.base.termosDoDoc(id) {
		ix.postsVivasPorTermo[t]--
		if ix.postsVivasPorTermo[t] == 0 {
			ix.termosVivosBase--
		}
	}
}

// removeLocked apaga o caminho do DELTA. O base é tratado por sombrearLocked.
func (ix *Inverted) removeLocked(path string) {
	delete(ix.docLengths, path)
	for term, docs := range ix.terms {
		delete(docs, path)
		if len(docs) == 0 {
			delete(ix.terms, term)
		}
	}
}

// obsoletoNoBaseLocked diz se o base deve ser ignorado para este caminho.
func (ix *Inverted) obsoletoNoBaseLocked(path string) bool {
	return ix.base == nil || ix.sombra[path]
}

// Postings devolve as ocorrências de um termo (normalizado) no índice.
// Os resultados são ordenados deterministicamente por caminho de nota.
func (ix *Inverted) Postings(term string) []Posting {
	norm := Normalize(term)

	ix.mu.RLock()
	defer ix.mu.RUnlock()

	doDelta := ix.terms[norm]

	var doBase []Posting
	if ix.base != nil {
		if t, ok := ix.base.buscaTermo(norm); ok {
			ini, fim := ix.base.termIni[t], ix.base.termIni[t+1]
			doBase = make([]Posting, 0, fim-ini)
			for j := ini; j < fim; j++ {
				path := ix.base.caminhos[ix.base.postPath[j]]
				if ix.sombra[path] {
					continue
				}
				pos := ix.base.posicoesDaPosting(j)
				doBase = append(doBase, Posting{
					Path:      path,
					Positions: pos,
					Frequency: len(pos),
				})
			}
		}
	}

	if len(doDelta) == 0 {
		// As postings do base já saem ordenadas por caminho, porque o pathID é
		// a posição no vetor ordenado de caminhos e o arquivo grava as postings
		// de cada termo por pathID crescente (conferido em validaOrdem). Sem o
		// delta não há o que reordenar, e este é o caso comum: consulta num
		// índice recém-carregado, sem edição nenhuma desde a partida.
		return doBase
	}

	result := doBase
	if result == nil {
		result = make([]Posting, 0, len(doDelta))
	}
	for path, posList := range doDelta {
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

	if posList := ix.terms[norm][path]; len(posList) > 0 {
		res := make([]TokenPosition, len(posList))
		copy(res, posList)
		return res
	}
	// O delta não tem. Se o caminho foi tocado desde a carga, o base está
	// obsoleto para ele e responder dali devolveria a versão da partida.
	if ix.obsoletoNoBaseLocked(path) {
		return nil
	}
	t, ok := ix.base.buscaTermo(norm)
	if !ok {
		return nil
	}
	id, ok := ix.base.idPorCaminho[path]
	if !ok {
		return nil
	}
	j, ok := ix.base.buscaPosting(t, id)
	if !ok {
		return nil
	}
	pos := ix.base.posicoesDaPosting(j)
	res := make([]TokenPosition, len(pos))
	copy(res, pos)
	return res
}

// HasTerm verifica se o termo (normalizado) existe no dicionário e possui ao menos um documento.
func (ix *Inverted) HasTerm(term string) bool {
	norm := Normalize(term)

	ix.mu.RLock()
	defer ix.mu.RUnlock()

	if docs := ix.terms[norm]; len(docs) > 0 {
		return true
	}
	if ix.base == nil {
		return false
	}
	t, ok := ix.base.buscaTermo(norm)
	return ok && ix.postsVivasPorTermo[t] > 0
}

// TermCount devolve a quantidade de termos ativos no dicionário.
//
// Percorre o delta e não o base: o base contribui com um contador mantido em
// sombrearLocked, e o delta tem os termos de algumas notas. O laço existe para
// não contar duas vezes um termo que está vivo nos dois lados.
func (ix *Inverted) TermCount() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	if ix.base == nil {
		return len(ix.terms)
	}
	n := ix.termosVivosBase
	for termo := range ix.terms {
		if t, ok := ix.base.buscaTermo(termo); ok && ix.postsVivasPorTermo[t] > 0 {
			continue
		}
		n++
	}
	return n
}

// DocCount devolve o número de notas indexadas.
//
// Um caminho sombreado sai do base e, se foi reindexado, volta pelo delta —
// contado uma vez. Se foi removido, não volta.
func (ix *Inverted) DocCount() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	if ix.base == nil {
		return len(ix.docLengths)
	}
	return len(ix.base.caminhos) - ix.sombraNoBase + len(ix.docLengths)
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
	if _, ok := ix.docLengths[path]; ok {
		return true
	}
	if ix.obsoletoNoBaseLocked(path) {
		return false
	}
	_, ok := ix.base.idPorCaminho[path]
	return ok
}

// DocPaths devolve os caminhos de todas as notas indexadas.
//
// Existe para a reconciliação por overflow: ela precisa tirar do índice o que
// sumiu do disco, e sem enumerar não há como saber o que sobrou aqui e não
// existe mais lá.
//
// Aloca a lista inteira de propósito, em vez de devolver um iterador sob o
// lock: quem chama vai chamar Remove para alguns dos caminhos, e Remove pega o
// mesmo lock.
func (ix *Inverted) DocPaths() []string {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	n := len(ix.docLengths)
	if ix.base != nil {
		n += len(ix.base.caminhos) - ix.sombraNoBase
	}
	out := make([]string, 0, n)
	if ix.base != nil {
		for _, p := range ix.base.caminhos {
			// Sombreado significa substituído ou removido; se foi
			// reindexado, ele volta pelo delta abaixo, uma vez só.
			if ix.sombra[p] {
				continue
			}
			out = append(out, p)
		}
	}
	for p := range ix.docLengths {
		out = append(out, p)
	}
	return out
}

// DocLength devolve o número total de tokens de uma nota indexada.
func (ix *Inverted) DocLength(path string) int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	if n, ok := ix.docLengths[path]; ok {
		return n
	}
	if ix.obsoletoNoBaseLocked(path) {
		return 0
	}
	id, ok := ix.base.idPorCaminho[path]
	if !ok {
		return 0
	}
	return int(ix.base.docLen[id])
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

	// O base entra primeiro e o delta sobrescreve, que é a mesma precedência
	// das leituras. Materializar mapas aqui é caro, e é deliberado: esta função
	// só roda em SaveInvertedCache, que por sua vez só roda dentro da
	// construção do índice. No caminho de cache completo — o comum — nada disto
	// executa. Se um dia a gravação passar a acontecer com o índice já
	// carregado, este é o lugar que vai precisar de um iterador em vez de uma
	// cópia.
	nDocs := len(ix.docLengths)
	nTermos := len(ix.terms)
	if ix.base != nil {
		nDocs += len(ix.base.caminhos) - ix.sombraNoBase
		nTermos += ix.termosVivosBase
	}

	comp := make(map[string]int, nDocs)
	out := make(map[string]map[string][]TokenPosition, nTermos)

	if ix.base != nil {
		for id, path := range ix.base.caminhos {
			if ix.sombra[path] {
				continue
			}
			comp[path] = int(ix.base.docLen[id])
		}
		for t := 0; t+1 < len(ix.base.termIni); t++ {
			ini, fim := ix.base.termIni[t], ix.base.termIni[t+1]
			var dCopy map[string][]TokenPosition
			for j := ini; j < fim; j++ {
				path := ix.base.caminhos[ix.base.postPath[j]]
				if ix.sombra[path] {
					continue
				}
				if dCopy == nil {
					dCopy = make(map[string][]TokenPosition, fim-ini)
				}
				pos := ix.base.posicoesDaPosting(j)
				pCopy := make([]TokenPosition, len(pos))
				copy(pCopy, pos)
				dCopy[path] = pCopy
			}
			// Termo sem nenhuma posting viva não entra: a versão em mapa
			// apagava o termo quando ele esvaziava, e o cache gravado precisa
			// dizer a mesma coisa que o índice em memória diz.
			if dCopy != nil {
				out[ix.base.termos[t]] = dCopy
			}
		}
	}

	for path, n := range ix.docLengths {
		comp[path] = n
	}
	for term, docMap := range ix.terms {
		dCopy := out[term]
		if dCopy == nil {
			dCopy = make(map[string][]TokenPosition, len(docMap))
			out[term] = dCopy
		}
		for path, posList := range docMap {
			pCopy := make([]TokenPosition, len(posList))
			copy(pCopy, posList)
			dCopy[path] = pCopy
		}
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

	if len(ix.terms) > 0 || len(ix.docLengths) > 0 || ix.base != nil {
		return fmt.Errorf("%w: %d termos e %d documentos no delta, base presente=%t",
			ErrIndiceNaoVazio, len(ix.terms), len(ix.docLengths), ix.base != nil)
	}

	ix.base = outro.base
	ix.sombra = outro.sombra
	ix.sombraNoBase = outro.sombraNoBase
	ix.postsVivasPorTermo = outro.postsVivasPorTermo
	ix.termosVivosBase = outro.termosVivosBase
	ix.terms = outro.terms
	ix.docLengths = outro.docLengths

	// `outro` fica vazio e utilizável, e não com mapas nil: um índice zerado por
	// engano deve responder "sem resultados", não entrar em pânico.
	outro.base = nil
	outro.sombra = nil
	outro.sombraNoBase = 0
	outro.postsVivasPorTermo = nil
	outro.termosVivosBase = 0
	outro.terms = make(map[string]map[string][]TokenPosition)
	outro.docLengths = make(map[string]int)

	return nil
}

// GetAvgdl devolve o comprimento médio dos documentos, usando cache
// invalidado por geração do índice. Quem chama passa a geração corrente
// do índice; se divergir da geração do cache, o valor é recalculado e
// guardado.
//
// idx pode ser nil. Se for, usa DocPaths() para iterar todos os documentos.
// Se DocCount for zero, devolve 1.0 como padrão.
func (ix *Inverted) GetAvgdl(idx *index.Index) float64 {
	// Determinar a geração para validação de cache
	var gen uint64
	if idx != nil {
		gen = idx.Generation()
		ix.mu.RLock()
		if gen == ix.avgdlGen && ix.avgdlCached > 0 {
			defer ix.mu.RUnlock()
			return ix.avgdlCached
		}
		ix.mu.RUnlock()
	}

	// Cache miss ou geração mudou — recalcular FORA do lock para evitar
	// deadlock ao chamar DocLength que também pega RLock.
	var sumDocLen float64

	if idx != nil {
		for _, p := range idx.Paths() {
			if dl := ix.DocLength(string(p)); dl > 0 {
				sumDocLen += float64(dl)
			}
		}
	} else {
		// Sem índice de metadados, usa todas as notas do invertido
		for _, p := range ix.DocPaths() {
			if dl := ix.DocLength(p); dl > 0 {
				sumDocLen += float64(dl)
			}
		}
	}

	totalDocs := ix.DocCount()
	if totalDocs == 0 {
		return 1.0
	}

	avgdl := sumDocLen / float64(totalDocs)
	if avgdl == 0 {
		avgdl = 1.0
	}

	// Armazenar no cache sob Write lock (só se temos índice para validar).
	if idx != nil {
		ix.mu.Lock()
		defer ix.mu.Unlock()

		// Checar novamente: outro goroutine pode ter recalculado enquanto
		// recalculávamos. Se a geração mudou de novo, não armazenar.
		if gen == idx.Generation() {
			ix.avgdlGen = gen
			ix.avgdlCached = avgdl
		}
	}

	return avgdl
}

// newInvertedFromSoA monta um índice sobre a base imutável vinda do cache.
//
// `postsVivasPorTermo` nasce com o total de postings de cada termo, e
// `termosVivosBase` com o total de termos: nada foi sombreado ainda. Os dois
// só descem, em sombrearLocked.
func newInvertedFromSoA(b *baseSoA) *Inverted {
	inv := NewInverted()
	if b == nil {
		return inv
	}
	inv.mu.Lock()
	defer inv.mu.Unlock()

	inv.base = b
	inv.sombra = make(map[string]bool)
	inv.postsVivasPorTermo = make([]int32, len(b.termos))
	for t := range b.termos {
		inv.postsVivasPorTermo[t] = b.numPostings(int32(t))
	}
	inv.termosVivosBase = len(b.termos)
	return inv
}
