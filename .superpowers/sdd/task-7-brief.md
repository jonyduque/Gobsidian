### Task 7: Caminho canônico e confinamento

**Files:**
- Create: `internal/vault/path.go`, `internal/vault/path_test.go`

**Interfaces:**
- Consumes: nada
- Produces: `vault.CanonicalPath` (alias de `string`); `vault.Canonicalize(root, abs string) (CanonicalPath, error)`; `vault.Resolve(root, input string) (abs string, canon CanonicalPath, err error)`; `vault.ErrOutsideVault`

- [ ] **Step 1: Escrever o teste de confinamento**

`internal/vault/path_test.go`:

```go
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
```

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/vault/ -v`
Esperado: FAIL — `undefined: vault.Resolve`.

- [ ] **Step 3: Implementar**

`internal/vault/path.go`:

```go
package vault

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// CanonicalPath e a unica forma interna de identificar uma nota: relativa a
// raiz do cofre, com separador "/", sem "./" inicial, e com a grafia exata do
// disco — maiusculas e minusculas inclusive.
//
// A grafia exata importa. Cofres reais acumulam inconsistencia de casing entre
// pastas, e o indice precisa refletir o que esta no disco, nao uma
// normalizacao inventada.
type CanonicalPath string

var (
	ErrOutsideVault = errors.New("caminho fora do cofre")
	ErrAbsolutePath = errors.New("caminho absoluto nao aceito")
	ErrEmptyPath    = errors.New("caminho vazio")
)

// Resolve traduz a entrada de uma tool em caminho absoluto e canonico,
// rejeitando qualquer coisa que escape do cofre.
func Resolve(root, input string) (string, CanonicalPath, error) {
	if strings.TrimSpace(input) == "" {
		return "", "", ErrEmptyPath
	}

	slashed := filepath.ToSlash(input)
	if path.IsAbs(slashed) || filepath.IsAbs(input) || hasDriveLetter(slashed) {
		return "", "", fmt.Errorf("%w: %q", ErrAbsolutePath, input)
	}

	cleaned := path.Clean(slashed)
	if cleaned == "." {
		return "", "", ErrEmptyPath
	}

	abs := filepath.Join(root, filepath.FromSlash(cleaned))
	canon, err := Canonicalize(root, abs)
	if err != nil {
		return "", "", err
	}
	return abs, canon, nil
}

// Canonicalize converte um caminho absoluto em CanonicalPath, verificando o
// confinamento por componente de caminho.
func Canonicalize(root, abs string) (CanonicalPath, error) {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrOutsideVault, abs)
	}

	slashed := filepath.ToSlash(rel)
	if slashed == ".." || strings.HasPrefix(slashed, "../") {
		return "", fmt.Errorf("%w: %q", ErrOutsideVault, abs)
	}
	if slashed == "." {
		return "", ErrEmptyPath
	}

	return CanonicalPath(slashed), nil
}

// hasDriveLetter detecta "C:/..." mesmo em plataformas onde filepath.IsAbs
// nao reconhece a forma do Windows. Uma tool nunca deve aceitar caminho
// absoluto, e a rejeicao precisa valer em qualquer sistema.
func hasDriveLetter(s string) bool {
	if len(s) < 2 || s[1] != ':' {
		return false
	}
	c := s[0]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
```

**Por que `filepath.Rel` e não comparação de prefixo.** `Rel` opera por componente, que é a única comparação correta. `/cofre-outro` tem `/cofre` como prefixo textual e não é interno a ele; um `strings.HasPrefix` aceitaria e abriria o confinamento inteiro.

- [ ] **Step 4: Rodar para confirmar que passa**

Run: `go test -race ./internal/vault/ -v`
Esperado: PASS, onze subcasos.

- [ ] **Step 5: Commit**

```bash
git add internal/vault
git commit -m "feat(vault): canonical paths with component-wise confinement check"
```

---

