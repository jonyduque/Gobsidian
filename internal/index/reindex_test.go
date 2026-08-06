package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// TestReresolucaoDirigidaCobreAliases prova a decisao fechada da Task 86:
// aliases contam. Uma nota citada SO por alias tem de ter o citante
// reprocessado quando o alias aparece — nao so quando o nome de arquivo
// aparece.
//
// Origem.md cita [[STJ]], que nao resolve (nenhum arquivo se chama STJ e
// nenhuma nota declara esse alias ainda). Cria-se Tribunal.md com
// "aliases: [STJ]" via Replace — o caminho dirigido do watcher — e o link
// tem de passar a resolver. Se chavesDaNota parar de percorrer os aliases
// (a mutacao desta prova), a chave "stj" nunca entra em citantesPorNome do
// lado da nota nova, reprocessLinksDirigidoLocked nao acha Origem.md, e o
// link continua quebrado.
func TestReresolucaoDirigidaCobreAliases(t *testing.T) {
	root := t.TempDir()
	writeFileHelper(t, root, "Origem.md", "# Origem\n\n[[STJ]]\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ctx := context.Background()
	ix := New()
	if err := ix.Build(ctx, v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	origem, ok := ix.Get("Origem.md")
	if !ok {
		t.Fatal("Origem.md ausente")
	}
	if len(origem.Links) != 1 || origem.Links[0].State != LinkTargetMissing {
		t.Fatalf("estado inicial: link[0] = %+v, quer LinkTargetMissing", origem.Links)
	}

	writeFileHelper(t, root, "Tribunal.md", "---\naliases: [STJ]\n---\n# Tribunal\n")
	if err := ix.Replace(ctx, v, "Tribunal.md"); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	origem, ok = ix.Get("Origem.md")
	if !ok {
		t.Fatal("Origem.md sumiu depois do Replace de Tribunal.md")
	}
	if len(origem.Links) != 1 {
		t.Fatalf("Origem.md tem %d links, quer 1", len(origem.Links))
	}
	got := origem.Links[0]
	if got.State != LinkOK || got.Resolved != "Tribunal.md" || got.Via != ViaAlias {
		t.Errorf("link [[STJ]] depois de criar Tribunal.md com alias STJ = %+v, "+
			"quer State=LinkOK Resolved=Tribunal.md Via=ViaAlias — "+
			"a nota nova cita por alias, e a re-resolucao dirigida tem de achar quem cita esse alias",
			got)
	}
}

// TestReresolucaoDirigidaIgualAGlobal e o diferencial que decide a Task 86:
// para a sequencia criar / renomear / apagar / criar de novo com alias, o
// indice que so passou pela re-resolucao DIRIGIDA (idx1, via Replace/
// Remove/MoveNote normais) tem de ficar IDENTICO ao indice que teve, depois
// de cada evento, uma passada GLOBAL forcada por cima (idx2, via
// resolveAllLinks + buildBacklinks — as mesmas funcoes que Build e
// LoadIndexCache usam para a resolucao inicial, nao uma reimplementacao).
//
// O caminho global fica so aqui, como referencia — nao volta a rodar por
// evento no produto (ver o comentario de reprocessLinksDirigidoLocked em
// update.go). Se a re-resolucao dirigida deixar de encontrar um citante —
// por exemplo porque a chave de escrita em citantesPorNome divergiu da
// chave de leitura — idx2 corrige sozinho na proxima passada global e idx1
// fica pra tras: e exatamente essa divergencia que o comparativo campo a
// campo abaixo pega.
func TestReresolucaoDirigidaIgualAGlobal(t *testing.T) {
	root1, root2 := t.TempDir(), t.TempDir()
	ctx := context.Background()

	inicial := map[string]string{
		"Original.md": "# Original\n",
		"Citante.md":  "# Citante\n\n[[Original]]\n[[Apelido]]\n",
	}
	for path, content := range inicial {
		writeFileHelper(t, root1, path, content)
		writeFileHelper(t, root2, path, content)
	}

	v1, err := vault.New(root1)
	if err != nil {
		t.Fatalf("vault.New v1: %v", err)
	}
	v2, err := vault.New(root2)
	if err != nil {
		t.Fatalf("vault.New v2: %v", err)
	}

	idx1, idx2 := New(), New()
	if err := idx1.Build(ctx, v1); err != nil {
		t.Fatalf("idx1.Build: %v", err)
	}
	if err := idx2.Build(ctx, v2); err != nil {
		t.Fatalf("idx2.Build: %v", err)
	}

	// aplicar roda a MESMA mutacao nos dois vaults e nos dois indices — idx1
	// so recebe o que a funcao de producao (Replace/Remove/MoveNote) ja faz
	// sozinha; idx2 recebe o mesmo e ainda leva uma passada global forcada
	// por cima, como oraculo.
	aplicar := func(fn func(v *vault.Vault, ix *Index)) {
		t.Helper()
		fn(v1, idx1)
		fn(v2, idx2)
		idx2.resolveAllLinks()
		idx2.buildBacklinks()
	}

	// 1. Criar: nova nota que cita a existente.
	aplicar(func(v *vault.Vault, ix *Index) {
		writeFileHelper(t, v.Root(), "Nova.md", "# Nova\n\n[[Original]]\n")
		if err := ix.Replace(ctx, v, "Nova.md"); err != nil {
			t.Fatalf("Replace Nova.md: %v", err)
		}
	})

	// 2. Renomear: Original.md -> Renomeada.md.
	aplicar(func(v *vault.Vault, ix *Index) {
		if err := os.Rename(filepath.Join(v.Root(), "Original.md"), filepath.Join(v.Root(), "Renomeada.md")); err != nil {
			t.Fatalf("os.Rename: %v", err)
		}
		ix.MoveNote(v, "Original.md", "Renomeada.md")
	})

	// 3. Apagar: Renomeada.md some do cofre. Os links por nome que
	// apontavam pra ela (Nova.md e Citante.md, via "Original") ficam
	// quebrados — nao ha alias "Original" declarado em lugar nenhum.
	aplicar(func(v *vault.Vault, ix *Index) {
		if err := os.Remove(filepath.Join(v.Root(), "Renomeada.md")); err != nil {
			t.Fatalf("os.Remove: %v", err)
		}
		ix.Remove("Renomeada.md")
	})

	// 4. Criar de novo, agora com o alias que Citante.md ja citava
	// (Apelido) desde o inicio — o link [[Apelido]] estava quebrado e passa
	// a resolver so por causa do alias, sem nenhum nome de arquivo em comum.
	aplicar(func(v *vault.Vault, ix *Index) {
		writeFileHelper(t, v.Root(), "Renomeada.md", "---\naliases: [Apelido]\n---\n# Renomeada de volta\n")
		if err := ix.Replace(ctx, v, "Renomeada.md"); err != nil {
			t.Fatalf("Replace Renomeada.md (recriada): %v", err)
		}
	})

	compararIndices(t, idx1, idx2)
}

// compararIndices confere, campo a campo, que os dois indices concordam:
// mesmas notas, mesma resolucao de cada link (Resolved/Via/State — nao so
// se resolveu, mas PRA ONDE), e mesmos backlinks em cada caminho conhecido.
// Comparar so o conjunto de caminhos deixaria passar exatamente o defeito
// que este teste existe pra pegar: link com state=ok apontando pra nota
// errada, ou backlink obsoleto que sobrevive.
func compararIndices(t *testing.T, idx1, idx2 *Index) {
	t.Helper()

	todos := make(map[vault.CanonicalPath]bool)
	for _, p := range idx1.Paths() {
		todos[p] = true
	}
	for _, p := range idx2.Paths() {
		todos[p] = true
	}

	for path := range todos {
		n1, ok1 := idx1.Get(path)
		n2, ok2 := idx2.Get(path)
		if ok1 != ok2 {
			t.Errorf("%s: presenca em notes diverge — dirigido=%v, global=%v", path, ok1, ok2)
			continue
		}
		if !ok1 {
			// Pode ser anexo, ou pode nao existir em nenhum dos dois — Get
			// so resolve nota, e a comparacao de link nao se aplica.
			continue
		}

		if len(n1.Links) != len(n2.Links) {
			t.Errorf("%s: %d links no dirigido, %d no global", path, len(n1.Links), len(n2.Links))
			continue
		}
		for i := range n1.Links {
			l1, l2 := n1.Links[i], n2.Links[i]
			if l1.Resolved != l2.Resolved || l1.Via != l2.Via || l1.State != l2.State {
				t.Errorf("%s: link %d (alvo %q) diverge — dirigido={Resolved:%q Via:%v State:%v}, "+
					"global={Resolved:%q Via:%v State:%v}",
					path, i, l1.Target, l1.Resolved, l1.Via, l1.State, l2.Resolved, l2.Via, l2.State)
			}
		}
	}

	for path := range todos {
		bl1 := idx1.Backlinks(path)
		bl2 := idx2.Backlinks(path)
		if len(bl1) != len(bl2) {
			t.Errorf("%s: %d backlinks no dirigido, %d no global", path, len(bl1), len(bl2))
			continue
		}
		vistos := make(map[vault.CanonicalPath]int)
		for _, bl := range bl1 {
			vistos[bl.From]++
		}
		for _, bl := range bl2 {
			vistos[bl.From]--
		}
		for from, delta := range vistos {
			if delta != 0 {
				t.Errorf("%s: backlink de %s presente em um indice e nao no outro (delta=%d)", path, from, delta)
			}
		}
	}
}
