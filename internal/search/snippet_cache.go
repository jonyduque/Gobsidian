package search

import (
	"container/list"
	"sync"
)

// DefaultSnippetCacheEntries e o teto de entradas do cache de trechos.
//
// Conta de memoria, por entrada: o Snippet guardado tem Text de no maximo
// MaxSnippetChars (1000) bytes; a chave soma o caminho da nota (~60 B num
// cofre real) mais 32 B de inteiros; o no da lista e a entrada do mapa custam
// ~100 B juntos. Pior caso 1024 x ~1,2 KB = ~1,2 MB; com o recorte padrao de
// DefaultSnippetChars (240) fica em ~450 KB.
//
// O teto nao e conservadorismo gratuito: o RNF-07 cobra RSS <= 60 MB (alvo) /
// 150 MB (degradado) e a medicao atual e 67 MB. Nao ha folga para um cache sem
// teto.
const DefaultSnippetCacheEntries = 1024

// chaveTrecho identifica um trecho ja recortado.
//
// hash e o xxhash do conteudo BRUTO da nota (index.Note.Hash), e e ele que faz
// a invalidacao. NAO existe Invalidate() neste cache, e isso e decisao, nao
// esquecimento: nota editada muda de hash, logo muda de chave, logo o acerto
// nao acontece, e a entrada velha morre por LRU. Nota que nao esta no indice de
// metadados nao tem hash, logo nao tem chave de invalidacao, logo nao e
// cacheada — ver GenerateSnippet.
//
// start e end sao os offsets da ocorrencia escolhida (relativos ao corpo sem
// BOM), e maxChars e o tamanho pedido: os dois entram na chave porque a janela
// recortada e derivada deles, e duas consultas sobre a mesma nota podem cair em
// ocorrencias diferentes.
type chaveTrecho struct {
	path     string
	hash     uint64
	start    int64
	end      int64
	maxChars int
}

// entradaTrecho e o que a lista de uso guarda. A chave viaja junto com o valor
// porque o despejo parte do no mais velho da lista e precisa saber qual chave
// apagar do mapa.
type entradaTrecho struct {
	chave  chaveTrecho
	trecho Snippet
}

// SnippetCache guarda trechos ja recortados, com despejo LRU por numero de
// entradas.
//
// O ponteiro nil e um cache DESLIGADO, e todos os metodos o aceitam. Isso nao e
// conveniencia: e o que permite ao benchmark medir o caminho frio, que e o que
// o RNF-04 cobra.
//
// O pool de recorte de service.Search chama isto de varias goroutines ao mesmo
// tempo, entao o mutex protege mapa e lista juntos.
type SnippetCache struct {
	mu    sync.Mutex
	teto  int
	ordem *list.List // frente = usado mais recentemente
	itens map[chaveTrecho]*list.Element
}

// NewSnippetCache cria um cache com o teto de entradas dado.
//
// teto <= 0 devolve nil, o cache desligado. Ter uma unica representacao de
// "desligado" evita um segundo caminho (cache com teto zero) que despejaria a
// cada Put e pagaria mutex para nunca acertar.
func NewSnippetCache(teto int) *SnippetCache {
	if teto <= 0 {
		return nil
	}
	return &SnippetCache{
		teto:  teto,
		ordem: list.New(),
		itens: make(map[chaveTrecho]*list.Element, teto),
	}
}

// Get devolve o trecho guardado para a chave, se houver.
func (c *SnippetCache) Get(k chaveTrecho) (Snippet, bool) {
	if c == nil {
		return Snippet{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.itens[k]
	if !ok {
		return Snippet{}, false
	}
	c.ordem.MoveToFront(el)
	return el.Value.(*entradaTrecho).trecho, true
}

// Put guarda o trecho, despejando o menos usado recentemente se estourar o teto.
func (c *SnippetCache) Put(k chaveTrecho, s Snippet) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.itens[k]; ok {
		el.Value.(*entradaTrecho).trecho = s
		c.ordem.MoveToFront(el)
		return
	}

	c.itens[k] = c.ordem.PushFront(&entradaTrecho{chave: k, trecho: s})

	for c.ordem.Len() > c.teto {
		velho := c.ordem.Back()
		if velho == nil {
			return
		}
		c.ordem.Remove(velho)
		delete(c.itens, velho.Value.(*entradaTrecho).chave)
	}
}

// Len devolve quantas entradas o cache guarda. Existe para o teste afirmar que
// um caminho NAO cacheou — "devolveu vazio" e "nao poluiu o cache" sao duas
// coisas, e so a segunda pega uma entrada envenenada.
func (c *SnippetCache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ordem.Len()
}
