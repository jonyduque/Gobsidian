package index

import "io"

// WriteIndexCacheForTest escreve um cache no formato corrente com o
// cabeçalho que o teste mandar — inclusive um cabeçalho que MENTE sobre a
// cobertura, para exercitar a checagem de LoadIndexCache sem depender de
// conseguir produzir um cache parcial de verdade (SaveIndexCache sempre
// grava um cabeçalho fiel ao que exportou).
//
// Vive aqui, e nao em persist.go, porque um simbolo que so teste chama nao
// pertence ao binario do produto. Arquivo _test.go: nada disso existe no
// binario final.
func WriteIndexCacheForTest(w io.Writer, h CacheHeader, notes []*Note, assets []*Asset) error {
	return escreveIndexCache(w, h, notes, assets)
}
