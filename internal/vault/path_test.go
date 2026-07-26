package vault_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

func TestResolveConfinement(t *testing.T) {
	root := filepath.FromSlash("C:/cofre")

	tests := []struct {
		name    string
		input   string
		wantErr bool
		want    vault.CanonicalPath
	}{
		{name: "simples", input: "Civil/PONTO 03.md", want: "Civil/PONTO 03.md"},
		{name: "barra invertida", input: `Civil\PONTO 03.md`, want: "Civil/PONTO 03.md"},
		{name: "ponto inicial", input: "./Civil/PONTO 03.md", want: "Civil/PONTO 03.md"},
		{name: "duplo ponto interno", input: "Civil/../Penal/A.md", want: "Penal/A.md"},
		{name: "escapa do cofre", input: "../outro/A.md", wantErr: true},
		{name: "escapa com muitos niveis", input: "a/b/../../../fora.md", wantErr: true},
		{name: "absoluto rejeitado", input: "C:/cofre/A.md", wantErr: true},
		{name: "absoluto unix rejeitado", input: "/etc/passwd", wantErr: true},
		{name: "vazio rejeitado", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, canon, err := vault.Resolve(root, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Resolve(%q) = %q, quer erro", tt.input, canon)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve(%q) error = %v", tt.input, err)
			}
			if canon != tt.want {
				t.Errorf("Resolve(%q) = %q, quer %q", tt.input, canon, tt.want)
			}
		})
	}
}

// O bug classico: comparacao por prefixo de string aceita um irmao cujo nome
// comeca com o nome da raiz. A comparacao tem que ser por componente.
func TestResolveRejectsSiblingWithSharedPrefix(t *testing.T) {
	root := filepath.FromSlash("C:/cofre")
	_, _, err := vault.Resolve(root, "../cofre-outro/A.md")
	if !errors.Is(err, vault.ErrOutsideVault) {
		t.Fatalf("erro = %v, quer ErrOutsideVault", err)
	}
}

func TestCanonicalizeUsesForwardSlashes(t *testing.T) {
	root := filepath.FromSlash("C:/cofre")
	abs := filepath.Join(root, "Civil", "PONTO 03.md")

	got, err := vault.Canonicalize(root, abs)
	if err != nil {
		t.Fatalf("Canonicalize error = %v", err)
	}
	if got != "Civil/PONTO 03.md" {
		t.Errorf("Canonicalize = %q, quer %q", got, "Civil/PONTO 03.md")
	}
}
