//go:build windows

package vaulttest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/vaulttest"
)

// A prova dentro do helper e o que o pacote vende. Este teste confere que ela
// existe: depois de TravarExclusivo, a leitura falha; depois do Cleanup, volta
// a funcionar (o handle foi fechado e nao vazou para o proximo teste).
func TestTravarExclusivoBarraLeituraEDevolveNoCleanup(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Run("travado", func(t *testing.T) {
		vaulttest.TravarExclusivo(t, abs)
		if _, err := os.ReadFile(abs); err == nil {
			t.Fatal("leitura passou com handle exclusivo aberto")
		}
	})
	if _, err := os.ReadFile(abs); err != nil {
		t.Fatalf("depois do Cleanup a leitura devia voltar: %v", err)
	}
}

func TestTravarDiretorioExclusivoBarraListagem(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Run("travado", func(t *testing.T) {
		vaulttest.TravarDiretorioExclusivo(t, dir)
		if _, err := os.ReadDir(dir); err == nil {
			t.Fatal("ReadDir passou com handle exclusivo aberto")
		}
	})
	if _, err := os.ReadDir(dir); err != nil {
		t.Fatalf("depois do Cleanup a listagem devia voltar: %v", err)
	}
}
