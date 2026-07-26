package vault

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Vault struct {
	root string
}

func New(root string) (*Vault, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolvendo raiz do cofre %q: %w", root, err)
	}
	info, err := os.Stat(LongPath(abs))
	if err != nil {
		return nil, fmt.Errorf("raiz do cofre inacessivel %q: %w", abs, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("raiz do cofre nao e diretorio: %q", abs)
	}
	return &Vault{root: abs}, nil
}

func (v *Vault) Root() string { return v.root }

// Abs devolve o caminho absoluto de um caminho canonico, ja preparado para
// chamadas do sistema.
func (v *Vault) Abs(p CanonicalPath) string {
	return LongPath(filepath.Join(v.root, filepath.FromSlash(string(p))))
}

func (v *Vault) Open(p CanonicalPath) (*os.File, error) {
	f, err := os.Open(v.Abs(p))
	if err != nil {
		return nil, fmt.Errorf("abrindo %q: %w", p, err)
	}
	return f, nil
}

// ReadRange le apenas a faixa pedida. Uma secao de 2 KB em uma nota de 500 KB
// custa 2 KB — e a razao de o indice guardar offsets em vez de conteudo.
func (v *Vault) ReadRange(ctx context.Context, p CanonicalPath, start, end int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if end < start {
		return nil, fmt.Errorf("faixa invalida em %q: %d..%d", p, start, end)
	}

	f, err := v.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, end-start)
	n, err := f.ReadAt(buf, start)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("lendo %q em %d..%d: %w", p, start, end, err)
	}
	return buf[:n], nil
}

func (v *Vault) ReadAll(ctx context.Context, p CanonicalPath) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(v.Abs(p))
	if err != nil {
		return nil, fmt.Errorf("lendo %q: %w", p, err)
	}
	return data, nil
}
