//go:build windows

package vault_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/vault"
	"github.com/jonyduque/Gobsidian/internal/vaulttest"
)

// TestWalkNaoEngoleRaizQueExisteMasNaoLe cobre a segunda forma de falha da
// raiz, a que FalhaNaRaiz existe para reconhecer: com um handle exclusivo sobre
// o diretorio, Lstat(raiz) passa e ReadDir(raiz) falha com
// ERROR_SHARING_VIOLATION. Um cofre inacessivel nao pode responder como cofre
// vazio.
//
// A trava vem de internal/vaulttest, que prova que ReadDir de fato falha antes
// de devolver — sem essa prova o teste percorreria um diretorio legivel e
// passaria sem exercitar nada. O arquivo virou package vault_test por isso: o
// helper importa vault, e so o pacote de teste externo pode importa-lo de volta.
func TestWalkNaoEngoleRaizQueExisteMasNaoLe(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	vaulttest.TravarDiretorioExclusivo(t, root)

	var vistos int
	err = v.Walk(context.Background(), func(vault.Entry) error { vistos++; return nil })
	if err == nil {
		t.Fatalf("Walk devolveu nil com %d entradas para uma raiz que ReadDir nao le: cofre inacessivel virou cofre vazio", vistos)
	}
}
