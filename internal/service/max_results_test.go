package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jonyd/gobsidian/internal/service"
)

func TestMaxResultsClampaLimit(t *testing.T) {
	arquivos := map[string]string{}
	for i := 0; i < 30; i++ {
		arquivos[fmt.Sprintf("n%02d.md", i)] = "palavra comum nesta nota\n"
	}
	_, v, idx, inv := createSearchService(t, arquivos)

	svc := service.New(v, idx, inv, nil, service.Options{MaxResults: 5})

	res, err := svc.Search(context.Background(), service.SearchOptions{
		Query: "comum",
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) != 5 && len(res.Results) != 5 {
		t.Fatalf("hits = %d, quer 5 (limit=100 clampado por MaxResults=5)", len(res.Hits))
	}

	semTeto := service.New(v, idx, inv, nil, service.Options{})
	res2, err := semTeto.Search(context.Background(), service.SearchOptions{
		Query: "comum", Limit: 100,
	})
	if err != nil {
		t.Fatalf("Search sem teto: %v", err)
	}
	n2 := len(res2.Hits)
	if n2 == 0 {
		n2 = len(res2.Results)
	}
	if n2 != 30 {
		t.Fatalf("sem teto: hits = %d, quer 30", n2)
	}

	if res.LimitEfetivo != 5 {
		t.Fatalf("LimitEfetivo = %d, quer 5 — o clamp nao foi anunciado na resposta",
			res.LimitEfetivo)
	}
}

func TestSnippetCharsClampAnunciado(t *testing.T) {
	svc, _, _, _ := createSearchService(t, map[string]string{
		"a.md": "palavra comum e muito texto depois dela para o trecho ter de onde sair\n",
	})
	res, err := svc.Search(context.Background(), service.SearchOptions{
		Query: "comum", Limit: 5, SnippetChars: 4000,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.SnippetCharsEfetivo == 0 {
		t.Fatal("a resposta nao diz qual snippet_chars foi usado")
	}
	if res.SnippetCharsEfetivo >= 4000 {
		t.Fatalf("SnippetCharsEfetivo = %d; o teto de MaxSnippetChars nao foi aplicado",
			res.SnippetCharsEfetivo)
	}
}
