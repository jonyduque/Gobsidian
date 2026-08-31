package service_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

// TestSearchMatchOffsetApontaParaOTermoNoArquivo prova o encadeamento: buscar,
// pegar o offset, ler ali, achar o termo.
//
// A asserção nao e sobre o VALOR do offset — numero conferido a mao vira
// tautologia na primeira mudanca de fixture. O enchimento antes do termo e
// obrigatorio: perto do inicio, um offset errado passaria por acidente.
func TestSearchMatchOffsetApontaParaOTermoNoArquivo(t *testing.T) {
	corpo := strings.Repeat("enchimento sem valor nenhum\n", 1500) +
		"a palavra procurada e xifopago aqui\n"

	svc, _, _, _ := createSearchService(t, map[string]string{
		"grande.md": corpo,
		"outra.md":  "nota sem o termo, so para o cofre nao ter uma nota so\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{
		Query: "xifopago",
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) == 0 {
		t.Fatal("nenhum resultado para um termo que existe na nota")
	}

	h := res.Hits[0]
	if h.Path != "grande.md" {
		t.Fatalf("primeiro hit = %q, quer grande.md", h.Path)
	}
	if h.MatchOffset == nil {
		t.Fatal("MatchOffset ausente num hit que tem trecho")
	}
	if *h.MatchOffset == 0 {
		t.Fatal("MatchOffset = 0, e o termo esta a mais de 40 KB do inicio do arquivo")
	}

	lido, err := svc.ReadNote(context.Background(), service.ReadRequest{
		Path:     h.Path,
		Offset:   h.MatchOffset,
		MaxBytes: 64,
	})
	if err != nil {
		t.Fatalf("ReadNote(offset=%d): %v", *h.MatchOffset, err)
	}
	if !strings.HasPrefix(lido.Content, "xifopago") {
		t.Fatalf("ler em match_offset=%d deu %q, que nao comeca pelo termo",
			*h.MatchOffset, lido.Content)
	}
}

// TestSearchMatchOffsetDentroDosLimites e a propriedade que vale para TODO hit,
// e o controle do teste acima: um offset fora da faixa do arquivo nao e
// utilizavel, e devolver um numero grande seria tao ruim quanto devolver zero.
func TestSearchMatchOffsetDentroDosLimites(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md":     "alfa beta gama\n",
		"b.md":     strings.Repeat("beta\n", 200) + "alfa no fim\n",
		"sub/c.md": "# Titulo\n\nalfa numa subpasta\n",
	})

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "alfa", Limit: 20})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) != 3 {
		t.Fatalf("hits = %d, quer 3 — a fixture nao monta o teste", len(res.Hits))
	}
	for _, h := range res.Hits {
		if h.Snippet == "" {
			t.Fatalf("%s: trecho vazio", h.Path)
		}
		if h.MatchOffset == nil {
			t.Fatalf("%s: hit COM trecho veio sem MatchOffset", h.Path)
		}
		// Le um byte no offset: se ele estiver fora da faixa, ReadNote falha, e
		// essa e a asserção — nao um numero escrito a mao.
		if _, err := svc.ReadNote(context.Background(), service.ReadRequest{
			Path: h.Path, Offset: h.MatchOffset, MaxBytes: 1,
		}); err != nil {
			t.Fatalf("%s: MatchOffset=%d nao e legivel: %v", h.Path, *h.MatchOffset, err)
		}
	}
}

// TestSearchMatchOffsetAusenteQuandoNaoHaTrecho: "ausente" e "zero" tem de ser
// distinguiveis, porque zero e um offset VALIDO — o inicio do arquivo.
//
// O cenario e real: a nota sai do disco entre o indice e o recorte, o hit fica
// na pagina com trecho vazio, e e ai que o campo nao pode virar zero.
func TestSearchMatchOffsetAusenteQuandoNaoHaTrecho(t *testing.T) {
	svc, v, _, _ := createSearchService(t, map[string]string{
		"some.md":  "o termo xifopago mora aqui\n",
		"outra.md": "nota qualquer para o cofre nao ter uma nota so\n",
	})

	// Depois de indexada, antes de buscar: o recorte le o disco, o indice nao.
	if err := os.Remove(filepath.Join(v.Root(), "some.md")); err != nil {
		t.Fatalf("removendo a nota: %v", err)
	}

	res, err := svc.Search(context.Background(), service.SearchOptions{Query: "xifopago", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) == 0 {
		t.Fatal("a nota sumiu do DISCO, nao do indice: o hit tinha de continuar na pagina")
	}

	h := res.Hits[0]
	if h.Snippet != "" {
		t.Fatalf("a fixture nao monta o teste: o hit tem trecho %q, "+
			"e este teste precisa do caso SEM trecho", h.Snippet)
	}
	if h.MatchOffset != nil {
		t.Fatalf("hit SEM trecho veio com match_offset=%d; "+
			"sem ocorrencia localizada o campo tem de ficar ausente", *h.MatchOffset)
	}
}
