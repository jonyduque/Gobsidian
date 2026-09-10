package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/index"
	"github.com/jonyduque/Gobsidian/internal/vault"
)

// TestAliasCollisions cobre Index.AliasCollisions, que estava a 0 %.
//
// O campo alias_collisions de vault_stats era um zero literal no codigo antes
// de existir esta conta — aparecia na resposta e mentia sempre. Um contador a
// 0 % de cobertura e a mesma mentira com mais passos, entao o teste afirma o
// NUMERO, e afirma-o num cofre montado para separar as tres coisas que o
// contador poderia estar contando por engano:
//
//   - aliases duplicados (o certo): STJ em duas notas, TRF em tres -> 2;
//   - NOTAS envolvidas em colisao: seriam 5;
//   - aliases declarados: seriam 3, porque STF esta em uma nota so.
//
// A nota com alias exclusivo e o controle: sem ela, "conta alias" e "conta
// alias duplicado" dariam o mesmo numero e o teste nao distinguiria os dois.
func TestAliasCollisions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "---\naliases: [STJ]\n---\n# Nota A\n")
	writeFile(t, root, "b.md", "---\naliases: [STJ]\n---\n# Nota B\n")
	writeFile(t, root, "c.md", "---\naliases: [TRF]\n---\n# Nota C\n")
	writeFile(t, root, "d.md", "---\naliases: [TRF]\n---\n# Nota D\n")
	writeFile(t, root, "e.md", "---\naliases: [TRF]\n---\n# Nota E\n")
	writeFile(t, root, "so-dela.md", "---\naliases: [STF]\n---\n# Nota sozinha\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if got := idx.AliasCollisions(); got != 2 {
		t.Errorf("AliasCollisions() = %d, quer 2 (STJ em 2 notas, TRF em 3, STF em 1). "+
			"5 seria contar notas em colisao; 3 seria contar aliases declarados", got)
	}
}

// TestAliasCollisionsZeroSemDuplicata e a outra metade: um contador que
// devolvesse sempre o numero de aliases passaria no teste acima com o cofre
// certo e mentiria aqui. Zero e uma resposta que precisa ser possivel.
func TestAliasCollisionsZeroSemDuplicata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "---\naliases: [STJ, Tribunal]\n---\n# Nota A\n")
	writeFile(t, root, "b.md", "---\naliases: [STF]\n---\n# Nota B\n")
	writeFile(t, root, "c.md", "# Nota C sem alias\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if got := idx.AliasCollisions(); got != 0 {
		t.Errorf("AliasCollisions() = %d, quer 0 — nenhum alias deste cofre e declarado por duas notas", got)
	}
}

func TestAliasSurvivesReplaceAndRemove(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.md", "---\naliases: [STJ]\n---\n# Nota A\n")
	writeFile(t, root, "b.md", "# Nota B\n\n[[STJ]]\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()
	ctx := context.Background()
	if err := idx.Build(ctx, v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// 1. After Build, b.md link [[STJ]] resolves to a.md with state LinkOK
	bNote, ok := idx.Get("b.md")
	if !ok || len(bNote.Links) != 1 {
		t.Fatalf("b.md links count = %d, want 1 (ok=%v)", len(bNote.Links), ok)
	}
	if bNote.Links[0].Resolved != "a.md" || bNote.Links[0].State != index.LinkOK {
		t.Fatalf("after Build: resolved=%q, state=%v; want 'a.md', LinkOK", bNote.Links[0].Resolved, bNote.Links[0].State)
	}

	// 2. After Replace(a.md), b.md link [[STJ]] should still resolve to a.md
	if err := idx.Replace(ctx, v, "a.md"); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	bNote, ok = idx.Get("b.md")
	if !ok || len(bNote.Links) != 1 {
		t.Fatalf("b.md links count = %d, want 1 (ok=%v)", len(bNote.Links), ok)
	}
	if bNote.Links[0].Resolved != "a.md" || bNote.Links[0].State != index.LinkOK {
		t.Fatalf("after Replace: resolved=%q, state=%v; want 'a.md', LinkOK", bNote.Links[0].Resolved, bNote.Links[0].State)
	}

	// 3. After Remove(a.md), b.md link [[STJ]] must resolve to "" with state LinkTargetMissing
	if err := os.Remove(filepath.Join(root, "a.md")); err != nil {
		t.Fatalf("Remove file: %v", err)
	}
	idx.Remove("a.md")

	bNote, ok = idx.Get("b.md")
	if !ok || len(bNote.Links) != 1 {
		t.Fatalf("b.md links count = %d, want 1 (ok=%v)", len(bNote.Links), ok)
	}
	if bNote.Links[0].Resolved != "" || bNote.Links[0].State != index.LinkTargetMissing {
		t.Fatalf("after Remove: resolved=%q, state=%v; want '', LinkTargetMissing", bNote.Links[0].Resolved, bNote.Links[0].State)
	}
}
