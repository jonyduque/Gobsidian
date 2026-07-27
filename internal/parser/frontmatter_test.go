package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		wantFM     string
		wantBody   string
		wantOffset int64
	}{
		{
			name:       "presente",
			in:         "---\ntitle: A\n---\n# Corpo\n",
			wantFM:     "title: A\n",
			wantBody:   "# Corpo\n",
			wantOffset: 17,
		},
		{
			name:     "ausente",
			in:       "# Corpo\n",
			wantFM:   "",
			wantBody: "# Corpo\n",
		},
		{
			name:     "tres tracos no meio nao conta",
			in:       "# Corpo\n---\nnao e frontmatter\n",
			wantFM:   "",
			wantBody: "# Corpo\n---\nnao e frontmatter\n",
		},
		{
			name:     "delimitador nao fechado",
			in:       "---\ntitle: A\n# Corpo\n",
			wantFM:   "",
			wantBody: "---\ntitle: A\n# Corpo\n",
		},
		{
			name:     "frontmatter vazio",
			in:       "---\n---\n# Corpo\n",
			wantFM:   "",
			wantBody: "# Corpo\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, off := parser.SplitFrontmatter([]byte(tt.in))
			if string(fm) != tt.wantFM {
				t.Errorf("fm = %q, quer %q", fm, tt.wantFM)
			}
			if string(body) != tt.wantBody {
				t.Errorf("body = %q, quer %q", body, tt.wantBody)
			}
			if tt.wantOffset != 0 && off != tt.wantOffset {
				t.Errorf("offset = %d, quer %d", off, tt.wantOffset)
			}
		})
	}
}

func TestDecodeFrontmatterPreservesTypes(t *testing.T) {
	fm := []byte("titulo: Ponto 3\nnumero: 42\nativo: true\ntags:\n  - civil\n  - obrigacoes\naliases: [P3, Ponto III]\ndata: 2026-07-25\n")

	got, err := parser.DecodeFrontmatter(fm)
	if err != nil {
		t.Fatalf("DecodeFrontmatter: %v", err)
	}

	if got["titulo"] != "Ponto 3" {
		t.Errorf("titulo = %v (%T), quer string", got["titulo"], got["titulo"])
	}
	if n, ok := got["numero"].(int); !ok || n != 42 {
		t.Errorf("numero = %v (%T), quer int 42", got["numero"], got["numero"])
	}
	if b, ok := got["ativo"].(bool); !ok || !b {
		t.Errorf("ativo = %v (%T), quer bool true", got["ativo"], got["ativo"])
	}
	if tags, ok := got["tags"].([]any); !ok || len(tags) != 2 {
		t.Errorf("tags = %v (%T), quer lista de 2", got["tags"], got["tags"])
	}
	if aliases, ok := got["aliases"].([]any); !ok || len(aliases) != 2 {
		t.Errorf("aliases = %v (%T), quer lista de 2", got["aliases"], got["aliases"])
	}
}

// Frontmatter malformado nao pode derrubar o parse: a nota ainda tem corpo,
// headings e links uteis. O erro e reportado e o resto segue.
func TestDecodeFrontmatterMalformedReturnsError(t *testing.T) {
	if _, err := parser.DecodeFrontmatter([]byte("a: [1, 2\nb: :\n")); err == nil {
		t.Fatal("frontmatter malformado deveria devolver erro")
	}
}
