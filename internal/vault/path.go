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
