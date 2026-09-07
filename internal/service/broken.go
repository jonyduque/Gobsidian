package service

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/text"
)

// BrokenLinksRequest sao os parametros de vault_broken_links.
type BrokenLinksRequest struct {
	// State filtra: "target_missing", "anchor_missing" ou "" (os dois).
	State string `json:"state"`
	// Prefix restringe a origem a um caminho canonico (pasta ou nota). Vazio =
	// cofre inteiro.
	Prefix string `json:"prefix"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// BrokenLink e um link sem alvo ou sem ancora, com o bastante para achar e
// corrigir: a nota de origem, a grafia da referencia e o texto ao redor dela.
type BrokenLink struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Anchor  string `json:"anchor,omitempty"`
	Alias   string `json:"alias,omitempty"`
	Kind    string `json:"kind"`
	State   string `json:"state"`
	Context string `json:"context,omitempty"`
}

// BrokenLinksResult e o retorno de vault_broken_links. Total e a contagem
// ANTES de offset/limit, como em note_list, para o cliente saber que existe
// mais do que ele recebeu.
type BrokenLinksResult struct {
	Links []BrokenLink `json:"links"`
	Total int          `json:"total"`
}

// achadoQuebrado carrega o offset da referencia no corpo so ate a ordenacao.
// Start nao vai para o retorno — quem corrige o link o localiza pelo Context e
// pela grafia —, mas e ele que torna a ordem dentro de uma nota deterministica
// sem depender da ordem em que o indice guardou os links.
type achadoQuebrado struct {
	link  BrokenLink
	start int64
}

// BrokenLinks lista o que vault_stats apenas conta.
//
// `vault_stats` com include_health devolve broken_links e broken_anchors, dois
// numeros; quem quer CORRIGIR precisa de onde eles estao, e a unica saida era
// chamar note_metadata nota a nota. Esta tool percorre o mesmo laco de saude de
// graph.go e devolve os links em vez da contagem.
//
// LinkExternal fica de fora pelo mesmo motivo que fica de fora de VaultStats:
// uma URL nunca foi para o cofre, e lista-la como quebrada afoga o sinal em
// falso positivo. LinkOK tambem nao entra — inclusive a auto-ancora, que desde
// a Task 182 resolve para a propria nota de origem.
//
// Nao recebe ctx util: le so o indice em memoria.
func (s *Service) BrokenLinks(_ context.Context, req BrokenLinksRequest) (BrokenLinksResult, error) {
	if s.index == nil {
		return BrokenLinksResult{}, Errorf(CodeVaultUnavailable, "indice indisponivel")
	}

	// Mesma porta de enum das outras tools: um state fora da lista tem de
	// dizer INVALID_ARGUMENT, e nao responder o cofre inteiro como se o filtro
	// tivesse valido. O padrao vazio significa "os dois estados".
	estado, err := ValidarEnum("state", req.State, "", "target_missing", "anchor_missing")
	if err != nil {
		return BrokenLinksResult{}, err
	}

	// A comparacao de prefixo passa pela mesma chave que o writer e o index
	// usam para caminho — NFC mais caixa —, senao "Sub/" e "sub/" seriam duas
	// pastas aqui e uma la. E prefixo de string, nao de segmento, como o de
	// tag_list: "sub" casa "sub/c.md" e tambem "subtotal.md".
	prefixo := text.ChaveDeCaminho(req.Prefix)

	limit := ComTeto(req.Limit)
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	// NotePaths, nao Paths: Paths inclui anexos, que Get nao resolve.
	var achados []achadoQuebrado
	for _, p := range s.index.NotePaths() {
		if prefixo != "" && !strings.HasPrefix(text.ChaveDeCaminho(string(p)), prefixo) {
			continue
		}
		n, ok := s.index.Get(p)
		if !ok {
			continue
		}
		for _, l := range n.Links {
			if l.State != index.LinkTargetMissing && l.State != index.LinkAnchorMissing {
				continue
			}
			if estado != "" && l.State.String() != estado {
				continue
			}
			achados = append(achados, achadoQuebrado{
				link: BrokenLink{
					Source:  string(p),
					Target:  l.Target,
					Anchor:  l.Anchor,
					Alias:   l.Alias,
					Kind:    l.Kind.String(),
					State:   l.State.String(),
					Context: l.Context,
				},
				start: l.Start,
			})
		}
	}

	// Ordem deterministica: a origem e depois a posicao no corpo. Uma lista
	// paginada cuja ordem varia entre chamadas repete item numa pagina e some
	// com outro na seguinte.
	slices.SortFunc(achados, func(a, b achadoQuebrado) int {
		if c := cmp.Compare(a.link.Source, b.link.Source); c != 0 {
			return c
		}
		return cmp.Compare(a.start, b.start)
	})

	total := len(achados)
	var pagina []achadoQuebrado
	if offset < total {
		pagina = achados[offset:min(offset+limit, total)]
	}

	res := BrokenLinksResult{Links: make([]BrokenLink, len(pagina)), Total: total}
	for i, a := range pagina {
		res.Links[i] = a.link
	}
	return res, nil
}
