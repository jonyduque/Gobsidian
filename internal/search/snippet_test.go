package search_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

func snippetPara(t *testing.T, v *vault.Vault, idx *index.Index, ix *search.Inverted, path string, term string, maxChars int) search.Snippet {
	t.Helper()
	s, err := search.GenerateSnippet(context.Background(), v, ix, idx, path, search.NovosTermosDeTrecho([]string{term}), maxChars, nil)
	if err != nil {
		t.Fatalf("GenerateSnippet: %v", err)
	}
	return s
}

func TestSnippetOffsetsAlignWithDiskBytesUnderBOM(t *testing.T) {
	root := t.TempDir()
	// BOM + frontmatter + corpo. Os três deslocamentos no mesmo arquivo.
	raw := []byte("\xEF\xBB\xBF---\ntitle: T\n---\n\n# Secao\n\nA prescricao intercorrente corre.\n")
	caminho := filepath.Join(root, "nota.md")
	if err := os.WriteFile(caminho, raw, 0644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}

	data, _ := v.ReadAll(context.Background(), "nota.md")
	body, _ := vault.StripBOM(data)
	ix := search.NewInverted()
	ix.Add("nota.md", search.Analyze(string(body)))

	trecho := snippetPara(t, v, idx, ix, "nota.md", "intercorrente", 240)

	// A asserção é sobre BYTES DO DISCO
	if !strings.Contains(trecho.Text, "intercorrente") {
		t.Fatalf("trecho = %q; nao contem o termo — offset deslocado pelo BOM ou frontmatter", trecho.Text)
	}
	if !utf8.ValidString(trecho.Text) || strings.Contains(trecho.Text, "\uFFFD") {
		t.Errorf("trecho tem byte invalido: recorte caiu no meio de um caractere")
	}
	if got := trecho.Text[trecho.HighlightStart:trecho.HighlightEnd]; got != "intercorrente" {
		t.Errorf("destaque = %q, quer %q", got, "intercorrente")
	}
}

func TestSnippetOffsetsWithoutBOM(t *testing.T) {
	root := t.TempDir()
	raw := []byte("---\ntitle: T\n---\n\n# Secao\n\nA prescricao intercorrente corre.\n")
	caminho := filepath.Join(root, "nota.md")
	if err := os.WriteFile(caminho, raw, 0644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}

	data, _ := v.ReadAll(context.Background(), "nota.md")
	body, _ := vault.StripBOM(data)
	ix := search.NewInverted()
	ix.Add("nota.md", search.Analyze(string(body)))

	trecho := snippetPara(t, v, idx, ix, "nota.md", "intercorrente", 240)

	if !strings.Contains(trecho.Text, "intercorrente") {
		t.Fatalf("trecho = %q; nao contem o termo", trecho.Text)
	}
	if got := trecho.Text[trecho.HighlightStart:trecho.HighlightEnd]; got != "intercorrente" {
		t.Errorf("destaque sem BOM = %q, quer %q", got, "intercorrente")
	}
}

func TestSnippetMaxCharsRespected(t *testing.T) {
	root := t.TempDir()
	longText := strings.Repeat("palavra ", 200) + "alvo " + strings.Repeat("palavra ", 200)
	caminho := filepath.Join(root, "longa.md")
	_ = os.WriteFile(caminho, []byte(longText), 0644)

	v, _ := vault.New(root)
	idx := index.New()
	_ = idx.Build(context.Background(), v)

	ix := search.NewInverted()
	ix.Add("longa.md", search.Analyze(longText))

	// Test default maxChars (240)
	t1 := snippetPara(t, v, idx, ix, "longa.md", "alvo", 0)
	if len(t1.Text) > 240 {
		t.Errorf("len(t1.Text) = %d, quer <= 240", len(t1.Text))
	}

	// Test custom maxChars (50)
	t2 := snippetPara(t, v, idx, ix, "longa.md", "alvo", 50)
	if len(t2.Text) > 50 {
		t.Errorf("len(t2.Text) = %d, quer <= 50", len(t2.Text))
	}

	// Test max clamp (1500 -> 1000)
	t3 := snippetPara(t, v, idx, ix, "longa.md", "alvo", 1500)
	if len(t3.Text) > 1000 {
		t.Errorf("len(t3.Text) = %d, quer <= 1000", len(t3.Text))
	}
}

func TestSnippetTermAtEdges(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "inicio.md"), []byte("alvo no inicio do arquivo texto texto"), 0644)
	_ = os.WriteFile(filepath.Join(root, "fim.md"), []byte("texto texto texto alvo"), 0644)

	v, _ := vault.New(root)
	idx := index.New()
	_ = idx.Build(context.Background(), v)

	ix := search.NewInverted()
	dataI, _ := v.ReadAll(context.Background(), "inicio.md")
	ix.Add("inicio.md", search.Analyze(string(dataI)))

	dataF, _ := v.ReadAll(context.Background(), "fim.md")
	ix.Add("fim.md", search.Analyze(string(dataF)))

	tInit := snippetPara(t, v, idx, ix, "inicio.md", "alvo", 100)
	if got := tInit.Text[tInit.HighlightStart:tInit.HighlightEnd]; got != "alvo" {
		t.Errorf("inicio: destaque = %q, quer %q", got, "alvo")
	}

	tFim := snippetPara(t, v, idx, ix, "fim.md", "alvo", 100)
	if got := tFim.Text[tFim.HighlightStart:tFim.HighlightEnd]; got != "alvo" {
		t.Errorf("fim: destaque = %q, quer %q", got, "alvo")
	}
}

func TestSnippetAccentedWordUTF8Safety(t *testing.T) {
	root := t.TempDir()
	raw := []byte("Texto com a palavra prescrição acentuada.")
	_ = os.WriteFile(filepath.Join(root, "acento.md"), raw, 0644)

	v, _ := vault.New(root)
	idx := index.New()
	_ = idx.Build(context.Background(), v)

	ix := search.NewInverted()
	ix.Add("acento.md", search.Analyze(string(raw)))

	trecho := snippetPara(t, v, idx, ix, "acento.md", "prescrição", 100)

	if !utf8.ValidString(trecho.Text) || strings.Contains(trecho.Text, "\uFFFD") {
		t.Errorf("Snippet contém caractere UTF-8 inválido")
	}
	if got := trecho.Text[trecho.HighlightStart:trecho.HighlightEnd]; got != "prescrição" {
		t.Errorf("destaque acentuado = %q, quer %q", got, "prescrição")
	}
}
