package vault

import (
	"io/fs"
	"testing"
)

// entradaFalsa satisfaz fs.DirEntry sem tocar o disco. So Name e usado
// pela conta; o resto existe para compilar.
type entradaFalsa struct{ nome string }

func (e entradaFalsa) Name() string               { return e.nome }
func (e entradaFalsa) IsDir() bool                { return true }
func (e entradaFalsa) Type() fs.FileMode          { return fs.ModeDir }
func (e entradaFalsa) Info() (fs.FileInfo, error) { return nil, fs.ErrNotExist }

func TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir(t *testing.T) {
	const raiz = `C:\cofre`
	casos := []struct {
		nome    string
		caminho string
		d       fs.DirEntry
		quer    bool
	}{
		{"Lstat da raiz falhou: d == nil", raiz, nil, true},
		{"ReadDir da raiz falhou: d != nil, caminho == raiz", raiz, entradaFalsa{"cofre"}, true},
		{"entrada comum ilegivel", `C:\cofre\sub`, entradaFalsa{"sub"}, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := FalhaNaRaiz(raiz, c.caminho, c.d); got != c.quer {
				t.Fatalf("FalhaNaRaiz(%q, %q, %v) = %v, quer %v", raiz, c.caminho, c.d, got, c.quer)
			}
		})
	}
}
