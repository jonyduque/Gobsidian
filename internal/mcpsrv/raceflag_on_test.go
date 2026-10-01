//go:build race

package mcpsrv_test

// raceEnabled diz se o binario de teste foi compilado com -race. Ver o gemeo
// em internal/search/raceflag_on_test.go: assercao de TEMPO nao vale sob o
// detector, que multiplica a latencia por 2 a 6.
//
// TestLerEsquemasCustaPouco nasceu sem esta guarda e reprovou o gate em
// 2026-10-01 sob -race, com 0,52 s contra teto de 0,5 s, sem regressao
// nenhuma. O teto continua cobrado em scripts/verify.ps1, na etapa sem -race.
const raceEnabled = true
