package search

import "testing"

// TestMelhorJanelaEscolheOndeOsTermosConvergem é o achado M16 isolado.
//
// A âncora era a primeira ocorrência do PRIMEIRO termo da consulta, na ordem em
// que o usuário a escreveu. Buscar `13.1.10 Substituição de candidatos` ancorava
// no `13` do topo do documento, e o trecho devolvido não continha a frase — foi
// a causa direta do incidente de campo de 2026-08-15.
//
// O cenário reproduz essa forma: um termo comum que aparece cedo e sozinho, e o
// grupo de termos que só converge mais adiante.
func TestMelhorJanelaEscolheOndeOsTermosConvergem(t *testing.T) {
	// "13" no topo, sozinho. Os três termos juntos lá no fim.
	todas := []ocorrenciaAncora{
		{term: "13", start: 10, end: 12},
		{term: "13", start: 40, end: 42},
		{term: "13", start: 900, end: 902},
		{term: "substituicao", start: 910, end: 922},
		{term: "candidatos", start: 930, end: 940},
	}

	got := melhorJanela(todas, 240)
	if got == nil {
		t.Fatal("melhorJanela devolveu nil com ocorrencias presentes")
	}
	if got.start != 900 {
		t.Errorf("ancora em %d (termo %q), queria 900\n"+
			"a janela de 900..940 reune TRES termos distintos; a de 10 reune um.\n"+
			"ancorar no primeiro termo da consulta e o defeito do M16",
			got.start, got.term)
	}
}

// TestMelhorJanelaDesempataPelaEsquerda fixa o critério de desempate.
//
// Sem ele, duas janelas igualmente informativas poderiam alternar entre
// chamadas — e o mesmo par (nota, consulta) devolveria trechos diferentes,
// que é a classe de resposta instável que esta base recusa.
func TestMelhorJanelaDesempataPelaEsquerda(t *testing.T) {
	todas := []ocorrenciaAncora{
		{term: "a", start: 0, end: 1},
		{term: "b", start: 10, end: 11},
		{term: "a", start: 500, end: 501},
		{term: "b", start: 510, end: 511},
	}
	got := melhorJanela(todas, 100)
	if got == nil || got.start != 0 {
		t.Errorf("ancora = %v, queria a de start=0: empate resolve pela ordem de leitura", got)
	}
}

func TestMelhorJanelaBordas(t *testing.T) {
	if melhorJanela(nil, 240) != nil {
		t.Error("sem ocorrencia nenhuma o resultado tem de ser nil")
	}
	uma := []ocorrenciaAncora{{term: "x", start: 7, end: 8}}
	if got := melhorJanela(uma, 240); got == nil || got.start != 7 {
		t.Errorf("uma ocorrencia so: got = %v, queria start=7", got)
	}
	// Largura zero não pode devolver nil nem entrar em laço: cai na primeira.
	if got := melhorJanela(uma, 0); got == nil || got.start != 7 {
		t.Errorf("largura zero: got = %v, queria start=7", got)
	}
}
