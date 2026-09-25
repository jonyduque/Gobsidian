package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/index"
)

// docs/TOOLS.md promete, nas convencoes gerais: "Respostas truncadas trazem
// truncated: true e total." Ate 2026-09-25 so vault_search, note_read e
// note_outline cumpriam. note_list e vault_broken_links traziam total e nao
// truncated, e link_graph nao trazia nenhum dos dois: um grafo cortado pelo
// limit era indistinguivel de um grafo completo.

func TestListNotesDizQueCortou(t *testing.T) {
	root := t.TempDir()
	for i := range 3 {
		writeFile(t, root, fmt.Sprintf("n%d.md", i), "# n\n")
	}
	svc := newTestService(t, root)

	casos := []struct {
		limit, offset int
		quer          bool
	}{
		{2, 0, true},
		{3, 0, false},
		{2, 1, false},
		{2, 5, false},
		{1, 1, true},
	}
	for _, c := range casos {
		res, err := svc.ListNotes(context.Background(), ListRequest{Query: index.Query{Limit: c.limit, Offset: c.offset}})
		if err != nil {
			t.Fatalf("ListNotes: %v", err)
		}
		if res.Total != 3 || res.Truncated != c.quer {
			t.Errorf("limit=%d offset=%d: total=%d truncated=%v, quer 3 e %v", c.limit, c.offset, res.Total, res.Truncated, c.quer)
		}
	}
}

func TestBrokenLinksDizQueCortou(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "[[x1]] [[x2]] [[x3]]\n")
	svc := newTestService(t, root)

	casos := []struct {
		limit, offset int
		quer          bool
	}{
		{2, 0, true},
		{3, 0, false},
		{2, 1, false},
		{1, 1, true},
	}
	for _, c := range casos {
		res, err := svc.BrokenLinks(context.Background(), BrokenLinksRequest{Limit: c.limit, Offset: c.offset})
		if err != nil {
			t.Fatalf("BrokenLinks: %v", err)
		}
		if res.Total != 3 || res.Truncated != c.quer {
			t.Errorf("limit=%d offset=%d: total=%d truncated=%v, quer 3 e %v", c.limit, c.offset, res.Total, res.Truncated, c.quer)
		}
	}
}

func TestLinkGraphDizQueCortou(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "centro.md", "[[a]] [[b]] [[c]] [[d]] [[e]]\n")
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		writeFile(t, root, n+".md", "# "+n+"\n")
	}
	// Triangulo: C e vizinho de A e de B. Com limit 3 o grafo inteiro cabe,
	// mas B ve C ja na fila e a conta len(nodes)+len(fila) recusa C de novo —
	// uma recusa que NAO e corte, porque C entra do mesmo jeito.
	writeFile(t, root, "ta.md", "[[tb]] [[tc]]\n")
	writeFile(t, root, "tb.md", "[[tc]]\n")
	writeFile(t, root, "tc.md", "# tc\n")
	svc := newTestService(t, root)

	casos := []struct {
		path  string
		limit int
		depth int
		quer  bool
	}{
		{"centro.md", 3, 1, true},
		{"centro.md", 6, 1, false},
		{"centro.md", 100, 1, false},
		{"ta.md", 3, 2, false},
		{"ta.md", 2, 2, true},
	}
	for _, c := range casos {
		res, err := svc.LinkGraph(context.Background(), GraphRequest{Path: c.path, Direction: "outgoing", Depth: c.depth, Limit: c.limit})
		if err != nil {
			t.Fatalf("LinkGraph: %v", err)
		}
		if res.Truncated != c.quer {
			t.Errorf("%s limit=%d depth=%d: truncated=%v com %d nos, quer %v", c.path, c.limit, c.depth, res.Truncated, len(res.Nodes), c.quer)
		}
	}

	// O outro jeito de cortar: o no de link quebrado entra no grafo sem passar
	// pela fila, enche o limit, e o laco para com "a" ainda na fila — sem
	// nenhuma recusa registrada.
	writeFile(t, root, "q.md", "[[a]] [[nada1]] [[nada2]]\n")
	svc = newTestService(t, root)
	res, err := svc.LinkGraph(context.Background(), GraphRequest{Path: "q.md", Direction: "outgoing", Depth: 1, Limit: 3, IncludeBroken: true})
	if err != nil {
		t.Fatalf("LinkGraph: %v", err)
	}
	if !res.Truncated {
		t.Errorf("q.md com link quebrado enchendo o limit: truncated=false com nos %+v, quer true", res.Nodes)
	}
}
