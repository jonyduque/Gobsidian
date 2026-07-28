package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(root, name), []byte(content), 0644)
	if err != nil {
		t.Fatalf("escrevendo %s: %v", name, err)
	}
}

func newTestService(t *testing.T, root string) *Service {
	t.Helper()
	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New(%q): %v", root, err)
	}

	// How to build an index for tests? We should probably just use index.Build
	ix := index.New()
	err = ix.Build(context.Background(), v)
	if err != nil {
		t.Fatalf("index.Build: %v", err)
	}

	// I need to use the interface Index if we didn't add the methods to Index yet.
	// We'll update the Index interface in service.go.
	return New(v, ix, Options{})
}

func TestReadNoteSection(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "---\ntitle: A\n---\n# Titulo\n\n## Cap 1\n\ntexto um\n\n## Cap 2\n\ntexto dois\n")

	svc := newTestService(t, root)

	res, err := svc.ReadNote(context.Background(), ReadRequest{
		Path:    "A.md",
		Heading: "Cap 1",
	})
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}

	if !strings.Contains(string(res.Content), "texto um") {
		t.Errorf("conteudo nao contem a secao: %q", res.Content)
	}
	if strings.Contains(string(res.Content), "texto dois") {
		t.Errorf("conteudo vazou para a secao seguinte: %q", res.Content)
	}
	if res.Hash == "" {
		t.Error("Hash vazio — expected_hash depende dele")
	}
	if res.Section == nil || res.Section.Level != 2 {
		t.Errorf("Section = %+v, quer nivel 2", res.Section)
	}
}

func TestReadNoteHeadingNotFoundListsAlternatives(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# T\n\n## Cap 115\n\n## Cap 116\n\n## Cap 117\n")

	svc := newTestService(t, root)

	_, err := svc.ReadNote(context.Background(), ReadRequest{Path: "A.md", Heading: "Cap 118"})
	if err == nil {
		t.Fatal("heading inexistente deveria falhar")
	}
	if CodeOf(err) != CodeHeadingNotFound {
		t.Errorf("codigo = %v, quer HEADING_NOT_FOUND", CodeOf(err))
	}
	for _, want := range []string{"Cap 115", "Cap 116", "Cap 117"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("mensagem nao lista %q: %s", want, err.Error())
		}
	}
}

func TestReadNoteAcceptsAccentInsensitiveHeading(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# T\n\n## Capítulo 118\n\ntexto\n")

	svc := newTestService(t, root)

	for _, query := range []string{"Capítulo 118", "capitulo 118", "CAPITULO 118"} {
		res, err := svc.ReadNote(context.Background(), ReadRequest{Path: "A.md", Heading: query})
		if err != nil {
			t.Errorf("ReadNote(%q): %v", query, err)
			continue
		}
		if !strings.Contains(string(res.Content), "texto") {
			t.Errorf("ReadNote(%q) nao trouxe a secao", query)
		}
	}
}

func TestReadNoteRejectsTraversal(t *testing.T) {
	svc := newTestService(t, t.TempDir())

	_, err := svc.ReadNote(context.Background(), ReadRequest{Path: "../fora.md"})
	if CodeOf(err) != CodePathOutsideVault {
		t.Errorf("codigo = %v, quer PATH_OUTSIDE_VAULT", CodeOf(err))
	}
}

func TestReadNoteCloudOnlyFails(t *testing.T) {
	t.Skip("requer arquivo com FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS")
}

// TestReadNoteBOMOffsetParity prova a costura entre index.Build (que soma
// vault.BOMLen aos offsets de uma nota com BOM) e service.ReadNote (que usa
// esses offsets contra vault.ReadRange, que le o arquivo — com o BOM ainda
// nele). Uma asercao so sobre Headings[0].Text passaria com o bug: o heading
// existe com o texto certo mesmo que os offsets estejam tres bytes cedo
// demais. Aqui indexamos a MESMA nota duas vezes, uma com BOM e outra sem, e
// exigimos que o conteudo devolvido pela mesma secao seja identico — isso so
// pode passar se os offsets guardados no indice forem os do arquivo, nao os
// do corpo sem BOM que o parser recebeu.
func TestReadNoteBOMOffsetParity(t *testing.T) {
	body := "# Titulo\n\n## Alvo\n\nCONTEUDO-ESPERADO\n\n## Depois\n\noutro texto\n"

	rootNoBOM := t.TempDir()
	writeFile(t, rootNoBOM, "A.md", body)
	svcNoBOM := newTestService(t, rootNoBOM)

	rootBOM := t.TempDir()
	writeFile(t, rootBOM, "A.md", "\xef\xbb\xbf"+body)
	svcBOM := newTestService(t, rootBOM)

	resNoBOM, err := svcNoBOM.ReadNote(context.Background(), ReadRequest{Path: "A.md", Heading: "Alvo"})
	if err != nil {
		t.Fatalf("ReadNote (sem BOM): %v", err)
	}
	resBOM, err := svcBOM.ReadNote(context.Background(), ReadRequest{Path: "A.md", Heading: "Alvo"})
	if err != nil {
		t.Fatalf("ReadNote (com BOM): %v", err)
	}
	t.Logf("conteudo sem BOM: %q", resNoBOM.Content)
	t.Logf("conteudo com BOM: %q", resBOM.Content)

	if resNoBOM.Content != resBOM.Content {
		t.Errorf("conteudo divergiu entre nota sem BOM e com BOM:\n  sem BOM: %q\n  com BOM: %q", resNoBOM.Content, resBOM.Content)
	}
	wantContent := "## Alvo\n\nCONTEUDO-ESPERADO\n\n"
	if resNoBOM.Content != wantContent {
		t.Errorf("conteudo sem BOM = %q, quer %q", resNoBOM.Content, wantContent)
	}
}

// heading_level nao era exercitado por nenhum teste. O parametro esta
// documentado em docs/TOOLS.md para note_read e note_patch, e existe para um
// caso concreto: o mesmo texto de heading aparecendo em niveis diferentes.
//
// Sem cobertura nao havia nada segurando nem a desambiguacao nem o caminho de
// erro que ela evita — neutralizar a comparacao de nivel deixava a suite verde.
func TestReadNoteHeadingLevelDisambiguates(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# T\n\n## Resumo\n\nnivel dois\n\n### Resumo\n\nnivel tres\n")
	svc := newTestService(t, root)

	casos := []struct {
		nivel int
		quer  string
	}{
		{2, "nivel dois"},
		{3, "nivel tres"},
	}

	for _, c := range casos {
		res, err := svc.ReadNote(context.Background(), ReadRequest{
			Path:         "A.md",
			Heading:      "Resumo",
			HeadingLevel: c.nivel,
		})
		if err != nil {
			t.Errorf("heading_level=%d: %v", c.nivel, err)
			continue
		}
		if !strings.Contains(res.Content, c.quer) {
			t.Errorf("heading_level=%d devolveu %q, queria conter %q", c.nivel, res.Content, c.quer)
		}
	}
}

// heading_level so separa headings de niveis DIFERENTES. Dois com o mesmo texto
// no MESMO nivel continuam ambiguos, e o erro precisa dizer isso em vez de
// devolver o primeiro em silencio — o cliente e um modelo decidindo o proximo
// passo a partir da mensagem.
func TestReadNoteAmbiguousSameLevel(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# T\n\n## Dup\n\num\n\n## Dup\n\ndois\n")
	svc := newTestService(t, root)

	_, err := svc.ReadNote(context.Background(), ReadRequest{Path: "A.md", Heading: "Dup"})
	if err == nil {
		t.Fatal("dois headings de mesmo texto e mesmo nivel deveriam ser ambiguos")
	}
	if CodeOf(err) != CodeAmbiguousHeading {
		t.Errorf("codigo = %v, quer AMBIGUOUS_HEADING", CodeOf(err))
	}
	if !strings.Contains(err.Error(), "2") {
		t.Errorf("mensagem nao diz quantas ocorrencias: %s", err.Error())
	}
}
