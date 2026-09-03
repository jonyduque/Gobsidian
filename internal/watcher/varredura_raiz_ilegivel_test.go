//go:build !windows

package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestVarreDiretorioNovoNaoEngoleRaizIlegivel prova que o watcher USA a conta
// de vault.FalhaNaRaiz, e nao uma decisao propria.
//
// O irmao TestVarreDiretorioNovoAcusaFalhaNaPropriaRaiz, em
// varredura_raiz_test.go, cobre so a PRIMEIRA forma de falha da raiz — o
// caminho que nao existe, onde Lstat falha e o callback vem com d == nil. A
// segunda forma nao aparece ali: a raiz EXISTE, Lstat passa, e e o ReadDir que
// falha. Ai o callback vem com d != nil, e a decisao feita a mao tratava isso
// como entrada ilegivel qualquer — engolia, e a varredura devolvia sucesso com
// zero entradas.
//
// Fica fora do Windows porque o instrumento e chmod 0o000, que no Windows nao
// remove permissao de leitura de diretorio. A mesma segunda forma esta provada
// la por outro instrumento — handle exclusivo — em
// internal/vault/walk_raiz_windows_test.go.
func TestVarreDiretorioNovoNaoEngoleRaizIlegivel(t *testing.T) {
	w, cancel, root, _ := setupTestWatcher(t)
	defer cancel()
	dir := filepath.Join(root, "chegou")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if _, err := os.ReadDir(dir); err == nil {
		t.Skip("ReadDir nao falhou com 0o000 (root?); o cenario nao se reproduz aqui")
	}
	err := w.varreDiretorioNovo(context.Background(), dir)
	if err == nil {
		t.Fatal("varreDiretorioNovo devolveu nil para uma raiz que ReadDir nao consegue ler: a varredura reportou sucesso com zero entradas")
	}
}
