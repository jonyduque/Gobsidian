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

func (ix *Index) buildAliasMap() {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	for _, n := range ix.notes {
		for _, alias := range n.Aliases {
			lower := strings.ToLower(alias)
			ix.byAlias[lower] = append(ix.byAlias[lower], n.Path)
		}
	}
}
