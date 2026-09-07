package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cofreQuebrado monta o cofre dos casos de BrokenLinks: uma nota com os cinco
// tipos de referencia que interessam, uma nota em subpasta e o alvo que existe.
func cofreQuebrado(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "a.md", strings.Join([]string{
		"# Topo",
		"",
		"[[b]]",
		"[[nada]]",
		"[[b#Nada]]",
		"[x](#Topo)",
		"[s](https://ex.com)",
		"",
	}, "\n"))
	writeFile(t, root, "b.md", "# Sec\n\ntexto\n")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0755); err != nil {
		t.Fatalf("MkdirAll sub: %v", err)
	}
	writeFile(t, root, filepath.Join("sub", "c.md"), "[[outra]]\n")
	return root
}

// TestBrokenLinksListaOQueVaultStatsSoConta cobre o contrato inteiro da tool:
// o que entra, o que fica de fora, a ordem e a paginacao.
func TestBrokenLinksListaOQueVaultStatsSoConta(t *testing.T) {
	svc := newTestService(t, cofreQuebrado(t))
	ctx := context.Background()

	t.Run("sem filtro", func(t *testing.T) {
		res, err := svc.BrokenLinks(ctx, BrokenLinksRequest{})
		if err != nil {
			t.Fatalf("BrokenLinks: %v", err)
		}
		if res.Total != 3 {
			t.Fatalf("Total = %d, queria 3: %+v", res.Total, res.Links)
		}
		// A ordem e o contrato: Source e depois a posicao no corpo.
		querido := []struct{ source, target, state string }{
			{"a.md", "nada", "target_missing"},
			{"a.md", "b", "anchor_missing"},
			{"sub/c.md", "outra", "target_missing"},
		}
		if len(res.Links) != len(querido) {
			t.Fatalf("len(Links) = %d, queria %d: %+v", len(res.Links), len(querido), res.Links)
		}
		for i, q := range querido {
			got := res.Links[i]
			if got.Source != q.source || got.Target != q.target || got.State != q.state {
				t.Errorf("Links[%d] = {%s %s %s}, queria {%s %s %s}",
					i, got.Source, got.Target, got.State, q.source, q.target, q.state)
			}
		}
		// O externo e a auto-ancora que resolve nao podem ter entrado. Sem estas
		// duas linhas, "Total == 3" ainda passaria se um filtro errado deixasse
		// a URL entrar e derrubasse outro item por engano.
		for _, l := range res.Links {
			if strings.Contains(l.Target, "ex.com") {
				t.Errorf("link externo entrou na lista: %+v", l)
			}
			if l.Target == "" && l.Anchor == "Topo" {
				t.Errorf("auto-ancora que resolve entrou na lista: %+v", l)
			}
		}
	})

	t.Run("filtro por estado", func(t *testing.T) {
		res, err := svc.BrokenLinks(ctx, BrokenLinksRequest{State: "anchor_missing"})
		if err != nil {
			t.Fatalf("BrokenLinks: %v", err)
		}
		if res.Total != 1 || len(res.Links) != 1 {
			t.Fatalf("Total = %d, len = %d, queria 1 e 1: %+v", res.Total, len(res.Links), res.Links)
		}
		l := res.Links[0]
		if l.State != "anchor_missing" || l.Anchor != "Nada" || l.Target != "b" {
			t.Errorf("Links[0] = %+v, queria State=anchor_missing Anchor=Nada Target=b", l)
		}
		if l.Context == "" {
			t.Errorf("Context vazio: sem ele quem recebe a lista nao acha o link no corpo")
		}
	})

	t.Run("filtro por prefixo", func(t *testing.T) {
		res, err := svc.BrokenLinks(ctx, BrokenLinksRequest{Prefix: "sub"})
		if err != nil {
			t.Fatalf("BrokenLinks: %v", err)
		}
		if res.Total != 1 || len(res.Links) != 1 {
			t.Fatalf("Total = %d, len = %d, queria 1 e 1: %+v", res.Total, len(res.Links), res.Links)
		}
		if res.Links[0].Source != "sub/c.md" || res.Links[0].Target != "outra" {
			t.Errorf("Links[0] = %+v, queria sub/c.md -> outra", res.Links[0])
		}
	})

	t.Run("prefixo que nao casa nada nao e erro", func(t *testing.T) {
		res, err := svc.BrokenLinks(ctx, BrokenLinksRequest{Prefix: "pasta-que-nao-existe"})
		if err != nil {
			t.Fatalf("prefixo sem casamento devolveu erro: %v", err)
		}
		if res.Total != 0 || len(res.Links) != 0 {
			t.Errorf("Total = %d, len = %d, queria 0 e 0", res.Total, len(res.Links))
		}
	})

	t.Run("paginacao", func(t *testing.T) {
		res, err := svc.BrokenLinks(ctx, BrokenLinksRequest{Limit: 1, Offset: 1})
		if err != nil {
			t.Fatalf("BrokenLinks: %v", err)
		}
		// Total e a contagem ANTES de offset/limit, como em note_list.
		if res.Total != 3 {
			t.Errorf("Total = %d, queria 3: o total nao pode encolher com a pagina", res.Total)
		}
		if len(res.Links) != 1 {
			t.Fatalf("len(Links) = %d, queria 1", len(res.Links))
		}
		if res.Links[0].Target != "b" || res.Links[0].State != "anchor_missing" {
			t.Errorf("Links[0] = %+v, queria o SEGUNDO item da ordem (b, anchor_missing)", res.Links[0])
		}
	})

	t.Run("offset alem do fim devolve pagina vazia", func(t *testing.T) {
		res, err := svc.BrokenLinks(ctx, BrokenLinksRequest{Offset: 99})
		if err != nil {
			t.Fatalf("BrokenLinks: %v", err)
		}
		if res.Total != 3 || len(res.Links) != 0 {
			t.Errorf("Total = %d, len = %d, queria 3 e 0", res.Total, len(res.Links))
		}
	})

	t.Run("estado invalido nao vira silencio", func(t *testing.T) {
		_, err := svc.BrokenLinks(ctx, BrokenLinksRequest{State: "quebrado"})
		if err == nil {
			t.Fatal("state invalido foi ACEITO e caiu no padrao em silencio")
		}
		if got := CodeOf(err); got != CodeInvalidArgument {
			t.Errorf("codigo = %s, queria %s\nerro: %v", got, CodeInvalidArgument, err)
		}
	})
}

// TestBrokenLinksAplicaOTetoDeLimit e o mesmo desenho de
// TestNoteListAplicaOTetoDeLimit: sem um cofre ACIMA do teto, o clamp e
// inobservavel — afirmar `len(Links) <= LimiteTeto` sobre tres links da o mesmo
// verde com ou sem a regra.
func TestBrokenLinksAplicaOTetoDeLimit(t *testing.T) {
	root := t.TempDir()
	var corpo strings.Builder
	for i := 0; i <= LimiteTeto; i++ {
		fmt.Fprintf(&corpo, "[[naoexiste-%04d]]\n", i)
	}
	writeFile(t, root, "muitos.md", corpo.String())
	svc := newTestService(t, root)

	res, err := svc.BrokenLinks(context.Background(), BrokenLinksRequest{Limit: 100000})
	if err != nil {
		t.Fatalf("BrokenLinks: %v", err)
	}
	// Controle: sem esta linha, um cofre que nao subisse acima do teto faria a
	// asercao seguinte passar sem exercer o clamp.
	if res.Total != LimiteTeto+1 {
		t.Fatalf("Total = %d, queria %d — o cofre nao tem links quebrados suficientes para o teto ser observavel",
			res.Total, LimiteTeto+1)
	}
	if len(res.Links) != LimiteTeto {
		t.Errorf("len(Links) = %d, queria %d: o limite absurdo chegou ao corte como veio", len(res.Links), LimiteTeto)
	}
}
