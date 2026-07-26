package vault_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestWalkExcludesAndClassifies(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "Civil/PONTO 03.md", "# A\n")
	writeFile(t, root, "Penal/B.md", "# B\n")
	writeFile(t, root, "Anexos/diagrama.png", "\x89PNG")

	// Todo diretorio de excludedDirs, para que a poda seja exercitada em cada
	// um, nao apenas nos tres primeiros.
	writeFile(t, root, ".obsidian/workspace.json", "{}")
	writeFile(t, root, ".git/config", "[core]")
	writeFile(t, root, ".trash/velha.md", "# velha\n")
	writeFile(t, root, ".stfolder/syncthing.db", "lixo-binario")

	// Todo ramo de isNoise, para que cada padrao de arquivo de controle seja
	// exercitado, nao apenas o prefixo "~$" e "desktop.ini".
	writeFile(t, root, "desktop.ini", "[.ShellClassInfo]")
	writeFile(t, root, "~$temp.md", "lixo")
	writeFile(t, root, ".~lock.Penal.md#", "usuario,host,pid,")
	writeFile(t, root, "Thumbs.db", "lixo-binario")
	writeFile(t, root, ".DS_Store", "lixo-binario")
	writeFile(t, root, "rascunho.tmp", "conteudo temporario")
	writeFile(t, root, ".gobsidian-tmp-abc123.md", "escrita atomica em andamento")

	writeFile(t, root, "notas.txt", "nao e nota")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var notes, assets []string
	err = v.Walk(context.Background(), func(e vault.Entry) error {
		if e.IsNote {
			notes = append(notes, string(e.Path))
		} else {
			assets = append(assets, string(e.Path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	sort.Strings(notes)
	sort.Strings(assets)

	wantNotes := []string{"Civil/PONTO 03.md", "Penal/B.md"}
	if len(notes) != len(wantNotes) {
		t.Fatalf("notas = %v, quer %v", notes, wantNotes)
	}
	for i := range wantNotes {
		if notes[i] != wantNotes[i] {
			t.Errorf("notas[%d] = %q, quer %q", i, notes[i], wantNotes[i])
		}
	}

	wantAssets := []string{"Anexos/diagrama.png"}
	if len(assets) != len(wantAssets) || assets[0] != wantAssets[0] {
		t.Errorf("anexos = %v, quer %v", assets, wantAssets)
	}
}

func TestNewRejectsMissingRoot(t *testing.T) {
	if _, err := vault.New(filepath.Join(t.TempDir(), "nao-existe")); err == nil {
		t.Fatal("New com raiz inexistente deveria falhar")
	}
}

func TestReadRange(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "0123456789")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got, err := v.ReadRange(context.Background(), "A.md", 2, 5)
	if err != nil {
		t.Fatalf("ReadRange: %v", err)
	}
	if string(got) != "234" {
		t.Errorf("ReadRange = %q, quer %q", got, "234")
	}
}

// TestReadRangeRejectsHugeRange confirms that an absurd requested range is
// rejected before any allocation, rather than attempting to allocate a
// buffer sized by an unchecked end-start.
func TestReadRangeRejectsHugeRange(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "0123456789")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = v.ReadRange(context.Background(), "A.md", 0, 1<<40)
	if err == nil {
		t.Fatal("ReadRange com faixa absurda deveria falhar")
	}
}
