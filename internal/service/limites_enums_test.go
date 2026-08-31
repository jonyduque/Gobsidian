package service

import (
	"context"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
)

// TestNoteListAplicaOTetoDeLimit é o achado B4.
//
// `docs/TOOLS.md` declara `"maximum": 500` para o `limit` de `note_list`, e o
// código nunca clampava — `link_graph` clampava, com os números mágicos inline,
// e `note_list` não. Schema que promete um teto e código que não o aplica é a
// mesma classe do M4: o cliente lê o schema para decidir o que pedir.
func TestNoteListAplicaOTetoDeLimit(t *testing.T) {
	root := t.TempDir()
	// 3 notas bastam: o que se testa é o clamp do parâmetro, não a paginação.
	for _, n := range []string{"a.md", "b.md", "c.md"} {
		writeFile(t, root, n, "---\ntitle: "+n+"\n---\n\ntexto\n")
	}
	svc := newTestService(t, root)

	casos := []struct {
		nome    string
		pedido  int
		efetivo int
	}{
		{"acima do teto", 100000, LimiteTeto},
		{"zero usa o padrao", 0, LimitePadrao},
		{"negativo usa o padrao", -5, LimitePadrao},
		{"dentro do teto passa", 7, 7},
		{"exatamente o teto passa", LimiteTeto, LimiteTeto},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := ComTeto(c.pedido); got != c.efetivo {
				t.Errorf("ComTeto(%d) = %d, queria %d", c.pedido, got, c.efetivo)
			}
		})
	}

	// E o caminho real: um pedido absurdo não pode chegar ao índice como veio.
	if _, err := svc.ListNotes(context.Background(), ListRequest{
		Query: index.Query{Limit: 100000},
	}); err != nil {
		t.Fatalf("ListNotes com limit absurdo: %v", err)
	}
}

// TestEnumInvalidoNaoViraSilencio é a outra metade do B4.
//
// `tag_mode`, `sort`, `order` e `direction` caíam no `default` de um switch e o
// pedido virava outra coisa **sem aviso**. O modelo do outro lado lê o enum do
// schema para decidir o que pedir; se um valor inválido responde como se fosse
// válido, ele não tem como saber que o pedido não fez o que dizia — é o mesmo
// defeito do `fields` de `note_list` que esta base já pagou.
func TestEnumInvalidoNaoViraSilencio(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "---\ntitle: A\ntags: [x]\n---\n\ntexto\n")
	svc := newTestService(t, root)
	ctx := context.Background()

	casos := []struct {
		nome string
		erro func() error
	}{
		{"note_list tag_mode", func() error {
			_, err := svc.ListNotes(ctx, ListRequest{Query: index.Query{TagMode: "qualquer"}})
			return err
		}},
		{"note_list sort", func() error {
			_, err := svc.ListNotes(ctx, ListRequest{Query: index.Query{Sort: "relevancia"}})
			return err
		}},
		{"note_list order", func() error {
			_, err := svc.ListNotes(ctx, ListRequest{Query: index.Query{Order: "aleatoria"}})
			return err
		}},
		{"link_graph direction", func() error {
			_, err := svc.LinkGraph(ctx, GraphRequest{Path: "a.md", Direction: "lateral"})
			return err
		}},
		{"tag_list sort", func() error {
			_, err := svc.TagList(ctx, TagRequest{Sort: "frequencia"})
			return err
		}},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			err := c.erro()
			if err == nil {
				t.Fatal("valor invalido foi ACEITO e caiu no padrao em silencio")
			}
			if got := CodeOf(err); got != CodeInvalidArgument {
				t.Errorf("codigo = %s, queria %s\nerro: %v", got, CodeInvalidArgument, err)
			}
		})
	}

	// O contrapeso: o valor VÁLIDO continua passando, e o vazio continua
	// significando "não informado". Sem isto, "corrigir" o B4 rejeitando tudo
	// passaria nos casos acima.
	if _, err := svc.ListNotes(ctx, ListRequest{Query: index.Query{TagMode: "any", Sort: "title", Order: "desc"}}); err != nil {
		t.Errorf("valores validos foram recusados: %v", err)
	}
	if _, err := svc.ListNotes(ctx, ListRequest{Query: index.Query{}}); err != nil {
		t.Errorf("campos omitidos foram recusados: %v", err)
	}
}

// TestPatchModeInvalidoNaoEErroInterno fecha o terceiro pedaço do B4.
//
// `note_patch` com `mode` inválido respondia `INTERNAL`, que diz ao host que o
// SERVIDOR quebrou — e host que lê INTERNAL tenta de novo, em vez de corrigir o
// pedido. O erro é do cliente, e o código tem de dizer isso.
func TestPatchModeInvalidoNaoEErroInterno(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "---\ntitle: A\n---\n\n# H\n\ntexto\n")
	svc := newTestService(t, root)

	_, err := svc.PatchNote(context.Background(), PatchNoteRequest{
		Path: "a.md", Mode: "modo_que_nao_existe", Content: "x",
	})
	if err == nil {
		t.Fatal("modo invalido foi aceito")
	}
	if got := CodeOf(err); got != CodeInvalidArgument {
		t.Errorf("codigo = %s, queria %s: INTERNAL manda o host tentar de novo\nerro: %v",
			got, CodeInvalidArgument, err)
	}
}
