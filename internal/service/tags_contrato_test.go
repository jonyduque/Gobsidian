package service_test

import (
	"context"
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

// O contrato de tag das tres tools que a recebem: a tag pedida casa a si mesma
// e suas subtags, o '#' inicial e opcional, e a comparacao ignora caixa e forma
// Unicode. Ate a Task 180 note_list casava subtag e vault_search nao — o MESMO
// parametro, duas respostas —, e tag_list devolvia uma entrada por grafia.
//
// O servico sai de createSearchService, e nao de newTestService: aquele injeta
// inverted = nil, e vault_search sobre um indice invertido nulo casa zero
// resultados sempre. Um teste de filtro que corre no caminho vazio passa por
// nao haver o que filtrar, e nao pode falhar.
//
// As duas grafias de "Acao" estao em notas SEPARADAS — d.md so em NFD com
// maiuscula, e.md so em NFC minuscula — e nao juntas na mesma nota. Com as duas
// na mesma nota, um pedido em NFC casava pela grafia NFC e o teste passava com
// a normalizacao Unicode removida: medido, `mutate.ps1` sobre ParaNFC deu exit
// 1. Separadas, so a dobra reune as duas, e a mutacao reprova.
//
// As tags acentuadas entram pelo FRONTMATTER: o parser inline corta a tag no
// sinal combinante (ver o comentario de internal/index/tag_chave_test.go),
// entao e o frontmatter que carrega a forma Unicode intacta ate o indice.
func cofreComTags(t *testing.T) *service.Service {
	t.Helper()
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "# A\n\nnota comum #Projeto/Alpha\n",
		"b.md": "# B\n\nnota comum #projeto\n",
		"c.md": "# C\n\nnota comum #outra\n",
		"d.md": "---\ntags: [\"Ação\"]\n---\n# D\n\nnota comum\n",
		"e.md": "---\ntags: [\"ação\"]\n---\n# E\n\nnota comum\n",
	})
	return svc
}

func TestVaultSearchTagsCasaSubtag(t *testing.T) {
	svc := cofreComTags(t)
	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "comum", Tags: []string{"#PROJETO"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Results) != 2 {
		t.Fatalf("tags=[#PROJETO]: %d resultados, quero 2 (projeto e projeto/alpha): %+v", len(res.Results), res.Results)
	}
}

// O pedido vem em NFC e casa as DUAS notas: a que gravou a tag em NFC e a que
// gravou a mesma tag em NFD com maiuscula.
func TestVaultSearchTagsNFD(t *testing.T) {
	svc := cofreComTags(t)
	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "comum", Tags: []string{"ação"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	achou := map[string]bool{}
	for _, r := range res.Results {
		achou[r.Path] = true
	}
	if len(res.Results) != 2 || !achou["d.md"] || !achou["e.md"] {
		t.Fatalf("tags=[ação]: %+v, quero d.md e e.md — a mesma tag em NFD e em NFC", res.Results)
	}
}

func TestTagListDevolveFormaDobrada(t *testing.T) {
	svc := cofreComTags(t)
	res, err := svc.TagList(context.Background(), service.TagRequest{Prefix: "a\u00e7"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tags) != 1 || res.Tags[0].Tag != "ação" || res.Tags[0].Count != 2 {
		t.Fatalf("tag_list(a\u00e7) = %+v, quero UMA entrada ação com count 2 (duas notas, duas grafias)", res.Tags)
	}
}

// O ramo hierarquico responde pela MESMA chave que o plano: duas grafias da
// mesma tag sao uma raiz, nao duas.
func TestTagListHierarquicoDobraGrafias(t *testing.T) {
	svc := cofreComTags(t)
	res, err := svc.TagList(context.Background(), service.TagRequest{Prefix: "a\u00e7", Hierarchical: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tags) != 1 || res.Tags[0].Tag != "ação" || res.Tags[0].Count != 2 {
		t.Fatalf("tag_list(a\u00e7, hierarquico) = %+v, quero UMA entrada ação com count 2", res.Tags)
	}
}
