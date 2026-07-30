package parser_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/parser"
)

func TestLinkMarkdownOffsets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKind parser.LinkKind
		wantText string
	}{
		{
			name:     "link markdown simples",
			input:    "Linha inicial\n[texto](destino.md)\nLinha final",
			wantKind: parser.LinkMarkdown,
			wantText: "[texto](destino.md)",
		},
		{
			name:     "embed markdown simples",
			input:    "Linha inicial\n![alt](img.png)\nLinha final",
			wantKind: parser.LinkEmbed,
			wantText: "![alt](img.png)",
		},
		{
			name:     "aninhado ambiguo",
			input:    "[[[a]] b](d.md)",
			wantKind: parser.LinkMarkdown,
			wantText: "[[[a]] b](d.md)",
		},
		{
			name:     "parenteses no destino",
			input:    "[x](a(b).md)",
			wantKind: parser.LinkMarkdown,
			wantText: "[x](a(b).md)",
		},
		{
			name:     "com titulo",
			input:    `[x](a.md "t")`,
			wantKind: parser.LinkMarkdown,
			wantText: `[x](a.md "t")`,
		},
		{
			name:     "com BOM",
			input:    "\xef\xbb\xbf# Topo\n[texto](destino.md)\n",
			wantKind: parser.LinkMarkdown,
			wantText: "[texto](destino.md)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := []byte(tt.input)
			note, err := parser.Parse(raw)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}

			var found *parser.Link
			for i := range note.Links {
				if note.Links[i].Kind == tt.wantKind {
					found = &note.Links[i]
					break
				}
			}

			if found == nil {
				t.Fatalf("nenhum link do tipo %v encontrado em:\n%s", tt.wantKind, tt.input)
			}

			if found.Start == -1 || found.End == -1 {
				t.Fatalf("Start ou End e -1: Start=%d, End=%d", found.Start, found.End)
			}

			if int(found.End) > len(raw) || found.Start < 0 || found.Start > found.End {
				t.Fatalf("range invalido [%d:%d] para buffer de tamanho %d", found.Start, found.End, len(raw))
			}

			gotText := string(raw[found.Start:found.End])
			if gotText != tt.wantText {
				t.Errorf("raw[Start:End] = %q, quer %q", gotText, tt.wantText)
			}
		})
	}
}
