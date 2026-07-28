package index_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func TestResolutionOrder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Civil/PONTO 03.md", "# Ponto 3\n\n## Cap 1\n\ntexto ^blk1\n")
	writeFile(t, root, "Penal/PONTO 03.md", "# Outro\n")
	writeFile(t, root, "P3.md", "# Arquivo P3\n")
	writeFile(t, root, "Apelidada.md", "---\naliases: [P3, Terceiro]\n---\n# Apelidada\n")
	writeFile(t, root, "Anexos/diagrama.png", "\x89PNG")
	writeFile(t, root, "Origem.md", strings.Join([]string{
		"# Origem",
		"[[Civil/PONTO 03]]",       // 0: caminho explicito
		"[[P3]]",                   // 1: nome de arquivo vence alias
		"[[Terceiro]]",             // 2: alias, sem arquivo homonimo
		"![[diagrama.png]]",        // 3: anexo
		"[[Civil/PONTO 03#Cap 1]]", // 4: ancora existente
		"[[Civil/PONTO 03#Cap 9]]", // 5: ancora inexistente
		"[[Civil/PONTO 03#^blk1]]", // 6: bloco existente
		"[[Nao Existe]]",           // 7: alvo inexistente
	}, "\n\n")+"\n")

	v, _ := vault.New(root)
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	note, ok := idx.Get("Origem.md")
	if !ok {
		t.Fatal("Origem.md ausente")
	}
	if len(note.Links) != 8 {
		t.Fatalf("links = %d, quer 8: %+v", len(note.Links), note.Links)
	}

	want := []struct {
		resolved vault.CanonicalPath
		via      index.ResolveVia
		state    index.LinkState
	}{
		{"Civil/PONTO 03.md", index.ViaPath, index.LinkOK},
		{"P3.md", index.ViaName, index.LinkOK},
		{"Apelidada.md", index.ViaAlias, index.LinkOK},
		{"Anexos/diagrama.png", index.ViaAsset, index.LinkOK},
		{"Civil/PONTO 03.md", index.ViaPath, index.LinkOK},
		{"Civil/PONTO 03.md", index.ViaPath, index.LinkAnchorMissing},
		{"Civil/PONTO 03.md", index.ViaPath, index.LinkOK},
		{"", index.ViaNone, index.LinkTargetMissing},
	}

	for i, w := range want {
		got := note.Links[i]
		if got.Resolved != w.resolved {
			t.Errorf("link %d Resolved = %q, quer %q", i, got.Resolved, w.resolved)
		}
		if got.Via != w.via {
			t.Errorf("link %d Via = %v, quer %v", i, got.Via, w.via)
		}
		if got.State != w.state {
			t.Errorf("link %d State = %v, quer %v", i, got.State, w.state)
		}
	}
}

func TestNameCollisionPicksNearest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Civil/A.md", "# A civil\n")
	writeFile(t, root, "Penal/A.md", "# A penal\n")
	writeFile(t, root, "Civil/Origem.md", "# O\n\n[[A]]\n")

	v, _ := vault.New(root)
	idx := index.New()
	_ = idx.Build(context.Background(), v)

	note, _ := idx.Get("Civil/Origem.md")
	if note.Links[0].Resolved != "Civil/A.md" {
		t.Errorf("Resolved = %q, quer Civil/A.md — o mais proximo da origem",
			note.Links[0].Resolved)
	}
}

func TestResolvePathCaseInsensitiveAndAmbiguous(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Civil/PONTO 03.md", "# A\n")

	v, _ := vault.New(root)
	idx := index.New()
	_ = idx.Build(context.Background(), v)

	// Casing divergente resolve.
	got, err := idx.ResolvePath("civil/ponto 03.md")
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	if got != "Civil/PONTO 03.md" {
		t.Errorf("ResolvePath = %q, quer a grafia do disco", got)
	}

	// Caminho inexistente falha, nao adivinha.
	if _, err := idx.ResolvePath("Civil/Nada.md"); err == nil {
		t.Error("ResolvePath de caminho inexistente deveria falhar")
	}
}
