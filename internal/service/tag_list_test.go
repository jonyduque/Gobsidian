package service

import (
	"context"
	"testing"
)

func cofreDeTags(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "a.md", "---\ntags: [projeto/ativo, projeto/ativo/2026, zzz]\n---\n\ntexto\n")
	writeFile(t, root, "b.md", "---\ntags: [projeto/ativo, projeto/ativo/2026, zzz]\n---\n\ntexto\n")
	writeFile(t, root, "c.md", "---\ntags: [projeto/ativo, zzz]\n---\n\ntexto\n")
	writeFile(t, root, "d.md", "---\ntags: [zzz]\n---\n\ntexto\n")
	return newTestService(t, root)
}

func TestTagListOrdenacao(t *testing.T) {
	svc := cofreDeTags(t)

	porNome, err := svc.TagList(context.Background(), TagRequest{Sort: "name", MinCount: 1})
	if err != nil {
		t.Fatalf("TagList(name): %v", err)
	}
	porContagem, err := svc.TagList(context.Background(), TagRequest{Sort: "count", MinCount: 1})
	if err != nil {
		t.Fatalf("TagList(count): %v", err)
	}
	if len(porNome.Tags) == 0 || len(porNome.Tags) != len(porContagem.Tags) {
		t.Fatalf("listas de tamanhos diferentes: %d e %d", len(porNome.Tags), len(porContagem.Tags))
	}

	for i := 1; i < len(porNome.Tags); i++ {
		if porNome.Tags[i-1].Tag > porNome.Tags[i].Tag {
			t.Fatalf("sort=name fora de ordem em %d: %q depois de %q",
				i, porNome.Tags[i].Tag, porNome.Tags[i-1].Tag)
		}
	}

	for i := 1; i < len(porContagem.Tags); i++ {
		a, b := porContagem.Tags[i-1], porContagem.Tags[i]
		if a.Count < b.Count {
			t.Fatalf("sort=count fora de ordem em %d: %d depois de %d", i, b.Count, a.Count)
		}
		if a.Count == b.Count && a.Tag > b.Tag {
			t.Fatalf("empate em %d nao desempatou por nome: %q depois de %q", i, b.Tag, a.Tag)
		}
	}

	if porNome.Tags[0].Tag == porContagem.Tags[0].Tag &&
		porNome.Tags[len(porNome.Tags)-1].Tag == porContagem.Tags[len(porContagem.Tags)-1].Tag {
		t.Fatal("sort=name e sort=count devolveram a mesma ordem; o campo foi ignorado")
	}
}

func TestTagListHierarquico(t *testing.T) {
	svc := cofreDeTags(t)

	plana, err := svc.TagList(context.Background(), TagRequest{MinCount: 1})
	if err != nil {
		t.Fatalf("TagList plana: %v", err)
	}
	arvore, err := svc.TagList(context.Background(), TagRequest{MinCount: 1, Hierarchical: true})
	if err != nil {
		t.Fatalf("TagList hierarquica: %v", err)
	}

	temRaizProjeto := false
	for _, n := range arvore.Tags {
		if n.Tag == "projeto" {
			temRaizProjeto = true
			if n.Count != 3 {
				t.Errorf("no raiz projeto tem Count=%d, quer 3 (incluindo filhos)", n.Count)
			}
		}
	}
	if !temRaizProjeto {
		t.Fatal("hierarchical=true devolveu lista plana: nao ha no raiz 'projeto'")
	}
	if len(arvore.Tags) == len(plana.Tags) {
		t.Fatal("hierarchical=true devolveu o mesmo numero de entradas que a lista plana")
	}
}
