package search

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// Codec binário do cache de busca, versão 2.
//
// O formato 1 era `gob` sobre `map[string]map[string][]TokenPosition`. Medido
// num cofre real de 3.148 notas e 109 MB, cujo cache dava 471,6 MB:
//
//	carregamento          5,03 s/op   3,94 GB/op   12.797.583 allocs/op
//	  reflexão do gob      ~3,6 s     72%
//	  materialização        1,40 s     28%
//
// Os 471,6 MB eram, na prática, duas coisas:
//
//	286,3 MB  caminhos REPETIDOS — 2.962.237 pares (termo,doc) × 101 chars
//	272,4 MB  posições — 17.849.848 × 16 bytes fixos
//
// Este formato ataca as duas. O caminho vira índice numa tabela escrita uma
// vez; a posição vira varint sobre o DELTA da anterior. Nada passa por
// reflexão: cada campo é lido por um loop que sabe o que está lendo.
//
// Layout, tudo em varint salvo os bytes crus de string:
//
//	magic      4 bytes  "GBS2"
//	versões    3 varints  (formato, parser, analisador)
//	vaultPath  string
//	noteCount  varint
//	totPost    varint    total de postings do arquivo inteiro
//	totPos     varint    total de posicoes do arquivo inteiro
//	nCaminhos  varint,  seguido de nCaminhos strings
//	nDocs      varint,  seguido de nDocs pares (pathID varint, tamanho varint)
//	nTermos    varint
//	  por termo:  string, nPostings varint
//	    por posting: pathID varint, nPos varint
//	      por posição: delta(Start) varint, (End-Start) varint
//
// `string` é sempre: comprimento em varint, seguido dos bytes.
const (
	cacheMagic     = "GBS4"
	cacheCodecVers = 4
)

// limiteRazoavel barra alocação gigante a partir de um arquivo corrompido.
//
// Sem isto, um comprimento adulterado — ou só um byte trocado por corrupção de
// disco — vira `make([]T, 4_000_000_000)` e o processo morre por OOM em vez de
// devolver "cache corrompido" e reconstruir. O cofre de referência tem 126 mil
// termos e 17,8 milhões de posições; estes tetos deixam ordens de grandeza de
// folga e ainda assim são finitos.
const (
	limiteCaminhos = 10_000_000
	limiteTermos   = 200_000_000
	limitePostings = 200_000_000
	limitePosicoes = 4_000_000_000
	limiteString   = 1 << 20
)

// --------------------------------------------------------------------------
// Escrita
// --------------------------------------------------------------------------

type escritor struct {
	w   *bufio.Writer
	buf [binary.MaxVarintLen64]byte
	err error
}

func (e *escritor) uvarint(v uint64) {
	if e.err != nil {
		return
	}
	n := binary.PutUvarint(e.buf[:], v)
	_, e.err = e.w.Write(e.buf[:n])
}

func (e *escritor) varint(v int64) {
	if e.err != nil {
		return
	}
	n := binary.PutVarint(e.buf[:], v)
	_, e.err = e.w.Write(e.buf[:n])
}

func (e *escritor) str(s string) {
	e.uvarint(uint64(len(s)))
	if e.err != nil {
		return
	}
	_, e.err = e.w.WriteString(s)
}

// escreveCache serializa o índice no formato 2.
//
// Recebe o mapa já exportado para não segurar o lock do índice durante a
// escrita em disco, que dura segundos.
func escreveCache(
	w io.Writer,
	h CacheHeader,
	termos map[string]map[string][]TokenPosition,
	docLengths map[string]int,
) error {
	e := &escritor{w: bufio.NewWriterSize(w, 1<<20)}

	if _, err := e.w.WriteString(cacheMagic); err != nil {
		return err
	}
	e.uvarint(uint64(h.FormatVersion))
	e.uvarint(uint64(h.ParserVersion))
	e.uvarint(uint64(h.AnalyzerVersion))
	e.str(h.VaultPath)
	e.uvarint(uint64(h.NoteCount))

	// Uma varredura só produz as duas coisas que precisam vir antes dos dados:
	// os totais e a tabela de caminhos. Os totais deixam o leitor dimensionar a
	// arena numa alocação; sem eles ela cresce por append, e acima de 256
	// elementos Go cresce 1,25×, então chegar aos 291 MB de posições do cofre de
	// referência custava ~1,45 GB em cópias — medido, 2,91 GB alocados no total
	// contra 66 MB de arquivo. Custam 8 bytes no arquivo.
	//
	// A tabela guarda cada caminho uma vez, e não uma vez por posting: era
	// metade do arquivo do formato antigo.
	var totPost, totPos uint64
	idPorCaminho := make(map[string]uint64, len(docLengths))
	caminhos := make([]string, 0, len(docLengths))
	registra := func(path string) uint64 {
		if id, ok := idPorCaminho[path]; ok {
			return id
		}
		id := uint64(len(caminhos))
		idPorCaminho[path] = id
		caminhos = append(caminhos, path)
		return id
	}
	// docLengths primeiro: uma nota SEM termo nenhum (arquivo vazio, ou só
	// pontuação) não aparece em posting alguma, e é justamente ela que precisa
	// entrar na tabela para o cache poder declarar que a cobre.
	for path := range docLengths {
		registra(path)
	}
	for _, docs := range termos {
		totPost += uint64(len(docs))
		for path, pos := range docs {
			totPos += uint64(len(pos))
			registra(path)
		}
	}
	e.uvarint(totPost)
	e.uvarint(totPos)

	e.uvarint(uint64(len(caminhos)))
	for _, p := range caminhos {
		e.str(p)
	}

	// docLengths é GRAVADO, e não mais derivado das postings na leitura.
	//
	// Derivar significava somar o número de posições de cada termo, e um token
	// cuja forma reduzida difere da raiz entra em DUAS postings. Medido: um
	// documento de 5 tokens que todos reduzem dava DocLength 5 recém-construído
	// e 10 recarregado do cache. DocLength é o divisor da normalização por
	// tamanho do BM25, então o mesmo cofre ranqueava diferente conforme o
	// servidor tivesse acabado de indexar ou de ler o cache — sem nada no log e
	// sem nenhum teste falhando. São 3.152 varints no cofre de referência.
	e.uvarint(uint64(len(docLengths)))
	for path, n := range docLengths {
		e.uvarint(idPorCaminho[path])
		e.uvarint(uint64(n))
	}

	e.uvarint(uint64(len(termos)))
	for termo, docs := range termos {
		e.str(termo)
		e.uvarint(uint64(len(docs)))
		for path, pos := range docs {
			e.uvarint(idPorCaminho[path])
			e.uvarint(uint64(len(pos)))
			// Delta contra o Start anterior. Posições de um mesmo termo numa
			// mesma nota crescem, então o delta é pequeno e cabe em 1-2 bytes
			// no lugar dos 8 de um int64. Guardado como varint COM sinal,
			// porque nada no índice garante ordenação — e um delta negativo
			// lido como positivo corromperia o offset em silêncio.
			var anterior int64
			for _, p := range pos {
				e.varint(p.Start - anterior)
				e.varint(p.End - p.Start)
				anterior = p.Start
			}
		}
	}

	if e.err != nil {
		return e.err
	}
	return e.w.Flush()
}

// --------------------------------------------------------------------------
// Leitura
// --------------------------------------------------------------------------

// leitor decodifica sobre o arquivo INTEIRO em memória, e não sobre um
// io.Reader.
//
// A versão anterior usava bufio e binary.ReadUvarint. Cada byte de cada varint
// passava por uma chamada de interface io.ByteReader para bufio.ReadByte, e o
// arquivo tem ~40 milhões de varints: no perfil isso era 15,03% em ReadUvarint,
// 10,12% em bufio.ReadByte e 5,21% em ReadVarint — perto de 30% do tempo gasto
// em despacho, não em decodificação. Sobre uma fatia, binary.Uvarint lê direto
// da memória.
//
// O custo é ter os 66 MB do arquivo em memória durante a leitura, ao lado dos
// ~290 MB da arena. Como o mapa resultante é várias vezes maior que o arquivo,
// isso não muda o pico de forma relevante.
type leitor struct {
	b   []byte
	i   int
	err error
}

// falha registra o primeiro erro. Os que vierem depois são consequência dele:
// com i parado no ponto do defeito, todo campo seguinte lê lixo, e sobrescrever
// a mensagem trocaria a causa pelo sintoma no log.
func (l *leitor) falha(formato string, args ...any) {
	if l.err == nil {
		l.err = fmt.Errorf(formato, args...)
	}
}

func (l *leitor) uvarint(limite uint64, oque string) uint64 {
	if l.err != nil {
		return 0
	}
	v, n := binary.Uvarint(l.b[l.i:])
	if n <= 0 {
		// n == 0: acabaram os bytes. n < 0: valor maior que 64 bits, o que só
		// acontece com arquivo adulterado ou corrompido.
		l.falha("%w: lendo %s: varint invalido em %d", ErrCacheCorrupted, oque, l.i)
		return 0
	}
	l.i += n
	if v > limite {
		l.falha("%w: %s = %d, acima do limite de %d", ErrCacheCorrupted, oque, v, limite)
		return 0
	}
	return v
}

func (l *leitor) varint() int64 {
	if l.err != nil {
		return 0
	}
	v, n := binary.Varint(l.b[l.i:])
	if n <= 0 {
		l.falha("%w: lendo varint em %d", ErrCacheCorrupted, l.i)
		return 0
	}
	l.i += n
	return v
}

func (l *leitor) str(oque string) string {
	n := l.uvarint(limiteString, oque+" (tamanho)")
	if l.err != nil || n == 0 {
		return ""
	}
	if uint64(len(l.b)-l.i) < n {
		l.falha("%w: lendo %s: faltam bytes (%d pedidos, %d restantes)",
			ErrCacheCorrupted, oque, n, len(l.b)-l.i)
		return ""
	}
	// Cópia deliberada. A conversão string(fatia) copia, e é isso que se quer:
	// apontar para dentro de l.b via unsafe prenderia os 66 MB do arquivo em
	// memória para sempre por causa de ~12 MB de termos.
	s := string(l.b[l.i : l.i+int(n)])
	l.i += int(n)
	return s
}

// leCache desserializa o formato corrente.
//
// Devolve o cabeçalho e os termos. As slices de posição saem todas de UMA
// arena contígua: 17,8 milhões de posições viravam 2,96 milhões de slices
// separadas, e agora são subfatias de um bloco só. Isso apaga a maior parte
// das 12,8 milhões de alocações do formato antigo, e faz a estrutura ser
// percorrida em ordem de memória em vez de saltar pelo heap.
//
// Devolve também `docLengths`, lido do arquivo. Ele NÃO pode ser derivado das
// postings: ver o comentário na escrita.
func leCache(dados []byte) (CacheHeader, map[string]map[string][]TokenPosition, map[string]int, error) {
	l := &leitor{b: dados}

	if len(dados) < len(cacheMagic) {
		return CacheHeader{}, nil, nil, fmt.Errorf("%w: arquivo com %d bytes, menor que a assinatura",
			ErrCacheCorrupted, len(dados))
	}
	l.i = len(cacheMagic)
	if string(dados[:len(cacheMagic)]) != cacheMagic {
		// Cache do formato 1 cai aqui: ele começa com o gob, não com "GBS2".
		// Versão incompatível e não corrupção — a diferença decide se o log
		// assusta alguém à toa.
		return CacheHeader{}, nil, nil, ErrCacheVersionMismatch
	}

	var h CacheHeader
	h.FormatVersion = int(l.uvarint(math.MaxInt32, "formatVersion"))
	h.ParserVersion = int(l.uvarint(math.MaxInt32, "parserVersion"))
	h.AnalyzerVersion = int(l.uvarint(math.MaxInt32, "analyzerVersion"))
	if l.err != nil {
		return CacheHeader{}, nil, nil, l.err
	}
	// Portao de versao ANTES de qualquer campo de layout.
	//
	// A mesma magica pode cobrir layouts diferentes conforme o formato evolui —
	// este ganhou dois totais entre noteCount e a tabela de caminhos. Se o
	// portao ficasse la embaixo, junto com a checagem de parser e analisador em
	// LoadInvertedCache, um arquivo do layout anterior seria decodificado
	// primeiro: os varints casariam com campos trocados e o resultado seria
	// lixo estruturalmente valido, ou um erro de corrupcao que culpa o disco por
	// uma troca de formato. Recusar aqui e a diferenca entre "versao velha" e
	// "arquivo corrompido".
	if h.FormatVersion != cacheCodecVers {
		return CacheHeader{}, nil, nil, ErrCacheVersionMismatch
	}
	h.VaultPath = l.str("vaultPath")
	h.NoteCount = int(l.uvarint(math.MaxInt32, "noteCount"))
	totPost := l.uvarint(limitePostings, "totalPostings")
	totPos := l.uvarint(limitePosicoes, "totalPosicoes")
	if l.err != nil {
		return CacheHeader{}, nil, nil, l.err
	}

	nCaminhos := l.uvarint(limiteCaminhos, "nCaminhos")
	if l.err != nil {
		return h, nil, nil, l.err
	}
	// Uma string por caminho, compartilhada por todas as postings que a citam.
	// No formato antigo cada uma das 2,96 milhões de postings alocava a sua.
	caminhos := make([]string, nCaminhos)
	for i := range caminhos {
		caminhos[i] = l.str("caminho")
	}
	if l.err != nil {
		return h, nil, nil, l.err
	}

	nDocs := l.uvarint(uint64(len(caminhos)), "nDocs")
	if l.err != nil {
		return h, nil, nil, l.err
	}
	docLengths := make(map[string]int, nDocs)
	for i := uint64(0); i < nDocs; i++ {
		pathID := l.uvarint(uint64(len(caminhos))-1, "docPathID")
		n := l.uvarint(math.MaxInt32, "docLength")
		if l.err != nil {
			return h, nil, nil, l.err
		}
		docLengths[caminhos[pathID]] = int(n)
	}

	nTermos := l.uvarint(limiteTermos, "nTermos")
	if l.err != nil {
		return h, nil, nil, l.err
	}

	// Dimensionado exato: sem isto o mapa cresce por realocação e reinsere
	// tudo a cada dobra.
	termos := make(map[string]map[string][]TokenPosition, nTermos)

	// Arena. Cresce por append; o tamanho final não é conhecido de antemão
	// sem um segundo passe pelo arquivo.
	arena := make([]TokenPosition, 0, totPos)

	// As subfatias precisam do arena estável, e append pode realocá-lo. Por
	// isso guardamos ÍNDICES aqui e só recortamos as fatias no fim.
	type faixa struct {
		termo  string
		path   string
		ini    int
		quantt int
	}
	faixas := make([]faixa, 0, totPost)

	for i := uint64(0); i < nTermos; i++ {
		termo := l.str("termo")
		nPost := l.uvarint(limitePostings, "nPostings")
		if l.err != nil {
			return h, nil, nil, l.err
		}
		docs := make(map[string][]TokenPosition, nPost)
		termos[termo] = docs

		for j := uint64(0); j < nPost; j++ {
			pathID := l.uvarint(uint64(len(caminhos)), "pathID")
			nPos := l.uvarint(limitePosicoes, "nPosicoes")
			if l.err != nil {
				return h, nil, nil, l.err
			}
			if pathID >= uint64(len(caminhos)) {
				return h, nil, nil, fmt.Errorf("%w: pathID %d fora da tabela de %d caminhos",
					ErrCacheCorrupted, pathID, len(caminhos))
			}
			ini := len(arena)
			var anterior int64
			for k := uint64(0); k < nPos; k++ {
				dStart := l.varint()
				dLen := l.varint()
				if l.err != nil {
					return h, nil, nil, l.err
				}
				start := anterior + dStart
				arena = append(arena, TokenPosition{Start: start, End: start + dLen})
				anterior = start
			}
			faixas = append(faixas, faixa{termo: termo, path: caminhos[pathID], ini: ini, quantt: int(nPos)})
		}
	}
	if l.err != nil {
		return h, nil, nil, l.err
	}

	for _, f := range faixas {
		// Capacidade travada em `ini+quant`: a fatia é uma janela dentro da
		// arena, e um append feito pelo watcher depois do boot escreveria por
		// cima da posting VIZINHA se sobrasse capacidade. Com cap == len o
		// append copia para fora e ninguém se contamina. Sem isto, editar uma
		// nota corromperia os offsets de outra, em silêncio.
		termos[f.termo][f.path] = arena[f.ini : f.ini+f.quantt : f.ini+f.quantt]
	}

	return h, termos, docLengths, nil
}
