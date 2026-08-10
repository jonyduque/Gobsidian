package search

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sort"
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
	cacheMagic     = "GBS6"
	cacheCodecVers = 6
)

// footerBytes é o tamanho do rodapé fixo escrito no fim do arquivo (Task 89):
// 8 bytes de assinatura, 8 bytes de offset da seção fixa de posições, 8 bytes
// de contagem de posições — todos little-endian, sem varint. Fixo e no fim
// de propósito: quem só quer mapear a arena (ver mmap.go) acha o rodapé sem
// decodificar o corpo inteiro em varint.
const footerBytes = 24

// footerMagic identifica o rodapé da seção fixa. Um cache do formato anterior
// (sem rodapé) quase certamente não termina com estes 8 bytes por acaso, e o
// leitor trata a ausência da assinatura como "sem arena para mapear" — nunca
// como corrupção; a decodificação integral cobre o arquivo de qualquer jeito.
var footerMagic = [8]byte{'G', 'B', 'S', 'A', 'R', 'E', 'N', 'A'}

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
	// n conta bytes efetivamente escritos, para a seção fixa (Task 89) saber
	// o próprio offset dentro do arquivo sem que bufio exponha isso.
	n   int64
	err error
}

// raw escreve bytes crus e soma em n. Todo outro método do escritor passa por
// aqui, para nunca haver um caminho de escrita que n não conte.
func (e *escritor) raw(b []byte) {
	if e.err != nil {
		return
	}
	nn, err := e.w.Write(b)
	e.n += int64(nn)
	if err != nil {
		e.err = err
	}
}

func (e *escritor) uvarint(v uint64) {
	if e.err != nil {
		return
	}
	n := binary.PutUvarint(e.buf[:], v)
	e.raw(e.buf[:n])
}

func (e *escritor) varint(v int64) {
	if e.err != nil {
		return
	}
	n := binary.PutVarint(e.buf[:], v)
	e.raw(e.buf[:n])
}

func (e *escritor) str(s string) {
	e.uvarint(uint64(len(s)))
	if e.err != nil {
		return
	}
	nn, err := e.w.WriteString(s)
	e.n += int64(nn)
	if err != nil {
		e.err = err
	}
}

// escreveCache serializa o índice no formato corrente.
//
// Recebe o mapa já exportado para não segurar o lock do índice durante a
// escrita em disco, que dura segundos.
//
// Grava TUDO em ordem: caminhos crescente, termos crescente, e as postings de
// cada termo por pathID crescente. A ordem é o contrato que permite ao leitor
// montar arrays achatados com busca binária, em vez de um mapa por termo (ver
// soa.go). Ordenar aqui custa ~126 mil comparações de string mais 3 milhões de
// comparações de int32, pagas numa gravação que já leva segundos; não ordenar
// custaria uma ordenação a cada partida.
func escreveCache(
	w io.Writer,
	h CacheHeader,
	termos map[string]map[string][]TokenPosition,
	docLengths map[string]int,
) error {
	e := &escritor{w: bufio.NewWriterSize(w, 1<<20)}

	e.raw([]byte(cacheMagic))
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
	vistos := make(map[string]struct{}, len(docLengths))
	caminhos := make([]string, 0, len(docLengths))
	registra := func(path string) {
		if _, ok := vistos[path]; ok {
			return
		}
		vistos[path] = struct{}{}
		caminhos = append(caminhos, path)
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

	// Seção fixa de posições (Task 89): mesmo conteúdo do delta-varint acima,
	// em bytes crus de 16 (Start int64 + End int64, little-endian), na MESMA
	// ordem — preenchida dentro do mesmo laço que grava o varint, para as duas
	// ordens não poderem divergir. Fica no fim do arquivo (ver mais abaixo) e
	// existe para ser mapeada em modo leitura: um array varint com delta não é
	// mapeável direto, mas quem só quer economia de disco nem olha para cá — o
	// varint acima continua sendo o que a decodificação integral usa.
	fixedSec := make([]byte, totPos*16)
	var fixedCursor int

	// O pathID passa a ser a POSIÇÃO no vetor ordenado, e não a ordem de
	// descoberta. É o que faz "postings ordenadas por pathID" e "resultado
	// ordenado por caminho" serem a mesma coisa, e o que deixa o leitor
	// devolver Postings já ordenado sem ordenar nada.
	sort.Strings(caminhos)
	idPorCaminho := make(map[string]uint64, len(caminhos))
	for i, c := range caminhos {
		idPorCaminho[c] = uint64(i)
	}

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

	termosOrd := make([]string, 0, len(termos))
	for termo := range termos {
		termosOrd = append(termosOrd, termo)
	}
	sort.Strings(termosOrd)

	e.uvarint(uint64(len(termos)))
	ids := make([]uint64, 0, 64)
	for _, termo := range termosOrd {
		docs := termos[termo]
		e.str(termo)
		e.uvarint(uint64(len(docs)))

		ids = ids[:0]
		for path := range docs {
			ids = append(ids, idPorCaminho[path])
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

		for _, id := range ids {
			pos := docs[caminhos[id]]
			e.uvarint(id)
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

				binary.LittleEndian.PutUint64(fixedSec[fixedCursor:], uint64(p.Start))
				binary.LittleEndian.PutUint64(fixedSec[fixedCursor+8:], uint64(p.End))
				fixedCursor += 16
			}
		}
	}

	// Alinhamento de 8 bytes ANTES da seção fixa. A leitura mapeada
	// reinterpreta esses bytes como []TokenPosition via unsafe.Slice, e isso
	// exige um ponteiro alinhado em 8 bytes: a base do mapeamento já vem
	// alinhada por página do sistema operacional, então basta o OFFSET dentro
	// do arquivo também ser múltiplo de 8.
	for e.n%8 != 0 {
		e.raw([]byte{0})
	}
	posArrayOffset := e.n
	e.raw(fixedSec)

	// Rodapé fixo, sempre nos ÚLTIMOS 24 bytes do arquivo — ver footerBytes.
	var rodape [footerBytes]byte
	copy(rodape[:8], footerMagic[:])
	binary.LittleEndian.PutUint64(rodape[8:16], uint64(posArrayOffset))
	binary.LittleEndian.PutUint64(rodape[16:24], totPos)
	e.raw(rodape[:])

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

// leCache desserializa o formato corrente direto para arrays achatados.
//
// Nenhum mapa por termo é construído. O arquivo declara os totais, então cada
// array sai de UMA alocação dimensionada e é preenchido sequencialmente: seis
// alocações grandes no lugar de 126 mil mapas e 3 milhões de entradas. Era o
// que sobrava do tempo de carga depois de o formato já ter sido resolvido —
// 35% em `aeshashbody`, `mapassign_faststr` e `matchH2`.
//
// A estrutura devolvida e o que ela garante estão em soa.go.
func leCache(dados []byte) (CacheHeader, *baseSoA, error) {
	return decodificaCache(dados, nil)
}

// leCacheComArena decodifica como leCache, mas usa `arena` — vinda de um
// arquivo mapeado (ver mmap.go) — como o array de posições em vez de alocar e
// preencher um novo a partir do varint. O corpo do varint ainda é percorrido
// byte a byte, e não pulado: é o que continua detectando um arquivo
// corrompido no meio das posições mesmo quando a arena está sendo usada.
//
// Quem chama garante len(arena) == totPos declarado no cabeçalho antes de
// chamar (ver tentaAbrirArena); decodificaCache confere de novo aqui, porque
// um arquivo cujo rodapé mente sobre a contagem não pode produzir uma base
// cujos limites não batem com o que ela própria despachou.
func leCacheComArena(dados []byte, arena []TokenPosition) (CacheHeader, *baseSoA, error) {
	return decodificaCache(dados, arena)
}

// decodificaCache é o corpo compartilhado por leCache e leCacheComArena.
//
// `arena`, quando não nil, substitui a alocação de b.pos: o array de posições
// vem de fora (mapeado do arquivo) em vez de heap comum, e o laço abaixo pula
// só a ESCRITA em b.pos — continua decodificando cada delta varint, para o
// corpo do arquivo continuar sendo a fonte de verdade sobre corrupção.
func decodificaCache(dados []byte, arena []TokenPosition) (CacheHeader, *baseSoA, error) {
	l := &leitor{b: dados}

	if len(dados) < len(cacheMagic) {
		return CacheHeader{}, nil, fmt.Errorf("%w: arquivo com %d bytes, menor que a assinatura",
			ErrCacheCorrupted, len(dados))
	}
	l.i = len(cacheMagic)
	if string(dados[:len(cacheMagic)]) != cacheMagic {
		// Cache de formato anterior cai aqui: a assinatura carrega a versão.
		// Versão incompatível e não corrupção — a diferença decide se o log
		// assusta alguém à toa.
		return CacheHeader{}, nil, ErrCacheVersionMismatch
	}

	var h CacheHeader
	h.FormatVersion = int(l.uvarint(math.MaxInt32, "formatVersion"))
	h.ParserVersion = int(l.uvarint(math.MaxInt32, "parserVersion"))
	h.AnalyzerVersion = int(l.uvarint(math.MaxInt32, "analyzerVersion"))
	if l.err != nil {
		return CacheHeader{}, nil, l.err
	}
	// Portao de versao ANTES de qualquer campo de layout.
	//
	// A mesma magica pode cobrir layouts diferentes conforme o formato evolui.
	// Se o portao ficasse la embaixo, junto com a checagem de parser e
	// analisador em LoadInvertedCache, um arquivo do layout anterior seria
	// decodificado primeiro: os varints casariam com campos trocados e o
	// resultado seria lixo estruturalmente valido, ou um erro de corrupcao que
	// culpa o disco por uma troca de formato.
	if h.FormatVersion != cacheCodecVers {
		return CacheHeader{}, nil, ErrCacheVersionMismatch
	}
	h.VaultPath = l.str("vaultPath")
	h.NoteCount = int(l.uvarint(math.MaxInt32, "noteCount"))
	totPost := l.uvarint(limitePostings, "totalPostings")
	totPos := l.uvarint(limitePosicoes, "totalPosicoes")
	if l.err != nil {
		return CacheHeader{}, nil, l.err
	}

	nCaminhos := l.uvarint(limiteCaminhos, "nCaminhos")
	if l.err != nil {
		return h, nil, l.err
	}
	b := &baseSoA{
		caminhos: make([]string, nCaminhos),
		docLen:   make([]int32, nCaminhos),
	}
	for i := range b.caminhos {
		b.caminhos[i] = l.str("caminho")
	}
	if l.err != nil {
		return h, nil, l.err
	}

	nDocs := l.uvarint(nCaminhos, "nDocs")
	if l.err != nil {
		return h, nil, l.err
	}
	for i := uint64(0); i < nDocs; i++ {
		pathID := l.uvarint(nCaminhos, "docPathID")
		n := l.uvarint(math.MaxInt32, "docLength")
		if l.err != nil {
			return h, nil, l.err
		}
		if pathID >= nCaminhos {
			return h, nil, fmt.Errorf("%w: docPathID %d fora da tabela de %d caminhos",
				ErrCacheCorrupted, pathID, nCaminhos)
		}
		b.docLen[pathID] = int32(n)
	}

	nTermos := l.uvarint(limiteTermos, "nTermos")
	if l.err != nil {
		return h, nil, l.err
	}

	// Todos dimensionados pelos totais do arquivo. termIni e postIni têm um
	// elemento a mais porque guardam FRONTEIRAS: a faixa do item i é
	// [ini[i], ini[i+1]), e o sentinela final evita um caso especial no último.
	b.termos = make([]string, nTermos)
	b.termIni = make([]int32, nTermos+1)
	b.postPath = make([]int32, totPost)
	b.postIni = make([]int32, totPost+1)
	if arena != nil {
		if uint64(len(arena)) != totPos {
			return h, nil, fmt.Errorf("%w: arena mapeada tem %d posicoes, cabecalho declara %d",
				ErrCacheCorrupted, len(arena), totPos)
		}
		b.pos = arena
	} else {
		b.pos = make([]TokenPosition, totPos)
	}

	var jPost, kPos int64
	for i := uint64(0); i < nTermos; i++ {
		b.termos[i] = l.str("termo")
		b.termIni[i] = int32(jPost)
		nPost := l.uvarint(limitePostings, "nPostings")
		if l.err != nil {
			return h, nil, l.err
		}
		if jPost+int64(nPost) > int64(totPost) {
			return h, nil, fmt.Errorf("%w: postings alem do total declarado de %d", ErrCacheCorrupted, totPost)
		}

		for j := uint64(0); j < nPost; j++ {
			pathID := l.uvarint(nCaminhos, "pathID")
			nPos := l.uvarint(limitePosicoes, "nPosicoes")
			if l.err != nil {
				return h, nil, l.err
			}
			if pathID >= nCaminhos {
				return h, nil, fmt.Errorf("%w: pathID %d fora da tabela de %d caminhos",
					ErrCacheCorrupted, pathID, nCaminhos)
			}
			if kPos+int64(nPos) > int64(totPos) {
				return h, nil, fmt.Errorf("%w: posicoes alem do total declarado de %d", ErrCacheCorrupted, totPos)
			}

			b.postPath[jPost] = int32(pathID)
			b.postIni[jPost] = int32(kPos)
			jPost++

			var anterior int64
			for k := uint64(0); k < nPos; k++ {
				dStart := l.varint()
				dLen := l.varint()
				if l.err != nil {
					return h, nil, l.err
				}
				start := anterior + dStart
				if arena == nil {
					b.pos[kPos] = TokenPosition{Start: start, End: start + dLen}
				}
				kPos++
				anterior = start
			}
		}
	}
	if l.err != nil {
		return h, nil, l.err
	}
	b.termIni[nTermos] = int32(jPost)
	b.postIni[jPost] = int32(kPos)

	// Os totais declarados no cabeçalho e o que o corpo entregou têm de bater.
	// Se o corpo trouxer MENOS, as caudas dos arrays ficam zeradas e viram
	// postings do caminho 0 com zero posições — dado inventado com aparência
	// legítima. É o mesmo motivo de o formato declarar os totais.
	if jPost != int64(totPost) || kPos != int64(totPos) {
		return h, nil, fmt.Errorf("%w: corpo trouxe %d postings e %d posicoes, cabecalho declarou %d e %d",
			ErrCacheCorrupted, jPost, kPos, totPost, totPos)
	}

	if err := b.validaOrdem(); err != nil {
		return h, nil, err
	}
	b.montaIDs()
	b.montaIndiceDireto()

	return h, b, nil
}
