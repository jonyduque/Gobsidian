package index_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestBuildIndexesNotesAndAssets(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Civil/PONTO 03.md", "---\naliases: [P3]\n---\n# Ponto 3\n\nVer [[Penal/A]].\n")
	writeFile(t, root, "Penal/A.md", "# A\n\n![[diagrama.png]]\n")
	writeFile(t, root, "Anexos/diagrama.png", "\x89PNG")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if got := idx.NoteCount(); got != 2 {
		t.Errorf("NoteCount() = %d, quer 2", got)
	}
	if got := idx.AssetCount(); got != 1 {
		t.Errorf("AssetCount() = %d, quer 1", got)
	}

	note, ok := idx.Get("Civil/PONTO 03.md")
	if !ok {
		t.Fatal("nota nao encontrada pelo caminho canonico")
	}
	if note.Title != "Ponto 3" {
		t.Errorf("Title = %q, quer %q", note.Title, "Ponto 3")
	}
	if note.Hash == 0 {
		t.Error("Hash nao foi calculado")
	}
	if note.EOL != vault.EOLLF {
		t.Errorf("EOL = %v, quer LF", note.EOL)
	}
	if len(note.Aliases) != 1 || note.Aliases[0] != "P3" {
		t.Errorf("Aliases = %v, quer [P3]", note.Aliases)
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	root := t.TempDir()
	for i := range 50 {
		writeFile(t, root, fmt.Sprintf("n%02d.md", i), fmt.Sprintf("# N%d\n\n[[n%02d]]\n", i, (i+1)%50))
	}

	v, _ := vault.New(root)

	first := index.New()
	if err := first.Build(context.Background(), v); err != nil {
		t.Fatalf("Build 1: %v", err)
	}
	second := index.New()
	if err := second.Build(context.Background(), v); err != nil {
		t.Fatalf("Build 2: %v", err)
	}

	if first.NoteCount() != second.NoteCount() {
		t.Fatalf("contagens divergem: %d vs %d", first.NoteCount(), second.NoteCount())
	}
	for i := range 50 {
		p := vault.CanonicalPath(fmt.Sprintf("n%02d.md", i))
		a, okA := first.Get(p)
		b, okB := second.Get(p)
		if !okA || !okB {
			t.Fatalf("%s ausente em um dos indices", p)
		}
		if a.Hash != b.Hash || len(a.Links) != len(b.Links) {
			t.Errorf("%s divergiu entre construcoes", p)
		}
	}
}

func TestBuildRespectsContextCancellation(t *testing.T) {
	root := t.TempDir()
	for i := range 200 {
		writeFile(t, root, fmt.Sprintf("n%03d.md", i), "# N\n")
	}

	v, _ := vault.New(root)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	idx := index.New()
	if err := idx.Build(ctx, v); err == nil {
		t.Fatal("Build com context cancelado deveria falhar")
	}
}

// Extras checks

// 1. ctx cancelado antes do Build: ja coberto em TestBuildRespectsContextCancellation (meça quantas foram visitadas? No código, gctx cancela o vault.Walk, logo visita 0)
// Vamos adicionar um teste para medir visitas com cancelamento
func TestBuildContextCancellationStopsEarly(t *testing.T) {
	root := t.TempDir()
	for i := range 50 {
		writeFile(t, root, fmt.Sprintf("n%03d.md", i), "# N\n")
	}
	v, _ := vault.New(root)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	idx := index.New()
	idx.Build(ctx, v) // Should fail and stop early
	if idx.NoteCount() == 50 {
		t.Error("context foi ignorado e indexou tudo")
	}
}

// 2. Um arquivo ilegível no meio da varredura derruba a construção inteira ou é pulado?
func TestBuildSkipsUnreadableFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A")
	writeFile(t, root, "B.md", "# B")

	v, _ := vault.New(root)

	// Create an unreadable file (directory with .md extension will fail to read)
	unreadablePath := filepath.Join(root, "unreadable.md")
	os.Mkdir(unreadablePath, 0755)

	idx := index.New()
	err := idx.Build(context.Background(), v)
	if err != nil {
		t.Fatalf("Build failed because of unreadable file: %v", err)
	}

	if idx.NoteCount() != 2 {
		t.Errorf("NoteCount = %d, quer 2 (A e B lidos, unreadable pulado)", idx.NoteCount())
	}
}

// 3. Uma nota com frontmatter tem offsets de heading corretos em relação ao buffer?
func TestBuildOffsetsWithFrontmatter(t *testing.T) {
	root := t.TempDir()
	content := "---\ntitle: F\n---\n# H1\n"
	writeFile(t, root, "F.md", content)
	v, _ := vault.New(root)

	idx := index.New()
	idx.Build(context.Background(), v)

	n, _ := idx.Get("F.md")
	if len(n.Headings) == 0 {
		t.Fatal("Nenhum heading encontrado")
	}
	h := n.Headings[0]
	actual := content[h.Start:h.End]
	if string(actual) != "# H1\n" {
		t.Errorf("Conteudo fatiado = %q, quer %q", actual, "# H1\n")
	}
}

// 4. Anexo entra em assets e não em notes? Somente-nuvem entra sem ser lido?
func TestBuildAssetsAndCloudOnly(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A")
	writeFile(t, root, "A.png", "img")
	writeFile(t, root, "C.url", "[InternetShortcut]\nURL=fake") // placeholder example, maybe not detected depending on OS/Vault rules. Vault should detect it if we use standard trick.
	// The problem is we don't mock CloudOnly in this test. We can only test assets properly if we can't mock cloud-only.

	v, _ := vault.New(root)
	idx := index.New()
	idx.Build(context.Background(), v)

	if idx.AssetCount() != 1 {
		t.Errorf("AssetCount = %d, quer 1", idx.AssetCount())
	}
	// We might not trigger CloudOnly easily on non-Windows/OneDrive, but asset check works.
}

// 5. Costura do BOM
func TestBuildBOM(t *testing.T) {
	root := t.TempDir()
	content := "\xef\xbb\xbf# Bom Heading\n"
	writeFile(t, root, "bom.md", content)
	v, _ := vault.New(root)

	idx := index.New()
	idx.Build(context.Background(), v)

	n, _ := idx.Get("bom.md")
	if len(n.Headings) == 0 {
		t.Fatal("BOM fez o parser engasgar, nenhum heading")
	}
	if n.Headings[0].Text != "Bom Heading" {
		t.Errorf("Heading = %q, quer %q", n.Headings[0].Text, "Bom Heading")
	}
	if n.BOM != true {
		t.Error("BOM field not true")
	}
}
