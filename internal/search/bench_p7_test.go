package search

import (
	"fmt"
	"io"
	"testing"
)

// corpusParaEscrita monta um conjunto de postings da ordem de grandeza que o
// achado P7 nomeia: o cofre de referência tem ~18,2 milhões de posições, e a
// seção fixa custa 16 bytes cada — os 291 MB documentados no próprio código.
//
// O benchmark usa uma fração disso para caber num teste, e o que importa é a
// RAZÃO entre alocar a seção inteira e escrevê-la em blocos, que não depende da
// escala.
func corpusParaEscrita(nTermos, nDocs, posPorPar int) (map[string]map[string][]TokenPosition, map[string]int) {
	termos := make(map[string]map[string][]TokenPosition, nTermos)
	docLengths := make(map[string]int, nDocs)
	for d := range nDocs {
		docLengths[fmt.Sprintf("nota%05d.md", d)] = nTermos * posPorPar
	}
	for t := range nTermos {
		termo := fmt.Sprintf("termo%05d", t)
		docs := make(map[string][]TokenPosition, nDocs)
		for d := range nDocs {
			pos := make([]TokenPosition, posPorPar)
			for k := range pos {
				base := int64((t*nDocs+d)*posPorPar + k)
				pos[k] = TokenPosition{Start: base * 8, End: base*8 + 5}
			}
			docs[fmt.Sprintf("nota%05d.md", d)] = pos
		}
		termos[termo] = docs
	}
	return termos, docLengths
}

// BenchmarkEscreveCache mede o achado P7 na dimensão em que ele custa: a
// ALOCAÇÃO durante o salvamento.
//
// A seção fixa mapeável era montada num `make([]byte, totPos*16)` dentro do laço
// que grava os varints, e vivia inteira na heap ao lado do índice — justamente
// no regime em que a arena mapeada existe para economizar RAM. Os salvamentos
// periódicos da construção em segundo plano repetem o pico a cada 60 s.
func BenchmarkEscreveCache(b *testing.B) {
	// 200 termos x 100 docs x 20 posições = 400 mil posições = 6,4 MB de seção
	// fixa. Grande o bastante para o buffer aparecer, pequeno o bastante para o
	// benchmark rodar.
	termos, docLengths := corpusParaEscrita(200, 100, 20)
	h := CacheHeader{FormatVersion: CacheFormatVersion, NoteCount: 100}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		if err := escreveCache(io.Discard, h, termos, docLengths); err != nil {
			b.Fatalf("escreveCache: %v", err)
		}
	}
}
