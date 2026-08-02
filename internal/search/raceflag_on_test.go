//go:build race

package search_test

// raceEnabled diz se o binario de teste foi compilado com -race. Ver o gemeo
// em internal/service/raceflag_on_test.go: a razao e a mesma, e o custo de nao
// ter esta guarda aqui foi medido.
//
// TestBM25KernelLatency nasceu com teto de 80 ms cobrado nos DOIS modos, sob o
// argumento de que 80 ms davam 4,5x de folga sobre os 17,9 ms que a maquina de
// desenvolvimento media com -race. O numero era da maquina errada: no runner
// compartilhado do CI, com -race, o mesmo teste mediu mediana de 26,6 ms e p95
// de 107,1 ms, e reprovou sem regressao nenhuma no codigo.
//
// Assercao de TEMPO nao vale sob o detector. O teto continua sendo cobrado —
// em scripts/verify.ps1, na etapa que roda sem -race.
const raceEnabled = true
