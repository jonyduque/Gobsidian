package service

import (
	"context"
	"encoding/json"
	"testing"
)

// TestNoteMetadataLinksSeguemOContrato: o JSON de links e backlinks de
// note_metadata e o que docs/TOOLS.md promete -- chaves minusculas, `state` e
// `via` como palavra, `kind` como palavra, como vault_broken_links ja fazia.
//
// Medido em 2026-10-01 pela CLI nova (que imprime o structuredContent da tool):
// cada link saia com "Context", "Resolved", "State":3, "Via":2 e "kind":1,
// porque MetadataResult expunha index.ResolvedLink e index.Backlink crus, sem
// tag JSON e com os estados como inteiro. O contrato diz `state` em
// ok|target_missing|anchor_missing e `via` em path|name|asset|alias; o modelo
// recebia numeros que nenhum documento traduz.
func TestNoteMetadataLinksSeguemOContrato(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# A\n\n## Sec\n\nVer [[B]] e [x](https://exemplo.org).\n")
	writeFile(t, root, "B.md", "# B\n\nVolta para [[A#Sec]].\n")
	svc := newTestService(t, root)

	res, err := svc.NoteMetadata(context.Background(), MetadataRequest{Path: "A.md", Include: []string{"links", "backlinks"}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var cru struct {
		Links     []map[string]any `json:"links"`
		Backlinks []map[string]any `json:"backlinks"`
	}
	if err := json.Unmarshal(b, &cru); err != nil {
		t.Fatal(err)
	}

	porAlvo := map[string]map[string]any{}
	for _, l := range cru.Links {
		porAlvo[l["target"].(string)] = l
	}
	paraB := porAlvo["B"]
	if paraB == nil {
		t.Fatalf("link para B ausente: %s", b)
	}
	quer := map[string]any{"state": "ok", "via": "name", "kind": "wikilink", "resolved": "B.md"}
	for k, v := range quer {
		if paraB[k] != v {
			t.Errorf("links[B].%s = %#v, quer %#v (%s)", k, paraB[k], v, b)
		}
	}
	if _, ok := paraB["context"]; !ok {
		t.Errorf("links[B] sem context: %s", b)
	}
	for k := range paraB {
		if k != "" && k[0] >= 'A' && k[0] <= 'Z' {
			t.Errorf("chave com maiuscula em links: %q (%s)", k, b)
		}
	}
	if ext := porAlvo["https://exemplo.org"]; ext == nil || ext["state"] != "external" || ext["kind"] != "markdown" {
		t.Errorf("link externo = %#v, quer state external e kind markdown", ext)
	}

	if len(cru.Backlinks) != 1 {
		t.Fatalf("backlinks = %d, quer 1 (%s)", len(cru.Backlinks), b)
	}
	bl := cru.Backlinks[0]
	quer = map[string]any{"from": "B.md", "anchor": "Sec", "kind": "wikilink", "heading": "B"}
	for k, v := range quer {
		if bl[k] != v {
			t.Errorf("backlinks[0].%s = %#v, quer %#v (%s)", k, bl[k], v, b)
		}
	}
	if _, ok := bl["context"]; !ok {
		t.Errorf("backlink sem context: %s", b)
	}
}
