package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestBuildRegistraArquivoIlegivel prova que a nota que some do indice deixa
// rastro. Hoje build.go:73 faz `continue` seco: a nota desaparece de todas as
// tools e NADA registra.
func TestBuildRegistraArquivoIlegivel(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "boa.md"), []byte("# Boa\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ilegivelPath := filepath.Join(root, "ilegivel.md")
	if err := os.WriteFile(ilegivelPath, []byte("# Ilegivel\n"), 0644); err != nil {
		t.Fatal(err)
	}

	unlock := lockFileForTest(t, ilegivelPath)
	defer unlock()

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if idx.NoteCount() != 1 {
		t.Fatalf("NoteCount = %d, quer 1 (so a boa)", idx.NoteCount())
	}
	count, pulados := v.SkippedEntries()
	if count == 0 || len(pulados) == 0 {
		t.Fatal("o arquivo ilegivel sumiu do indice sem deixar registro em " +
			"SkippedEntries — cofre com nota perdida responde igual a cofre limpo")
	}
}
