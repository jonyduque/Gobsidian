package search

import (
	"fmt"
	"sort"
)

// baseSoA é a metade IMUTÁVEL do índice invertido: exatamente o que veio do
// cache, em arrays achatados, sem um mapa por termo.
//
// # O que isto substitui e por quê
//
// A representação anterior era `map[string]map[string][]TokenPosition`. No
// cofre de referência isso são 126.342 mapas internos, 3.020.792 entradas e
// 3.020.792 fatias — e montá-los era o que sobrava depois de o formato do
// arquivo já ter sido resolvido: 35% do tempo de carga em `aeshashbody`,
// `mapassign_faststr` e `matchH2`, mais a maior parte das 643.413 alocações.
//
// Aqui cada estrutura é UM array. A carga preenche seis deles em ordem
// sequencial, sem hash e sem realocação, porque o arquivo declara os totais.
//
// Layout
//
//	termos[i]                      termo i, em ordem crescente
//	postings do termo i            [termIni[i], termIni[i+1])
//	postPath[j]                    pathID da posting j
//	posições da posting j          pos[postIni[j]:postIni[j+1]]
//	caminhos[p]                    caminho do documento p, em ordem crescente
//	docLen[p]                      número de tokens do documento p
//
// A ordem NÃO é decorativa: `termos` e `caminhos` ordenados são o que permite
// busca binária no lugar do mapa, e `postPath` ordenado dentro de cada termo é
// o que faz Postings devolver resultado já ordenado por caminho. O leitor
// confere as três ordens (ver validaOrdem) porque um arquivo fora de ordem não
// falharia: ele responderia "termo não existe" para termos que existem.
//
// Índice direto (documento -> termos)
//
// docTermIni/docTermos é a transposta de postPath. Existe para uma operação só,
// e ela é a que decidia o custo de editar uma nota: marcar um documento como
// substituído precisa saber a quais termos ele pertencia. Sem a transposta isso
// é uma varredura dos 126 mil termos — que é literalmente o que
// `removeLocked` fazia a CADA arquivo mudado, no mapa antigo. Com ela é O(termos
// da nota).
//
// Ela não é lida do arquivo: sai de duas passadas lineares sobre postPath
// (contagem e distribuição), sem ordenação, porque percorrer os termos em ordem
// crescente já deposita os termIDs de cada documento em ordem.
type baseSoA struct {
	termos  []string
	termIni []int32

	postPath []int32
	postIni  []int32
	pos      []TokenPosition

	caminhos []string
	docLen   []int32

	// idPorCaminho é o único mapa que sobra, e tem uma entrada por DOCUMENTO —
	// 3.152 no cofre de referência, contra os 3 milhões de entradas que a
	// representação antiga tinha. Ele existe porque DocLength é chamado dentro
	// do laço de postings do BM25, e ali uma busca binária sobre caminhos que
	// compartilham prefixo longo ("pasta08/nota0008.md") custa mais que um hash.
	idPorCaminho map[string]int32

	docTermIni []int32
	docTermos  []int32

	// arenaFechar desfaz o mapeamento de arquivo que fornece `pos`, quando
	// pos vem de uma arena mapeada (ver mmap.go) em vez de um []TokenPosition
	// alocado no heap. nil no caminho comum (decodificação integral).
	//
	// SaveInvertedCache chama promoverArenaSePresente ANTES do os.Rename do
	// salvamento atômico: renomear por cima de um arquivo que este processo
	// ainda tem mapeado falha no Windows (ERROR_SHARING_VIOLATION). Promover
	// copia `pos` para o heap e desfaz o mapeamento primeiro — custa uma
	// cópia só no caminho raro de cache parcial retomado e depois regravado;
	// no caminho comum (cache completo, "pronta") SaveInvertedCache nunca
	// roda de novo neste processo, então isto nunca dispara.
	arenaFechar func() error
}

// buscaTermo devolve o índice do termo e se ele existe.
func (b *baseSoA) buscaTermo(termo string) (int32, bool) {
	i := sort.SearchStrings(b.termos, termo)
	if i < len(b.termos) && b.termos[i] == termo {
		return int32(i), true
	}
	return 0, false
}

// termosDoDoc devolve os termIDs a que o documento pertence.
func (b *baseSoA) termosDoDoc(id int32) []int32 {
	return b.docTermos[b.docTermIni[id]:b.docTermIni[id+1]]
}

// posicoesDaPosting devolve as posições da posting j.
//
// Capacidade travada em `fim`: a fatia é uma janela dentro do array de posições
// compartilhado por TODAS as postings, e um append feito por quem recebeu a
// fatia escreveria por cima da primeira posição da posting vizinha — em
// silêncio, e numa nota diferente da que o chamador estava mexendo. Com
// cap == len o append copia para fora.
func (b *baseSoA) posicoesDaPosting(j int32) []TokenPosition {
	ini, fim := b.postIni[j], b.postIni[j+1]
	return b.pos[ini:fim:fim]
}

// numPostings devolve quantas postings o termo tem.
func (b *baseSoA) numPostings(t int32) int32 {
	return b.termIni[t+1] - b.termIni[t]
}

// validaOrdem confere as três ordens de que a busca binária depende.
//
// Fora de ordem, `sort.SearchStrings` não erra: ela devolve "não existe" para
// um termo que existe, e a busca passa a não achar notas que contêm a palavra —
// sem erro, sem log, e com aparência de cofre que simplesmente não tem o termo.
// É barato conferir (três varreduras lineares) e caro descobrir depois.
func (b *baseSoA) validaOrdem() error {
	for i := 1; i < len(b.termos); i++ {
		if b.termos[i-1] >= b.termos[i] {
			return fmt.Errorf("%w: termos fora de ordem em %d: %q >= %q",
				ErrCacheCorrupted, i, b.termos[i-1], b.termos[i])
		}
	}
	for i := 1; i < len(b.caminhos); i++ {
		if b.caminhos[i-1] >= b.caminhos[i] {
			return fmt.Errorf("%w: caminhos fora de ordem em %d: %q >= %q",
				ErrCacheCorrupted, i, b.caminhos[i-1], b.caminhos[i])
		}
	}
	for t := 0; t+1 < len(b.termIni); t++ {
		for j := b.termIni[t] + 1; j < b.termIni[t+1]; j++ {
			if b.postPath[j-1] >= b.postPath[j] {
				return fmt.Errorf("%w: postings do termo %d fora de ordem: %d >= %d",
					ErrCacheCorrupted, t, b.postPath[j-1], b.postPath[j])
			}
		}
	}
	return nil
}

// montaIndiceDireto constrói docTermIni/docTermos a partir de postPath.
//
// Duas passadas lineares, sem ordenação e sem mapa. A segunda percorre os
// termos em ordem crescente, então os termIDs de cada documento saem ordenados
// como efeito colateral — não é preciso ordenar depois, e não se deve passar a
// depender disso sem dizer aqui.
func (b *baseSoA) montaIndiceDireto() {
	nDocs := len(b.caminhos)
	b.docTermIni = make([]int32, nDocs+1)

	for _, p := range b.postPath {
		b.docTermIni[p+1]++
	}
	for i := 1; i <= nDocs; i++ {
		b.docTermIni[i] += b.docTermIni[i-1]
	}

	b.docTermos = make([]int32, len(b.postPath))
	cursor := make([]int32, nDocs)
	copy(cursor, b.docTermIni[:nDocs])

	for t := 0; t+1 < len(b.termIni); t++ {
		for j := b.termIni[t]; j < b.termIni[t+1]; j++ {
			p := b.postPath[j]
			b.docTermos[cursor[p]] = int32(t)
			cursor[p]++
		}
	}
}

// montaIDs preenche idPorCaminho.
func (b *baseSoA) montaIDs() {
	b.idPorCaminho = make(map[string]int32, len(b.caminhos))
	for i, c := range b.caminhos {
		b.idPorCaminho[c] = int32(i)
	}
}

// buscaPosting localiza a posting do documento `id` dentro do termo `t`.
//
// Busca binária sobre postPath, que é crescente dentro de cada termo. Um termo
// muito comum tem milhares de postings, e Positions é chamado por termo da
// consulta e por nota do resultado ao montar os trechos.
func (b *baseSoA) buscaPosting(t, id int32) (int32, bool) {
	ini, fim := b.termIni[t], b.termIni[t+1]
	faixa := b.postPath[ini:fim]
	k := sort.Search(len(faixa), func(i int) bool { return faixa[i] >= id })
	if k < len(faixa) && faixa[k] == id {
		return ini + int32(k), true
	}
	return 0, false
}
