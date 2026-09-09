package selfupdate

import "testing"

// TestPrecisaAtualizarNaoRebaixaUmBuildMaisNovo cobre um defeito encontrado
// rodando `gobsidian update --check` contra a API real do GitHub em 2026-09-09.
//
// O binario local reportava "instalada: v1.5.1-20-gcf8991a-dirty, publicada:
// v1.5.1" e anunciava versao nova. Um `update` naquele estado trocaria um build
// MAIS NOVO por um mais velho -- e o teste com transporte falso nunca pegaria
// isso, porque ele nunca comparou versoes de verdade.
func TestPrecisaAtualizarNaoRebaixaUmBuildMaisNovo(t *testing.T) {
	casos := []struct {
		instalada string
		publicada string
		precisa   bool
		porQue    string
	}{
		{"v1.5.1", "v1.5.1", false, "identicas"},
		{"v1.5.1-20-gcf8991a-dirty", "v1.5.1", false, "build local DEPOIS da tag: esta a frente"},
		{"v1.5.1-1-gabcdef0", "v1.5.1", false, "um commit depois da tag"},
		{"v1.5.0", "v1.5.1", true, "release anterior"},
		{"v1.4.9-3-gabc", "v1.5.1", true, "descendente de OUTRA tag, mais velha"},
		{"dev", "v1.5.1", true, "sem versao: nao da para provar que esta em dia"},
		{"", "v1.5.1", true, "vazia"},
		{"v1.5.1", "", false, "sem publicada nao ha o que comparar"},
		{"v1.5.10", "v1.5.1", true, "prefixo de TEXTO sem o hifen NAO conta como descendente"},
	}

	for _, c := range casos {
		if got := PrecisaAtualizar(c.instalada, c.publicada); got != c.precisa {
			t.Errorf("PrecisaAtualizar(%q, %q) = %v, esperado %v (%s)",
				c.instalada, c.publicada, got, c.precisa, c.porQue)
		}
	}
}
