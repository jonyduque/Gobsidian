package service

// SetSnippetWorkers troca o limite de trabalhadores de recorte e devolve a
// funcao que o restaura. Existe para TestRNF04SnippetConcurrencyLimit200 medir
// o caminho sequencial (1) e o concorrente (o padrao) sob a mesma carga de
// maquina, intercalados na mesma rodada. Nenhum teste paralelo deve chamar
// Search enquanto ele esta em vigor: a variavel nao tem trava, de proposito,
// para nao pagar um atomic no caminho quente de producao por causa de um teste.
func SetSnippetWorkers(n int) (restaurar func()) {
	antes := maxSnippetWorkers
	maxSnippetWorkers = n
	return func() { maxSnippetWorkers = antes }
}
