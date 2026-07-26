//go:build windows

package vault_test

import (
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// TestLongPathWindows exercises LongPath's four load-bearing behaviors on
// Windows: the 240-character threshold, the \\?\ prefix, idempotence against
// an already-prefixed path, and UNC handling. LongPath had no test anywhere
// before this — one of Task 8's five named deliverables was entirely
// unverified.
func TestLongPathWindows(t *testing.T) {
	// pad(n) builds a path component of exactly n bytes so test cases can
	// hit precise lengths without hardcoding the package's private
	// threshold constant.
	pad := func(n int) string {
		return strings.Repeat("a", n)
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "caminho curto e devolvido sem alteracao",
			in:   `C:\Users\jonyd\Documents\note.md`,
			want: `C:\Users\jonyd\Documents\note.md`,
		},
		{
			name: "caminho logo abaixo do limiar (239) nao ganha prefixo",
			// "C:\" (3) + 236 'a' = 239 bytes.
			in:   `C:\` + pad(236),
			want: `C:\` + pad(236),
		},
		{
			name: "caminho no limiar (240) ganha o prefixo \\\\?\\\\",
			// "C:\" (3) + 237 'a' = 240 bytes.
			in:   `C:\` + pad(237),
			want: `\\?\C:\` + pad(237),
		},
		{
			name: "caminho acima do limiar ganha o prefixo",
			in:   `C:\` + pad(300),
			want: `\\?\C:\` + pad(300),
		},
		{
			name: "caminho ja prefixado nao e prefixado de novo",
			in:   `\\?\C:\` + pad(300),
			want: `\\?\C:\` + pad(300),
		},
		{
			name: "UNC vira \\\\?\\\\UNC\\\\ sem barra tripla",
			in:   `\\server\share\` + pad(230),
			want: `\\?\UNC\server\share\` + pad(230),
		},
		{
			name: "barras normais e segmentos . e .. somem do resultado",
			in:   `C:/Users/jonyd/../jonyd/AppData/Local/` + pad(230),
			want: `\\?\C:\Users\jonyd\AppData\Local\` + pad(230),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := vault.LongPath(tt.in)
			if got != tt.want {
				t.Errorf("LongPath(%d bytes) = %q, quer %q", len(tt.in), got, tt.want)
			}
			if strings.Contains(got, "/") {
				t.Errorf("LongPath resultado contem barra normal: %q", got)
			}
			if strings.Contains(got, `\..\`) || strings.HasSuffix(got, `\..`) {
				t.Errorf("LongPath resultado contem segmento '..': %q", got)
			}
			if strings.Contains(got, `\?\\\`) {
				t.Errorf("LongPath resultado tem barra tripla apos o prefixo UNC: %q", got)
			}
		})
	}
}
