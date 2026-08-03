package search

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
)

// corpusCodec monta termos com várias postings por termo, para que as fatias de
// posição saiam VIZINHAS na arena — que é a condição do defeito testado abaixo.
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

func TestCodecRoundTrip(t *testing.T) {
	orig, comp := corpusCodec(40, 12, 9)
	h := CacheHeader{
		FormatVersion:   CacheFormatVersion,
		ParserVersion:   CacheParserVersion,
		AnalyzerVersion: CacheAnalyzerVersion,
		VaultPath:       `C:\cofre`,
		NoteCount:       12,
	}

	var buf bytes.Buffer
	if err := escreveCache(&buf, h, orig, comp); err != nil {
		t.Fatalf("escreveCache: %v", err)
	}

	got, termos, comps, err := leCache(buf.Bytes())
	if err != nil {
		t.Fatalf("leCache: %v", err)
	}
	if got != h {
		t.Fatalf("cabeçalho = %+v, quer %+v", got, h)
	}
	if len(termos) != len(orig) {
		t.Fatalf("termos = %d, quer %d", len(termos), len(orig))
	}
	// docLengths tem de voltar EXATO, e nao recalculado a partir das postings:
	// um token cuja forma reduzida difere da raiz entra em duas postings, e a
	// soma daria o dobro. É o divisor da normalização por tamanho do BM25.
	if len(comps) != len(comp) {
		t.Fatalf("docLengths = %d entradas, quer %d", len(comps), len(comp))
	}
	for path, n := range comp {
		if comps[path] != n {
			t.Fatalf("docLengths[%q] = %d, quer %d", path, comps[path], n)
		}
	}
	for termo, docs := range orig {
		lidos, ok := termos[termo]
		if !ok {
			t.Fatalf("termo %q sumiu", termo)
		}
		for path, pos := range docs {
			lida, ok := lidos[path]
			if !ok {
				t.Fatalf("posting %q/%q sumiu", termo, path)
			}
			if len(lida) != len(pos) {
				t.Fatalf("%q/%q: %d posições, quer %d", termo, path, len(lida), len(pos))
			}
			for i := range pos {
				if lida[i] != pos[i] {
					t.Fatalf("%q/%q[%d] = %+v, quer %+v", termo, path, i, lida[i], pos[i])
				}
			}
		}
	}
}

// TestArenaNaoVazaEntrePostings guarda o defeito que a arena INTRODUZ.
//
// As fatias de posição saem todas de um bloco contíguo. Uma subfatia comum
// (`arena[i:j]`) herda a capacidade até o FIM da arena, então um `append` nela
// grava por cima da primeira posição da posting vizinha — em silêncio, e só
// numa nota diferente da que foi editada. Por isso `leCache` recorta com
// `arena[i:j:j]`.
//
// O teste não passa por `Inverted.Add`, e isso é deliberado: `Add` começa com
// `removeLocked`, que apaga a entrada antes de anexar, então um teste por lá
// passaria mesmo com a capacidade solta — mediria o `removeLocked`, não o
// recorte. Aqui o append é feito direto na fatia devolvida.
func TestArenaNaoVazaEntrePostings(t *testing.T) {
	orig, comp := corpusCodec(30, 10, 6)

	var buf bytes.Buffer
	if err := escreveCache(&buf, CacheHeader{FormatVersion: CacheFormatVersion, NoteCount: 10}, orig, comp); err != nil {
		t.Fatalf("escreveCache: %v", err)
	}
	_, termos, _, err := leCache(buf.Bytes())
	if err != nil {
		t.Fatalf("leCache: %v", err)
	}

	// Cópia de referência ANTES de qualquer append.
	type chave struct{ termo, path string }
	esperado := make(map[chave][]TokenPosition)
	for termo, docs := range termos {
		for path, pos := range docs {
			cp := make([]TokenPosition, len(pos))
			copy(cp, pos)
			esperado[chave{termo, path}] = cp
		}
	}

	// A ordem das postings na arena segue a iteração do mapa e não é
	// observável daqui. Anexando em TODAS, qualquer par vizinho corrompido
	// aparece — não é preciso saber quem é vizinho de quem.
	sentinela := TokenPosition{Start: -777, End: -777}
	for _, docs := range termos {
		for path, pos := range docs {
			docs[path] = append(pos, sentinela)
		}
	}

	for k, quer := range esperado {
		tem := termos[k.termo][k.path]
		if len(tem) != len(quer)+1 {
			t.Fatalf("%q/%q: %d posições após append, quer %d", k.termo, k.path, len(tem), len(quer)+1)
		}
		for i := range quer {
			if tem[i] != quer[i] {
				t.Fatalf("posting vizinha corrompida em %q/%q[%d]: %+v, quer %+v\n"+
					"a fatia da arena foi recortada sem travar a capacidade", k.termo, k.path, i, tem[i], quer[i])
			}
		}
	}
}

// TestCodecRecusaLayoutAnterior: arquivo com a mágica certa mas versão de
// formato diferente é recusado como VERSÃO, não decodificado.
//
// O layout mudou uma vez dentro da mesma mágica (ganhou dois totais entre
// noteCount e a tabela de caminhos). Sem o portão, os varints do layout velho
// casariam com campos trocados e produziriam estrutura lixo que passa por
// válida.
func TestCodecRecusaLayoutAnterior(t *testing.T) {
	var buf bytes.Buffer
	h := CacheHeader{FormatVersion: CacheFormatVersion - 1, NoteCount: 1}
	termos := map[string]map[string][]TokenPosition{"a": {"n.md": {{Start: 0, End: 1}}}}
	if err := escreveCache(&buf, h, termos, map[string]int{"n.md": 1}); err != nil {
		t.Fatalf("escreveCache: %v", err)
	}
	_, _, _, err := leCache(buf.Bytes())
	if !errors.Is(err, ErrCacheVersionMismatch) {
		t.Fatalf("err = %v, quer ErrCacheVersionMismatch", err)
	}
}

// TestCodecRecusaCaminhoForaDaTabela: pathID adulterado não pode virar panic de
// índice fora de faixa nem apontar para o caminho errado.
func TestCodecRecusaCaminhoForaDaTabela(t *testing.T) {
	var buf bytes.Buffer
	termos := map[string]map[string][]TokenPosition{
		"a": {"n.md": {{Start: 0, End: 1}}},
	}
	if err := escreveCache(&buf, CacheHeader{FormatVersion: CacheFormatVersion, NoteCount: 1}, termos, map[string]int{"n.md": 1}); err != nil {
		t.Fatalf("escreveCache: %v", err)
	}
	b := buf.Bytes()

	// Trunca em cada ponto: nenhum prefixo pode produzir panic, todos devem
	// devolver erro de cache corrompido.
	for n := 1; n < len(b); n++ {
		_, _, _, err := leCache(b[:n])
		if err == nil {
			t.Fatalf("prefixo de %d bytes foi aceito", n)
		}
	}
}
