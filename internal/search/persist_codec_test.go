package search

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"testing"
)

// corpusCodec monta termos com várias postings por termo, para que as fatias de
// posição saiam VIZINHAS no array de posições — que é a condição do defeito
// testado em TestArenaNaoVazaEntrePostings.
func corpusCodec(nTermos, nDocs, nPos int) (map[string]map[string][]TokenPosition, map[string]int) {
	termos := make(map[string]map[string][]TokenPosition, nTermos)
	for t := 0; t < nTermos; t++ {
		docs := make(map[string][]TokenPosition, nDocs)
		for d := 0; d < nDocs; d++ {
			pos := make([]TokenPosition, nPos)
			for k := range pos {
				start := int64(t*1_000_000 + d*1_000 + k*7)
				pos[k] = TokenPosition{Start: start, End: start + int64(3+k%5)}
			}
			docs[fmt.Sprintf("pasta%02d/nota%04d.md", d%10, d)] = pos
		}
		termos[fmt.Sprintf("termo%05d", t)] = docs
	}
	comp := make(map[string]int, nDocs)
	for d := 0; d < nDocs; d++ {
		comp[fmt.Sprintf("pasta%02d/nota%04d.md", d%10, d)] = nTermos * nPos
	}
	return termos, comp
}

// ciclo grava e relê, devolvendo o índice montado sobre a base achatada.
func ciclo(t *testing.T, h CacheHeader, termos map[string]map[string][]TokenPosition, comp map[string]int) (*Inverted, *baseSoA) {
	t.Helper()
	var buf bytes.Buffer
	if err := escreveCache(&buf, h, termos, comp); err != nil {
		t.Fatalf("escreveCache: %v", err)
	}
	got, base, err := leCache(buf.Bytes())
	if err != nil {
		t.Fatalf("leCache: %v", err)
	}
	if got != h {
		t.Fatalf("cabeçalho = %+v, quer %+v", got, h)
	}
	return newInvertedFromSoA(base), base
}

func TestCodecRoundTrip(t *testing.T) {
	orig, comp := corpusCodec(40, 12, 9)
	h := CacheHeader{
		FormatVersion:   CacheFormatVersion,
		ParserVersion:   CacheParserVersion,
		AnalyzerVersion: CacheAnalyzerVersion,
		VaultPath:       `C:\cofre`,
		NoteCount:       12,
	}

	inv, base := ciclo(t, h, orig, comp)

	if len(base.termos) != len(orig) {
		t.Fatalf("termos = %d, quer %d", len(base.termos), len(orig))
	}
	// docLengths tem de voltar EXATO, e nao recalculado a partir das postings:
	// um token cuja forma reduzida difere da raiz entra em duas postings, e a
	// soma daria o dobro. É o divisor da normalização por tamanho do BM25.
	if inv.DocCount() != len(comp) {
		t.Fatalf("DocCount = %d, quer %d", inv.DocCount(), len(comp))
	}
	for path, n := range comp {
		if got := inv.DocLength(path); got != n {
			t.Fatalf("DocLength(%q) = %d, quer %d", path, got, n)
		}
	}

	for termo, docs := range orig {
		lidas := inv.Postings(termo)
		if len(lidas) != len(docs) {
			t.Fatalf("%q: %d postings, quer %d", termo, len(lidas), len(docs))
		}
		for _, p := range lidas {
			quer, ok := docs[p.Path]
			if !ok {
				t.Fatalf("%q: posting %q nao existia no original", termo, p.Path)
			}
			if len(p.Positions) != len(quer) {
				t.Fatalf("%q/%q: %d posições, quer %d", termo, p.Path, len(p.Positions), len(quer))
			}
			for i := range quer {
				if p.Positions[i] != quer[i] {
					t.Fatalf("%q/%q[%d] = %+v, quer %+v", termo, p.Path, i, p.Positions[i], quer[i])
				}
			}
		}
		// Postings sai ordenado por caminho SEM ordenar nada, porque o pathID é
		// a posição no vetor ordenado de caminhos. Se isto quebrar, quebrou o
		// contrato de ordem do arquivo, não o Postings.
		if !sort.SliceIsSorted(lidas, func(i, j int) bool { return lidas[i].Path < lidas[j].Path }) {
			t.Fatalf("%q: postings fora de ordem", termo)
		}
	}
}

// TestArenaNaoVazaEntrePostings guarda o defeito que o array compartilhado de
// posições INTRODUZ.
//
// Todas as posições vivem num bloco só. Uma subfatia comum (`pos[i:j]`) herda a
// capacidade até o FIM do bloco, então um `append` nela grava por cima da
// primeira posição da posting vizinha — em silêncio, e numa nota diferente da
// que o chamador estava mexendo. Por isso posicoesDaPosting recorta com
// `pos[i:j:j]`.
func TestArenaNaoVazaEntrePostings(t *testing.T) {
	orig, comp := corpusCodec(30, 10, 6)
	inv, _ := ciclo(t, CacheHeader{FormatVersion: CacheFormatVersion, NoteCount: 10}, orig, comp)

	// Cópia de referência ANTES de qualquer append.
	type chave struct{ termo, path string }
	esperado := make(map[chave][]TokenPosition)
	for termo := range orig {
		for _, p := range inv.Postings(termo) {
			cp := make([]TokenPosition, len(p.Positions))
			copy(cp, p.Positions)
			esperado[chave{termo, p.Path}] = cp
		}
	}

	// A vizinhança no bloco não é observável daqui. Anexando em TODAS as
	// fatias devolvidas, qualquer par vizinho corrompido aparece.
	sentinela := TokenPosition{Start: -777, End: -777}
	for termo := range orig {
		for _, p := range inv.Postings(termo) {
			_ = append(p.Positions, sentinela) //nolint:gocritic // o append é o teste
		}
	}

	for k, quer := range esperado {
		var tem []TokenPosition
		for _, p := range inv.Postings(k.termo) {
			if p.Path == k.path {
				tem = p.Positions
			}
		}
		if len(tem) != len(quer) {
			t.Fatalf("%q/%q: %d posições, quer %d", k.termo, k.path, len(tem), len(quer))
		}
		for i := range quer {
			if tem[i] != quer[i] {
				t.Fatalf("posting vizinha corrompida em %q/%q[%d]: %+v, quer %+v\n"+
					"a fatia do bloco de posições foi recortada sem travar a capacidade",
					k.termo, k.path, i, tem[i], quer[i])
			}
		}
	}
}

// TestCodecRecusaLayoutAnterior: arquivo com a mágica certa mas versão de
// formato diferente é recusado como VERSÃO, não decodificado.
func TestCodecRecusaLayoutAnterior(t *testing.T) {
	var buf bytes.Buffer
	h := CacheHeader{FormatVersion: CacheFormatVersion - 1, NoteCount: 1}
	termos := map[string]map[string][]TokenPosition{"a": {"n.md": {{Start: 0, End: 1}}}}
	if err := escreveCache(&buf, h, termos, map[string]int{"n.md": 1}); err != nil {
		t.Fatalf("escreveCache: %v", err)
	}
	_, _, err := leCache(buf.Bytes())
	if !errors.Is(err, ErrCacheVersionMismatch) {
		t.Fatalf("err = %v, quer ErrCacheVersionMismatch", err)
	}
}

// TestCodecRecusaPrefixoTruncado: nenhum prefixo do corpo em varint pode virar
// panic nem estrutura aceita.
//
// O limite do laço é posArrayOffset, e não len(b), desde a Task 89:
// escreveCache passou a acrescentar, DEPOIS do corpo em varint que leCache
// decodifica, uma seção fixa de posições e um rodapé — conteúdo que existe
// para quem mapeia o arquivo (leCacheComArena), e que leCache (o caminho sem
// arena, testado aqui) nunca lê. Um prefixo que corta só essa cauda não é
// truncamento do que leCache consome; testá-lo reprovaria por um motivo que
// não é o que este teste guarda.
func TestCodecRecusaPrefixoTruncado(t *testing.T) {
	var buf bytes.Buffer
	termos := map[string]map[string][]TokenPosition{
		"a": {"n.md": {{Start: 0, End: 1}}},
		"b": {"n.md": {{Start: 5, End: 9}}, "o.md": {{Start: 2, End: 4}}},
	}
	comp := map[string]int{"n.md": 2, "o.md": 1}
	if err := escreveCache(&buf, CacheHeader{FormatVersion: CacheFormatVersion, NoteCount: 2}, termos, comp); err != nil {
		t.Fatalf("escreveCache: %v", err)
	}
	b := buf.Bytes()

	offset, _, ok := leRodape(b)
	if !ok {
		t.Fatalf("rodape ausente ou invalido no arquivo recem-gravado")
	}

	// Entre o último byte que leCache realmente consome e posArrayOffset fica
	// o padding de alinhamento — até 7 bytes zero, sem significado nenhum
	// para quem não mapeia. Um prefixo que corte só essa zona também
	// decodifica com sucesso, e isso não é o defeito que este teste guarda;
	// por isso o limite fica 7 bytes antes do offset, e não nele.
	limite := int(offset) - 7

	for n := 1; n < limite; n++ {
		_, _, err := leCache(b[:n])
		if err == nil {
			t.Fatalf("prefixo de %d bytes (< %d, com folga do padding) foi aceito", n, limite)
		}
	}
}

// TestCodecRecusaOrdemQuebrada guarda a checagem de ordem.
//
// Busca binária sobre um vetor fora de ordem NÃO devolve erro: devolve "não
// existe" para um termo que existe. A busca passaria a não achar notas que
// contêm a palavra, sem log e com cara de cofre que não tem o termo. Aqui a
// ordem é quebrada trocando dois termos vizinhos no arquivo já gravado.
func TestCodecRecusaOrdemQuebrada(t *testing.T) {
	var buf bytes.Buffer
	// Termos de mesmo tamanho para a troca ser um swap de bytes, sem mexer nos
	// varints de comprimento.
	termos := map[string]map[string][]TokenPosition{
		"aaa": {"n.md": {{Start: 0, End: 1}}},
		"bbb": {"n.md": {{Start: 2, End: 3}}},
	}
	if err := escreveCache(&buf, CacheHeader{FormatVersion: CacheFormatVersion, NoteCount: 1}, termos, map[string]int{"n.md": 2}); err != nil {
		t.Fatalf("escreveCache: %v", err)
	}
	b := buf.Bytes()

	i := bytes.Index(b, []byte("aaa"))
	j := bytes.Index(b, []byte("bbb"))
	if i < 0 || j < 0 {
		t.Fatalf("termos nao encontrados no arquivo (i=%d j=%d)", i, j)
	}
	copy(b[i:i+3], "bbb")
	copy(b[j:j+3], "aaa")

	_, _, err := leCache(b)
	if !errors.Is(err, ErrCacheCorrupted) {
		t.Fatalf("err = %v, quer ErrCacheCorrupted — ordem quebrada tem de ser recusada", err)
	}
}

// TestCodecRecusaTotaisQueNaoBatem guarda o caso em que o corpo entrega menos do
// que o cabeçalho declarou.
//
// Os arrays são dimensionados pelos totais. Se o corpo trouxer menos, as caudas
// ficam ZERADAS — e uma cauda zerada não é um erro visível: são postings do
// caminho 0 com zero posições, dado inventado com aparência legítima.
func TestCodecRecusaTotaisQueNaoBatem(t *testing.T) {
	var buf bytes.Buffer
	termos := map[string]map[string][]TokenPosition{
		"aaa": {"n.md": {{Start: 0, End: 1}}},
	}
	if err := escreveCache(&buf, CacheHeader{FormatVersion: CacheFormatVersion, NoteCount: 1}, termos, map[string]int{"n.md": 1}); err != nil {
		t.Fatalf("escreveCache: %v", err)
	}
	b := buf.Bytes()

	// nTermos vem logo antes do primeiro termo. Zerá-lo faz o corpo entregar
	// zero postings contra o total declarado de 1.
	i := bytes.Index(b, []byte("aaa"))
	if i < 2 {
		t.Fatalf("termo nao encontrado (i=%d)", i)
	}
	b[i-2] = 0 // nTermos = 0

	_, _, err := leCache(b)
	if !errors.Is(err, ErrCacheCorrupted) {
		t.Fatalf("err = %v, quer ErrCacheCorrupted — totais que nao batem tem de ser recusados", err)
	}
}
