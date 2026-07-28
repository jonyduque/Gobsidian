package index_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

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

// O irmao acima prova que o Build FALHA com context cancelado. Este prova que
// ele PARA: falhar depois de percorrer o cofre inteiro satisfaria aquele teste
// e nao satisfaria o requisito, e a diferenca entre os dois so aparece contando
// o que entrou no indice.
//
// O cancelamento e explicito, nao um timeout curto: um prazo de 1 ms compete
// com a maquina, e num host rapido as 50 notas entram antes de ele expirar — o
// teste passaria a reprovar por velocidade, nao por defeito.
func TestBuildContextCancellationStopsEarly(t *testing.T) {
	root := t.TempDir()
	for i := range 50 {
		writeFile(t, root, fmt.Sprintf("n%03d.md", i), "# N\n")
	}
	v, _ := vault.New(root)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	idx := index.New()
	if err := idx.Build(ctx, v); err == nil {
		t.Fatal("Build com context cancelado deveria falhar")
	}
	if n := idx.NoteCount(); n != 0 {
		t.Errorf("NoteCount = %d, quer 0 - a varredura seguiu depois do cancelamento", n)
	}
}

// Um arquivo ilegivel no meio da varredura derruba a construcao inteira ou e pulado?
func TestBuildSkipsUnreadableFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A")
	writeFile(t, root, "B.md", "# B")

	v, _ := vault.New(root)

	// Create an unreadable file (directory with .md extension will fail to read)
	// Um diretorio com extensao .md: a varredura o classifica como nota e a
	// leitura falha. Se o Mkdir falhar em silencio, o teste passa sem ter
	// exercitado o caso ilegivel — que e justamente o que ele existe para
	// cobrir.
	unreadablePath := filepath.Join(root, "unreadable.md")
	if err := os.Mkdir(unreadablePath, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	idx := index.New()
	err := idx.Build(context.Background(), v)
	if err != nil {
		t.Fatalf("Build failed because of unreadable file: %v", err)
	}

	if idx.NoteCount() != 2 {
		t.Errorf("NoteCount = %d, quer 2 (A e B lidos, unreadable pulado)", idx.NoteCount())
	}
}

// Uma nota com frontmatter tem offsets de heading corretos em relacao ao buffer?
func TestBuildOffsetsWithFrontmatter(t *testing.T) {
	root := t.TempDir()
	content := "---\ntitle: F\n---\n# H1\n"
	writeFile(t, root, "F.md", content)
	v, _ := vault.New(root)

	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	n, ok := idx.Get("F.md")
	if !ok {
		t.Fatal("F.md nao entrou no indice")
	}
	if len(n.Headings) == 0 {
		t.Fatal("Nenhum heading encontrado")
	}
	h := n.Headings[0]
	actual := content[h.Start:h.End]
	if string(actual) != "# H1\n" {
		t.Errorf("Conteudo fatiado = %q, quer %q", actual, "# H1\n")
	}
}

// Anexo entra em assets e nao em notes?
//
// O caso somente-nuvem NAO esta coberto aqui, e dizer isso e melhor que
// insinuar que esta: o atributo depende do OneDrive e do sistema de arquivos, e
// nao da para produzi-lo num TempDir. Fica registrado como lacuna.
func TestBuildAssetsAndCloudOnly(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A")
	writeFile(t, root, "A.png", "img")

	v, _ := vault.New(root)
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if idx.AssetCount() != 1 {
		t.Errorf("AssetCount = %d, quer 1", idx.AssetCount())
	}
	if idx.NoteCount() != 1 {
		t.Errorf("NoteCount = %d, quer 1 - o anexo nao pode entrar como nota", idx.NoteCount())
	}
}

// Costura do BOM
func TestBuildBOM(t *testing.T) {
	root := t.TempDir()
	content := "\xef\xbb\xbf# Bom Heading\n"
	writeFile(t, root, "bom.md", content)
	v, _ := vault.New(root)

	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	n, ok := idx.Get("bom.md")
	if !ok {
		t.Fatal("bom.md nao entrou no indice")
	}
	if len(n.Headings) == 0 {
		t.Fatal("BOM fez o parser engasgar, nenhum heading")
	}
	if n.Headings[0].Text != "Bom Heading" {
		t.Errorf("Heading = %q, quer %q", n.Headings[0].Text, "Bom Heading")
	}
	if n.BOM != true {
		t.Error("BOM field not true")
	}

	// Uma asercao de presenca nao pega deslocamento: o heading pode "existir"
	// com o texto certo mesmo que Start/End estejam tres bytes cedo demais.
	// Os offsets guardados no indice sao do ARQUIVO (com BOM), nao do corpo
	// que o parser recebeu (sem BOM) — por isso fatiamos content, que ainda
	// tem os tres bytes do marcador, e nao body.
	h := n.Headings[0]
	if h.Start < 0 || h.End > int64(len(content)) {
		t.Fatalf("offsets fora do arquivo: Start=%d End=%d len=%d", h.Start, h.End, len(content))
	}
	got := content[h.Start:h.End]
	want := "# Bom Heading\n"
	if got != want {
		t.Errorf("fatia pelos offsets do indice = %q, quer %q (offset de BOM nao foi somado)", got, want)
	}
}
