//go:build !windows

package vault_test

import (
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

// TestLongPathOther confirms LongPath is the identity function off Windows:
// there is no MAX_PATH to work around, so nothing about the input should
// change, no matter how long it is.
func TestLongPathOther(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"caminho curto", "/home/jonyd/vault/note.md"},
		{"caminho muito longo", "/home/jonyd/vault/" + strings.Repeat("a/", 200) + "note.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vault.LongPath(tt.in); got != tt.in {
				t.Errorf("LongPath(%q) = %q, quer o mesmo valor (identidade fora do Windows)", tt.in, got)
			}
		})
	}
}
