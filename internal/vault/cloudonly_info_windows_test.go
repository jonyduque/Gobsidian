//go:build windows

package vault

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// TestIsCloudOnlyInfoConcordaComIsCloudOnly guarda o achado P14 pela parte que
// pode dar errado em SILÊNCIO.
//
// A varredura trocou `IsCloudOnly(abs)` — um syscall por entrada — por
// `IsCloudOnlyInfo(info)`, que lê os atributos do `fs.FileInfo` que o walk já
// tinha. Se `info.Sys()` não trouxer `*syscall.Win32FileAttributeData`, a
// versão nova devolve false por não saber, e o efeito é a detecção de
// somente-nuvem sumir sem nada falhar: o índice passaria a ABRIR placeholders,
// disparando download síncrono do cofre inteiro no boot — que é exatamente o
// que a regra existe para impedir.
//
// FILE_ATTRIBUTE_OFFLINE é gravável por SetFileAttributes e entra na mesma
// máscara, então serve para montar a condição sem depender do OneDrive.
func TestIsCloudOnlyInfoConcordaComIsCloudOnly(t *testing.T) {
	dir := t.TempDir()
	normal := filepath.Join(dir, "normal.md")
	nuvem := filepath.Join(dir, "nuvem.md")
	for _, p := range []string{normal, nuvem} {
		if err := os.WriteFile(p, []byte("# x\n"), 0644); err != nil {
			t.Fatalf("escrevendo %s: %v", p, err)
		}
	}

	pw, err := windows.UTF16PtrFromString(LongPath(nuvem))
	if err != nil {
		t.Fatalf("UTF16PtrFromString: %v", err)
	}
	if err := windows.SetFileAttributes(pw, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Skipf("nao foi possivel marcar FILE_ATTRIBUTE_OFFLINE: %v", err)
	}
	t.Cleanup(func() { _ = windows.SetFileAttributes(pw, windows.FILE_ATTRIBUTE_NORMAL) })

	// Sem isto o teste compararia false com false nos dois arquivos e passaria
	// sem exercitar nada.
	if !IsCloudOnly(nuvem) {
		t.Fatal("IsCloudOnly nao viu o FILE_ATTRIBUTE_OFFLINE: o cenario nao se montou")
	}

	for _, c := range []struct {
		nome    string
		caminho string
	}{{"normal", normal}, {"somente-nuvem", nuvem}} {
		info, err := os.Lstat(c.caminho)
		if err != nil {
			t.Fatalf("Lstat %s: %v", c.nome, err)
		}
		if got, quer := IsCloudOnlyInfo(info), IsCloudOnly(c.caminho); got != quer {
			t.Errorf("%s: IsCloudOnlyInfo = %v, IsCloudOnly = %v\n"+
				"as duas contas divergiram: a varredura passaria a ABRIR placeholder",
				c.nome, got, quer)
		}
	}
}

// TestWalkMarcaSomenteNuvem fecha o caminho de verdade: não basta a função
// concordar isolada, o `fs.FileInfo` que o WalkDir entrega tem de carregar os
// atributos. É esse `info`, e não um do `Lstat` do teste, que a varredura usa.
func TestWalkMarcaSomenteNuvem(t *testing.T) {
	dir := t.TempDir()
	nuvem := filepath.Join(dir, "nuvem.md")
	if err := os.WriteFile(nuvem, []byte("# x\n"), 0644); err != nil {
		t.Fatalf("escrevendo: %v", err)
	}
	pw, err := windows.UTF16PtrFromString(LongPath(nuvem))
	if err != nil {
		t.Fatalf("UTF16PtrFromString: %v", err)
	}
	if err := windows.SetFileAttributes(pw, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Skipf("nao foi possivel marcar FILE_ATTRIBUTE_OFFLINE: %v", err)
	}
	t.Cleanup(func() { _ = windows.SetFileAttributes(pw, windows.FILE_ATTRIBUTE_NORMAL) })

	v, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var visto bool
	err = v.Walk(context.Background(), func(e Entry) error {
		if filepath.Base(string(e.Path)) == "nuvem.md" {
			visto = true
			if !e.CloudOnly {
				t.Error("Walk devolveu CloudOnly=false para placeholder: o fs.FileInfo do WalkDir " +
					"nao carrega os atributos, e IsCloudOnlyInfo cai em false por nao saber")
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if !visto {
		t.Fatal("a nota nao apareceu na varredura: o cenario nao exercita nada")
	}
}
