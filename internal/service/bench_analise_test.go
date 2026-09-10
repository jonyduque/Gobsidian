package service_test

import (
	"context"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/index"
	"github.com/jonyduque/Gobsidian/internal/service"
)

// Benchmarks das tools que a analise de 2026-09-02 quer mexer e que nao tinham
// medida nenhuma: link_graph, tag_list (plano e hierarquico) e note_list com
// filtro de tag. Sem numero antes, nao ha como dizer que a mudanca nao piorou.

func BenchmarkLinkGraphBothDepth2(b *testing.B) {
	svc := benchServico(b)
	req := service.GraphRequest{Path: "Nota_1", Direction: "both", Depth: 2, Limit: 500}
	res, err := svc.LinkGraph(context.Background(), req)
	if err != nil {
		b.Fatalf("LinkGraph: %v", err)
	}
	if len(res.Nodes) < 2 {
		b.Fatalf("grafo com %d nos; o cofre de bench deveria ligar Nota_1", len(res.Nodes))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := svc.LinkGraph(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTagListPlano(b *testing.B) {
	svc := benchServico(b)
	req := service.TagRequest{Sort: "count"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		res, err := svc.TagList(context.Background(), req)
		if err != nil || len(res.Tags) == 0 {
			b.Fatalf("TagList: %v, %d tags", err, len(res.Tags))
		}
	}
}

func BenchmarkTagListHierarquico(b *testing.B) {
	svc := benchServico(b)
	req := service.TagRequest{Sort: "count", Hierarchical: true}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		res, err := svc.TagList(context.Background(), req)
		if err != nil || len(res.Tags) == 0 {
			b.Fatalf("TagList: %v, %d tags", err, len(res.Tags))
		}
	}
}

func BenchmarkNoteListPorTag(b *testing.B) {
	svc := benchServico(b)
	req := service.ListRequest{Query: index.Query{Tags: []string{"golang"}, TagMode: "any", Sort: "title", Limit: 100}}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		res, err := svc.ListNotes(context.Background(), req)
		if err != nil || len(res.Notes) == 0 {
			b.Fatalf("ListNotes: %v, %d notas", err, len(res.Notes))
		}
	}
}
