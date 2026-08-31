package mcpsrv_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoteReadAceitaListaMista prova o que o schema promete: string e objeto na
// MESMA lista. Um teste que so manda objetos nao prova a mistura, que e a forma
// que o cliente real vai usar.
//
// E ele exercita o schema, nao so o UnmarshalJSON: o SDK valida a entrada contra
// o InputSchema ANTES de decodificar, entao um schema que declarasse so a forma
// objeto reprovaria o item string aqui, com o UnmarshalJSON nunca chamado.
func TestNoteReadAceitaListaMista(t *testing.T) {
	root := t.TempDir()
	conteudo := "# Alfa\n\ntexto de alfa\n\n# Beta\n\ntexto de beta\n"
	for _, n := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte(conteudo), 0644); err != nil {
			t.Fatal(err)
		}
	}
	session, ctx := novaSessao(t, root)

	var out struct {
		Items []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
			Err     any    `json:"err"`
		} `json:"items"`
	}
	chamaTool(ctx, t, session, "note_read", map[string]any{
		"paths": []any{
			"a.md",
			map[string]any{"path": "b.md", "heading": "Beta"},
		},
	}, &out)

	if len(out.Items) != 2 {
		t.Fatalf("items = %d, quer 2", len(out.Items))
	}
	if !strings.Contains(out.Items[0].Content, "texto de alfa") {
		t.Fatalf("item 0 (string) = %q", out.Items[0].Content)
	}
	if !strings.Contains(out.Items[1].Content, "texto de beta") {
		t.Fatalf("item 1 (objeto) nao recortou a secao: %q", out.Items[1].Content)
	}
	if strings.Contains(out.Items[1].Content, "texto de alfa") {
		t.Fatal("item 1 devolveu a nota inteira; o heading por item foi ignorado")
	}
}

// TestNoteReadListaSoDeStringsContinuaValendo e o contrapeso: a forma que todo
// cliente manda hoje passa pela validacao do SDK sem mudanca nenhuma.
//
// Sem ele, um schema que so aceitasse objeto passaria no teste acima se a
// mistura fosse decodificada por outro caminho, e a quebra apareceria em
// producao no primeiro cliente antigo.
func TestNoteReadListaSoDeStringsContinuaValendo(t *testing.T) {
	root := t.TempDir()
	for _, n := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte("corpo de "+n+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	session, ctx := novaSessao(t, root)

	var out struct {
		Items []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"items"`
	}
	chamaTool(ctx, t, session, "note_read", map[string]any{
		"paths": []any{"a.md", "b.md"},
	}, &out)

	if len(out.Items) != 2 {
		t.Fatalf("items = %d, quer 2", len(out.Items))
	}
	for i, it := range out.Items {
		if !strings.Contains(it.Content, "corpo de ") {
			t.Fatalf("item %d (%s) veio vazio: %q", i, it.Path, it.Content)
		}
	}
}
