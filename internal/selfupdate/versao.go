package selfupdate

import "strings"

// PrecisaAtualizar compara a versao instalada com a publicada.
//
// A comparacao ingenua -- `instalada == publicada` -- tem um defeito que so
// aparece fora de um release limpo, e foi assim que ele apareceu: rodando
// `update --check` de verdade contra a API do GitHub, em 2026-09-09, um binario
// compilado localmente reportou "instalada: v1.5.1-20-gcf8991a-dirty,
// publicada: v1.5.1" e anunciou que havia versao NOVA. Um `update` naquele
// estado faria DOWNGRADE, trocando um build mais novo por um mais velho.
//
// `scripts/build.ps1` carimba `git describe --tags --always --dirty`, cuja
// saida para um commit depois da tag e "<tag>-<n>-g<sha>" (com "-dirty" quando
// ha trabalho nao commitado). Uma instalada que comeca com "<publicada>-" e,
// por construcao, DESCENDENTE dela: esta a frente, nao atras.
//
// Nao ha comparacao semantica de versao aqui, e e deliberado: exigiria uma
// dependencia nova, e `go mod tidy` e proibido neste projeto (varias deps estao
// fixadas sem importador). O que se responde e mais estreito e verificavel:
// "esta instalada e a publicada, ou um descendente dela?".
//
// Versao vazia, "dev" ou "unknown" devolve true: nao da para provar que esta em
// dia, e mandar conferir e o lado seguro -- o oposto, dizer que esta atualizado
// sem saber, e a resposta que engana.
func PrecisaAtualizar(instalada, publicada string) bool {
	if publicada == "" {
		return false
	}
	if instalada == publicada {
		return false
	}
	// Descendente da tag publicada: "v1.5.1-20-gcf8991a-dirty" para "v1.5.1".
	if strings.HasPrefix(instalada, publicada+"-") {
		return false
	}
	return true
}
