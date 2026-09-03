//go:build windows

package vault

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// travarDiretorioExclusivo abre dir com dwShareMode = 0. Enquanto o handle
// vive, ReadDir(dir) falha com ERROR_SHARING_VIOLATION e Lstat(dir) passa —
// a segunda forma de falha da raiz que FalhaNaRaiz existe para reconhecer.
// Provado nesta maquina em 2026-09-02; o teste confere de novo e pula, com o
// motivo, se o SO desta vez deixar ler.
func travarDiretorioExclusivo(t *testing.T, dir string) {
	t.Helper()
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		t.Fatalf("CreateFile exclusivo em %q: %v", dir, err)
	}
	t.Cleanup(func() { _ = windows.CloseHandle(h) })
	if _, err := os.ReadDir(dir); err == nil {
		t.Skip("handle exclusivo NAO impediu ReadDir nesta maquina; o cenario nao se reproduz")
	}
}

func TestWalkNaoEngoleRaizQueExisteMasNaoLe(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	travarDiretorioExclusivo(t, root)

	var vistos int
	err = v.Walk(context.Background(), func(Entry) error { vistos++; return nil })
	if err == nil {
		t.Fatalf("Walk devolveu nil com %d entradas para uma raiz que ReadDir nao le: cofre inacessivel virou cofre vazio", vistos)
	}
}
