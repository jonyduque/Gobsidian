// Package index guarda o cofre em memoria: metadados e OFFSETS DE BYTE,
// nunca conteudo.
//
// E essa escolha que sustenta o orcamento de 60 MB de RSS e que faz ler uma
// secao de 2 KB numa nota de 500 KB custar 2 KB em vez de 500. Anexo entra
// por nome e nunca e aberto — abri-lo dispararia download de arquivo
// somente-nuvem, e nao abri-lo ainda impede que todo embed de imagem seja
// contado como link quebrado.
package index

import "strings"

// aliasKey normaliza a chave de byAlias. Toda escrita e toda leitura passam
// por aqui: o boot indexava minusculo e Replace indexava cru, e a entrada
// que Remove nao encontrava sobrevivia apontando para uma nota deletada.
func aliasKey(alias string) string { return strings.ToLower(alias) }
