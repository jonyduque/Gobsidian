package search

import "io"

// WriteCacheForTest escreve um cache no formato corrente, com o cabecalho que
// o teste mandar.
//
// Substituiu NewEncoderForTest, que devolvia um gob.Encoder. Depois da troca
// de formato, um arquivo gob deixa de ter a assinatura "GBS2" e e recusado
// logo na primeira leitura — entao o teste de "versao de analisador
// divergente" passaria sem nunca chegar a comparar versao nenhuma. Teste que
// passa pelo motivo errado e pior que teste ausente.
//
// Vive aqui, e nao em persist.go, porque um simbolo que so teste chama nao
// pertence ao binario do produto. Arquivo _test.go: nada disso existe no
// binario final.
func WriteCacheForTest(
	w io.Writer,
	h CacheHeader,
	termos map[string]map[string][]TokenPosition,
	docLengths map[string]int,
) error {
	return escreveCache(w, h, termos, docLengths)
}
