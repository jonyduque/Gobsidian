package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

// O slug existe para casar o texto do heading com a ancora de um wikilink.
// [[nota#Capitulo 118]] tem que achar "## Capítulo 118".
func TestSlug(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Capítulo 118", "capitulo 118"},
		{"CAPÍTULO 118", "capitulo 118"},
		{"  Capitulo   118  ", "capitulo 118"},
		{"Ação & Reação", "acao reacao"},
		{"Art. 1.234 — CPC", "art 1234 cpc"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := parser.Slug(tt.in); got != tt.want {
			t.Errorf("Slug(%q) = %q, quer %q", tt.in, got, tt.want)
		}
	}
}
