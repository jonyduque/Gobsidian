//go:build race

package service_test

// raceEnabled diz se o binario de teste foi compilado com -race. O detector de
// corrida multiplica a latencia por algo entre 2x e 6x — aqui, a busca por
// frase saiu de 174 ms para 1,00 s —, entao asseracao de TEMPO nao pode valer
// sob ele. A medicao continua sendo feita e registrada nos dois modos; so o
// teto deixa de ser cobrado quando o numero nao e comparavel.
//
// Build tag em arquivo separado, e nao um `if` no corpo do teste, pela mesma
// razao que o codigo de plataforma deste projeto: a condicao pertence a
// compilacao, nao a logica.
const raceEnabled = true
