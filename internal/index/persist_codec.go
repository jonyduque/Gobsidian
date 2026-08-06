package index

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"sort"
	"time"

	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Codec binário do cache do índice de metadados.
//
// Mesma técnica do cache de busca (internal/search/persist_codec.go): varint
// para inteiro, string sempre como tamanho+bytes, cabeçalho com portão de
// versão ANTES de qualquer campo de layout. Não é o MESMO código — os dois
// pacotes não se importam um ao outro (search importa index, e não o
// contrário; ver CLAUDE.md, "nenhum tipo do SDK cruza..." é o mesmo
// raciocínio aplicado a essa fronteira), e o layout aqui é outro: uma nota
// carrega frontmatter arbitrário (YAML decodificado, tipos variados),
// headings, blocos e links, que uma posting de termo não tem. É a mesma
// REGRA reaplicada — não inventar um terceiro jeito de serializar dentro do
// mesmo projeto — não uma cópia de arquivo.
//
// Divergência deliberada do cache de busca: aqui o corpo carrega um CRC32
// (IEEE) como rodapé. O cache de busca não tem checksum e confia nos totais
// declarados e nos limites por campo para pegar corrupção — suficiente
// quando o pior caso é uma busca com menos resultados. Aqui o pior caso é
// outro: "um cache de metadados errado serve nota errada, não nota lenta"
// (brief da Task 85). Um byte trocado no meio do arquivo, se cair dentro do
// PAYLOAD de uma string em vez de um comprimento, decodifica sem erro
// estrutural nenhum — os limites e os totais não veem nada de errado, e o
// servidor serviria um título ou um caminho corrompido com confiança. O
// checksum fecha essa lacuna: cobre o arquivo inteiro, então qualquer
// corrupção de um byte — em comprimento OU em payload — é pega antes de
// qualquer campo ser interpretado.
const (
	indexCacheMagic     = "GIC1"
	indexCacheCodecVers = 1
)

// Tags do codec de valor genérico de frontmatter. Frontmatter vem do YAML
// decodificado por yaml.v3 em map[string]any, e em Go 64 bits os tipos que
// ele produz são: nil, bool, int, uint64 (só para inteiro que estoura
// int64), float64, string, time.Time, []any e map[string]any — nunca int64
// isolado, porque yaml.v3 só devolve int64 quando o valor NÃO cabe em `int`,
// e em Go 64 bits `int` já tem 64 bits. A tag para int64 existe mesmo assim,
// por precaução: decodificar de volta como `int` um valor que era `int64`
// trocaria o tipo dinâmico do valor, e query.go faz asserção de tipo
// (`.(int)`) sobre isto — um Frontmatter recarregado do cache com o tipo
// errado faria note_list filtrar por número de forma silenciosamente
// diferente do índice recém-construído.
//
// nil vs slice/map vazio é distinguido por tags separadas — não por um
// comprimento -1, como o resto do codec faz para []string. any/map[string]any
// já carregam a própria tag de tipo, então usar duas tags aqui é mais barato
// que reservar um valor sentinela dentro do espaço de comprimento.
const (
	valNil byte = iota
	valBool
	valInt
	valInt64
	valUint64
	valFloat64
	valString
	valTime
	valSliceNil
	valSlice
	valMapNil
	valMap
)

// Limites contra alocação gigante a partir de um arquivo corrompido — mesmo
// raciocínio do cache de busca (limiteRazoavel em internal/search). Um
// cofre real tem milhares de notas, não milhões; estes tetos deixam ordens
// de grandeza de folga e continuam finitos.
const (
	limiteNotas        = 10_000_000
	limiteSliceLen     = 1_000_000
	limiteMapLen       = 1_000_000
	limiteString       = 1 << 20
	limiteTimeBlob     = 64
	limiteValorProfund = 64
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

func (e *escritor) fixed64(bits uint64) {
	if e.err != nil {
		return
	}
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], bits)
	_, e.err = e.w.Write(buf[:])
}

func (e *escritor) str(s string) {
	e.uvarint(uint64(len(s)))
	if e.err != nil {
		return
	}
	_, e.err = e.w.WriteString(s)
}

func (e *escritor) boolean(b bool) {
	if b {
		e.uvarint(1)
	} else {
		e.uvarint(0)
	}
}

// timeBlob grava time.Time via MarshalBinary — wall clock, fuso e precisão,
// exatos. Um par (offset em nanossegundos, UTC assumido) perderia o fuso de
// um horário com deslocamento explícito no frontmatter (yaml.v3 devolve
// time.Time com Location != UTC quando a string tem "+05:00" etc.), e a
// comparação por reflect.DeepEqual do teste diferencial pegaria isso.
func (e *escritor) timeBlob(t time.Time) {
	if e.err != nil {
		return
	}
	b, err := t.MarshalBinary()
	if err != nil {
		e.err = err
		return
	}
	e.uvarint(uint64(len(b)))
	if e.err != nil {
		return
	}
	_, e.err = e.w.Write(b)
}

// strSlice grava um []string com sentinela -1 para nil, distinto de
// comprimento 0. reflect.DeepEqual distingue os dois, e Tags/Aliases de uma
// nota sem nenhum são nil, não fatia vazia.
func (e *escritor) strSlice(ss []string) {
	if ss == nil {
		e.varint(-1)
		return
	}
	e.varint(int64(len(ss)))
	for _, s := range ss {
		e.str(s)
	}
}

func (e *escritor) headings(hs []parser.Heading) {
	if hs == nil {
		e.varint(-1)
		return
	}
	e.varint(int64(len(hs)))
	for _, h := range hs {
		e.varint(int64(h.Level))
		e.str(h.Text)
		e.str(h.Slug)
		e.varint(h.Start)
		e.varint(h.End)
		e.varint(h.BodyStart)
	}
}

func (e *escritor) blocks(bs []parser.Block) {
	if bs == nil {
		e.varint(-1)
		return
	}
	e.varint(int64(len(bs)))
	for _, b := range bs {
		e.str(b.ID)
		e.varint(b.Start)
		e.varint(b.End)
	}
}

// links grava só os campos de parser.Link. Resolved, Via e State NÃO são
// gravados — ver o comentário de LoadIndexCache: são recalculados depois do
// load pelas MESMAS funções que Build chama (resolveAllLinks,
// buildBacklinks), e persistir os dois seria uma segunda forma de calcular
// o mesmo dado. A lição deste projeto, escrita em CLAUDE.md, é que a forma
// menos usada é a que diverge.
func (e *escritor) links(ls []ResolvedLink) {
	if ls == nil {
		e.varint(-1)
		return
	}
	e.varint(int64(len(ls)))
	for _, rl := range ls {
		e.str(rl.Raw)
		e.str(rl.Target)
		e.str(rl.Alias)
		e.str(rl.Anchor)
		e.varint(int64(rl.Kind))
		e.varint(rl.Start)
		e.varint(rl.End)
	}
}

func (e *escritor) inline(m map[string][]string) {
	if m == nil {
		e.varint(-1)
		return
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	e.varint(int64(len(keys)))
	for _, k := range keys {
		e.str(k)
		e.strSlice(m[k])
	}
}

// value grava um valor de frontmatter — ver o comentário das tags val* para
// o conjunto de tipos e por que nil/vazio de slice e mapa levam tags
// separadas.
func (e *escritor) value(v any) {
	if e.err != nil {
		return
	}
	switch x := v.(type) {
	case nil:
		e.uvarint(uint64(valNil))
	case bool:
		e.uvarint(uint64(valBool))
		e.boolean(x)
	case int:
		e.uvarint(uint64(valInt))
		e.varint(int64(x))
	case int64:
		e.uvarint(uint64(valInt64))
		e.varint(x)
	case uint64:
		e.uvarint(uint64(valUint64))
		e.uvarint(x)
	case float64:
		e.uvarint(uint64(valFloat64))
		e.fixed64(math.Float64bits(x))
	case string:
		e.uvarint(uint64(valString))
		e.str(x)
	case time.Time:
		e.uvarint(uint64(valTime))
		e.timeBlob(x)
	case []any:
		if x == nil {
			e.uvarint(uint64(valSliceNil))
			return
		}
		e.uvarint(uint64(valSlice))
		e.uvarint(uint64(len(x)))
		for _, item := range x {
			e.value(item)
		}
	case map[string]any:
		if x == nil {
			e.uvarint(uint64(valMapNil))
			return
		}
		e.uvarint(uint64(valMap))
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		e.uvarint(uint64(len(keys)))
		for _, k := range keys {
			e.str(k)
			e.value(x[k])
		}
	default:
		// Não deveria acontecer: Frontmatter só é populado por
		// parser.DecodeFrontmatter, que decodifica YAML em map[string]any e
		// só produz os tipos acima. Falhar alto em vez de perder dado em
		// silêncio se o parser algum dia mudar o que produz.
		e.err = fmt.Errorf("valor de frontmatter com tipo nao suportado: %T", v)
	}
}

func (e *escritor) note(n *Note) {
	e.str(string(n.Path))
	e.str(n.Title)
	e.uvarint(uint64(n.Size))
	e.timeBlob(n.ModTime)
	e.uvarint(n.Hash)
	e.uvarint(uint64(n.EOL))
	e.boolean(n.BOM)
	e.boolean(n.CloudOnly)
	e.value(n.Frontmatter)
	e.strSlice(n.Tags)
	e.strSlice(n.Aliases)
	e.headings(n.Headings)
	e.blocks(n.Blocks)
	e.links(n.Links)
	e.inline(n.Inline)
}

func (e *escritor) asset(a *Asset) {
	e.str(string(a.Path))
	e.uvarint(uint64(a.Size))
	e.timeBlob(a.ModTime)
}

// escreveIndexCache serializa o índice no formato corrente, com um CRC32
// como rodapé (ver o comentário no topo do arquivo).
//
// Grava em ordem de caminho crescente — não é exigido por nenhum leitor (ao
// contrário do cache de busca, aqui não há busca binária sobre arrays
// achatados), mas torna o arquivo determinístico entre gravações do mesmo
// índice, o que ajuda quem for revisar um diff ou reproduzir um bug.
func escreveIndexCache(w io.Writer, h CacheHeader, notes []*Note, assets []*Asset) error {
	hasher := crc32.NewIEEE()
	mw := io.MultiWriter(w, hasher)
	e := &escritor{w: bufio.NewWriterSize(mw, 1<<20)}

	if _, err := e.w.WriteString(indexCacheMagic); err != nil {
		return err
	}
	e.uvarint(uint64(h.FormatVersion))
	e.uvarint(uint64(h.ParserVersion))
	e.str(h.VaultPath)
	e.uvarint(uint64(h.NoteCount))
	e.uvarint(uint64(h.AssetCount))

	sort.Slice(notes, func(i, j int) bool { return notes[i].Path < notes[j].Path })
	e.uvarint(uint64(len(notes)))
	for _, n := range notes {
		e.note(n)
	}

	sort.Slice(assets, func(i, j int) bool { return assets[i].Path < assets[j].Path })
	e.uvarint(uint64(len(assets)))
	for _, a := range assets {
		e.asset(a)
	}

	if e.err != nil {
		return e.err
	}
	if err := e.w.Flush(); err != nil {
		return err
	}

	var sumBuf [4]byte
	binary.BigEndian.PutUint32(sumBuf[:], hasher.Sum32())
	_, err := w.Write(sumBuf[:])
	return err
}

// --------------------------------------------------------------------------
// Leitura
// --------------------------------------------------------------------------

type leitor struct {
	b   []byte
	i   int
	err error
}

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
		l.falha("%w: lendo %s: varint invalido em %d", ErrIndexCacheCorrupted, oque, l.i)
		return 0
	}
	l.i += n
	if v > limite {
		l.falha("%w: %s = %d, acima do limite de %d", ErrIndexCacheCorrupted, oque, v, limite)
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
		l.falha("%w: lendo varint em %d", ErrIndexCacheCorrupted, l.i)
		return 0
	}
	l.i += n
	return v
}

func (l *leitor) fixed64() uint64 {
	if l.err != nil {
		return 0
	}
	if len(l.b)-l.i < 8 {
		l.falha("%w: lendo float64: faltam bytes", ErrIndexCacheCorrupted)
		return 0
	}
	v := binary.BigEndian.Uint64(l.b[l.i : l.i+8])
	l.i += 8
	return v
}

func (l *leitor) str(oque string) string {
	n := l.uvarint(limiteString, oque+" (tamanho)")
	if l.err != nil || n == 0 {
		return ""
	}
	if uint64(len(l.b)-l.i) < n {
		l.falha("%w: lendo %s: faltam bytes (%d pedidos, %d restantes)",
			ErrIndexCacheCorrupted, oque, n, len(l.b)-l.i)
		return ""
	}
	s := string(l.b[l.i : l.i+int(n)])
	l.i += int(n)
	return s
}

func (l *leitor) boolean(oque string) bool {
	return l.uvarint(1, oque) != 0
}

func (l *leitor) timeBlob(oque string) time.Time {
	n := l.uvarint(limiteTimeBlob, oque+" (tamanho)")
	if l.err != nil {
		return time.Time{}
	}
	if uint64(len(l.b)-l.i) < n {
		l.falha("%w: lendo %s: faltam bytes", ErrIndexCacheCorrupted, oque)
		return time.Time{}
	}
	b := l.b[l.i : l.i+int(n)]
	l.i += int(n)
	var t time.Time
	if err := t.UnmarshalBinary(b); err != nil {
		l.falha("%w: %s invalido: %v", ErrIndexCacheCorrupted, oque, err)
		return time.Time{}
	}
	return t
}

func (l *leitor) strSlice(oque string) []string {
	n := l.varint()
	if l.err != nil {
		return nil
	}
	if n < 0 {
		return nil
	}
	if uint64(n) > limiteSliceLen {
		l.falha("%w: %s tem %d elementos, acima do limite", ErrIndexCacheCorrupted, oque, n)
		return nil
	}
	out := make([]string, n)
	for i := range out {
		out[i] = l.str(oque + " item")
	}
	return out
}

func (l *leitor) headings(oque string) []parser.Heading {
	n := l.varint()
	if l.err != nil {
		return nil
	}
	if n < 0 {
		return nil
	}
	if uint64(n) > limiteSliceLen {
		l.falha("%w: %s tem %d elementos, acima do limite", ErrIndexCacheCorrupted, oque, n)
		return nil
	}
	out := make([]parser.Heading, n)
	for i := range out {
		out[i] = parser.Heading{
			Level:     int(l.varint()),
			Text:      l.str(oque + " text"),
			Slug:      l.str(oque + " slug"),
			Start:     l.varint(),
			End:       l.varint(),
			BodyStart: l.varint(),
		}
	}
	return out
}

func (l *leitor) blocks(oque string) []parser.Block {
	n := l.varint()
	if l.err != nil {
		return nil
	}
	if n < 0 {
		return nil
	}
	if uint64(n) > limiteSliceLen {
		l.falha("%w: %s tem %d elementos, acima do limite", ErrIndexCacheCorrupted, oque, n)
		return nil
	}
	out := make([]parser.Block, n)
	for i := range out {
		out[i] = parser.Block{
			ID:    l.str(oque + " id"),
			Start: l.varint(),
			End:   l.varint(),
		}
	}
	return out
}

func (l *leitor) links(oque string) []ResolvedLink {
	n := l.varint()
	if l.err != nil {
		return nil
	}
	if n < 0 {
		return nil
	}
	if uint64(n) > limiteSliceLen {
		l.falha("%w: %s tem %d elementos, acima do limite", ErrIndexCacheCorrupted, oque, n)
		return nil
	}
	out := make([]ResolvedLink, n)
	for i := range out {
		out[i] = ResolvedLink{
			Link: parser.Link{
				Raw:    l.str(oque + " raw"),
				Target: l.str(oque + " target"),
				Alias:  l.str(oque + " alias"),
				Anchor: l.str(oque + " anchor"),
				Kind:   parser.LinkKind(l.varint()),
				Start:  l.varint(),
				End:    l.varint(),
			},
			// Resolved, Via e State ficam no zero-value de propósito — ver o
			// comentário de escritor.links.
		}
	}
	return out
}

func (l *leitor) inline(oque string) map[string][]string {
	n := l.varint()
	if l.err != nil {
		return nil
	}
	if n < 0 {
		return nil
	}
	if uint64(n) > limiteMapLen {
		l.falha("%w: %s tem %d chaves, acima do limite", ErrIndexCacheCorrupted, oque, n)
		return nil
	}
	out := make(map[string][]string, n)
	for i := int64(0); i < n; i++ {
		k := l.str(oque + " chave")
		out[k] = l.strSlice(oque + " valores")
	}
	return out
}

func (l *leitor) value(profundidade int) any {
	if l.err != nil {
		return nil
	}
	if profundidade > limiteValorProfund {
		l.falha("%w: frontmatter aninhado alem do limite de profundidade", ErrIndexCacheCorrupted)
		return nil
	}
	tag := l.uvarint(uint64(valMap), "tipo de valor")
	if l.err != nil {
		return nil
	}
	switch byte(tag) {
	case valNil:
		return nil
	case valBool:
		return l.boolean("valor bool")
	case valInt:
		return int(l.varint())
	case valInt64:
		return l.varint()
	case valUint64:
		return l.uvarint(math.MaxUint64, "valor uint64")
	case valFloat64:
		return math.Float64frombits(l.fixed64())
	case valString:
		return l.str("valor string")
	case valTime:
		return l.timeBlob("valor time")
	case valSliceNil:
		return []any(nil)
	case valSlice:
		n := l.uvarint(limiteSliceLen, "tamanho de lista")
		if l.err != nil {
			return nil
		}
		out := make([]any, n)
		for i := range out {
			out[i] = l.value(profundidade + 1)
		}
		return out
	case valMapNil:
		return map[string]any(nil)
	case valMap:
		n := l.uvarint(limiteMapLen, "tamanho de mapa")
		if l.err != nil {
			return nil
		}
		out := make(map[string]any, n)
		for i := uint64(0); i < n; i++ {
			k := l.str("chave de frontmatter")
			out[k] = l.value(profundidade + 1)
		}
		return out
	default:
		l.falha("%w: tag de valor desconhecida %d", ErrIndexCacheCorrupted, tag)
		return nil
	}
}

func (l *leitor) note() *Note {
	path := l.str("note path")
	title := l.str("note title")
	size := int64(l.uvarint(math.MaxInt64, "note size"))
	modTime := l.timeBlob("note modTime")
	hash := l.uvarint(math.MaxUint64, "note hash")
	eol := vault.EOLStyle(l.uvarint(1, "note eol"))
	bom := l.boolean("note bom")
	cloudOnly := l.boolean("note cloudOnly")

	fmVal := l.value(0)
	fm, ok := fmVal.(map[string]any)
	if l.err == nil && !ok {
		l.falha("%w: frontmatter da nota %q nao decodificou como mapa (tipo %T)",
			ErrIndexCacheCorrupted, path, fmVal)
	}

	tags := l.strSlice("note tags")
	aliases := l.strSlice("note aliases")
	headings := l.headings("note headings")
	blocks := l.blocks("note blocks")
	links := l.links("note links")
	inline := l.inline("note inline")

	if l.err != nil {
		return nil
	}

	return &Note{
		Path:        vault.CanonicalPath(path),
		Title:       title,
		TitleNorm:   normalizeTitleForNote(title),
		Size:        size,
		ModTime:     modTime,
		Hash:        hash,
		EOL:         eol,
		BOM:         bom,
		CloudOnly:   cloudOnly,
		Frontmatter: fm,
		Tags:        tags,
		Aliases:     aliases,
		Headings:    headings,
		Blocks:      blocks,
		Links:       links,
		Inline:      inline,
	}
}

func (l *leitor) asset() *Asset {
	path := l.str("asset path")
	size := int64(l.uvarint(math.MaxInt64, "asset size"))
	modTime := l.timeBlob("asset modTime")
	if l.err != nil {
		return nil
	}
	return &Asset{
		Path:    vault.CanonicalPath(path),
		Size:    size,
		ModTime: modTime,
	}
}

// leIndexCache desserializa o formato corrente.
//
// Confere o CRC32 do rodapé ANTES de decodificar qualquer campo — ver o
// comentário no topo do arquivo sobre por que este cache não pode se
// contentar com os limites por campo que bastam para o de busca.
func leIndexCache(dados []byte) (CacheHeader, []*Note, []*Asset, error) {
	if len(dados) < len(indexCacheMagic)+4 {
		return CacheHeader{}, nil, nil, fmt.Errorf("%w: arquivo com %d bytes, menor que assinatura+checksum",
			ErrIndexCacheCorrupted, len(dados))
	}

	corpo := dados[:len(dados)-4]
	somaGravada := binary.BigEndian.Uint32(dados[len(dados)-4:])
	somaCalculada := crc32.ChecksumIEEE(corpo)
	if somaCalculada != somaGravada {
		return CacheHeader{}, nil, nil, fmt.Errorf("%w: checksum nao bate (gravado %08x, calculado %08x)",
			ErrIndexCacheCorrupted, somaGravada, somaCalculada)
	}

	l := &leitor{b: corpo}

	if string(corpo[:len(indexCacheMagic)]) != indexCacheMagic {
		// Cache de formato anterior ou arquivo de outra origem cai aqui: a
		// assinatura carrega a versão. Versão incompatível não é corrupção —
		// a distinção decide se o log assusta alguém à toa.
		return CacheHeader{}, nil, nil, ErrIndexCacheVersionMismatch
	}
	l.i = len(indexCacheMagic)

	var h CacheHeader
	h.FormatVersion = int(l.uvarint(math.MaxInt32, "formatVersion"))
	h.ParserVersion = int(l.uvarint(math.MaxInt32, "parserVersion"))
	if l.err != nil {
		return CacheHeader{}, nil, nil, l.err
	}
	// Portão de versão ANTES de qualquer campo de layout — mesma ordem e
	// mesmo motivo do cache de busca: um layout futuro pode reaproveitar a
	// mesma mágica, e decodificar campos de um formato mais novo com o
	// leitor antigo produz lixo estruturalmente válido em vez de um erro
	// claro de incompatibilidade.
	if h.FormatVersion != indexCacheCodecVers {
		return CacheHeader{}, nil, nil, ErrIndexCacheVersionMismatch
	}

	h.VaultPath = l.str("vaultPath")
	h.NoteCount = int(l.uvarint(math.MaxInt32, "noteCount"))
	h.AssetCount = int(l.uvarint(math.MaxInt32, "assetCount"))
	if l.err != nil {
		return h, nil, nil, l.err
	}

	nNotas := l.uvarint(limiteNotas, "nNotas")
	if l.err != nil {
		return h, nil, nil, l.err
	}
	notes := make([]*Note, nNotas)
	for i := range notes {
		notes[i] = l.note()
		if l.err != nil {
			return h, nil, nil, l.err
		}
	}

	nAnexos := l.uvarint(limiteNotas, "nAnexos")
	if l.err != nil {
		return h, nil, nil, l.err
	}
	assets := make([]*Asset, nAnexos)
	for i := range assets {
		assets[i] = l.asset()
		if l.err != nil {
			return h, nil, nil, l.err
		}
	}

	return h, notes, assets, nil
}
