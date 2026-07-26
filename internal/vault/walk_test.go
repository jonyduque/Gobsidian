package vault_test

import (
	"context"
	"math"
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

	// Todo diretorio de excludedDirs recebe um arquivo .md que SERIA uma nota
	// se a poda nao acontecesse. Isso importa: o arquivo de exemplo original
	// de cada pasta (workspace.json, config, syncthing.db) nao tem extensao
	// de nota nem de anexo, entao o filtro de extensao (o "if !isNote &&
	// !isAsset { return nil }" no fim de Walk) ja o descartaria de qualquer
	// jeito — apagar a entrada correspondente de excludedDirs nao mudaria a
	// contagem, e a fixture "provaria" uma poda que na verdade nunca era
	// exercitada. Os arquivos .md abaixo sao o que faz a poda ter efeito
	// observavel.
	writeFile(t, root, ".obsidian/workspace.json", "{}")
	writeFile(t, root, ".obsidian/Nota.md", "# obsidian\n")
	writeFile(t, root, ".git/config", "[core]")
	writeFile(t, root, ".git/Nota.md", "# git\n")
	writeFile(t, root, ".trash/velha.md", "# velha\n")
	writeFile(t, root, ".stfolder/syncthing.db", "lixo-binario")
	writeFile(t, root, ".stfolder/Nota.md", "# stfolder\n")

	// Ramos de isNoise alcancaveis por efeito: cada um recebe um nome cuja
	// extensao e ".md" — isto e, um nome que SERIA classificado como nota se
	// isNoise nao o interceptasse primeiro. ".~lock.Penal.md#" virou
	// ".~lock.Penal.md" (sem o "#" final): com o "#", filepath.Ext devolve
	// ".md#", que o filtro de extensao ja rejeita sozinho, e apagar o ramo
	// ".~lock." de isNoise nao mudaria a contagem.
	writeFile(t, root, "~$temp.md", "lixo")
	writeFile(t, root, ".~lock.Penal.md", "usuario,host,pid,")
	writeFile(t, root, ".gobsidian-tmp-abc123.md", "escrita atomica em andamento")

	// Ramos de isNoise defensivos, NAO alcancaveis por efeito: nomes fixos
	// (desktop.ini, Thumbs.db, .DS_Store) ou uma extensao fora de assetExts
	// (.tmp) que o filtro de extensao ja descarta sozinho, com ou sem
	// isNoise. Mantidos aqui como documentacao de que esses arquivos existem
	// em cofres reais — mas apagar os ramos correspondentes em walk.go nao
	// faz este teste falhar. Ver o comentario acima deles em walk.go.
	writeFile(t, root, "desktop.ini", "[.ShellClassInfo]")
	writeFile(t, root, "Thumbs.db", "lixo-binario")
	writeFile(t, root, ".DS_Store", "lixo-binario")
	writeFile(t, root, "rascunho.tmp", "conteudo temporario")

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

// TestWalkFailsWhenRootRemoved e a regressao permanente para o Finding 1 do
// fix pass anterior: um cofre que existia em New e some antes de Walk (unidade
// removivel desconectada, pasta de nuvem movida, share de rede caido) precisa
// devolver um erro nao-nulo, nao sucesso silencioso com zero entradas. Sem
// este teste, nada no suite comittado pegaria uma regressao para o antigo
// "return nil" silencioso em walk.go.
func TestWalkFailsWhenRootRemoved(t *testing.T) {
	root := t.TempDir()

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}

	err = v.Walk(context.Background(), func(vault.Entry) error {
		t.Fatal("callback nao deveria ser chamado: a raiz nao existe mais")
		return nil
	})
	if err == nil {
		t.Fatal("Walk com raiz removida apos New deveria falhar, nao devolver sucesso com zero entradas")
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

// TestReadRangeRejectsOverflowPair confirms the pathological pair
// start=math.MinInt64, end=math.MaxInt64 is rejected without panicking.
// Before the fix, end-start wrapped around int64's range to a negative
// value, sailed past the "> maxReadRangeBytes" check (a negative number is
// never greater than a positive cap), and reached make([]byte, end-start)
// with a negative length — a guaranteed panic in the exact code path the cap
// exists to protect.
func TestReadRangeRejectsOverflowPair(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "0123456789")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = v.ReadRange(context.Background(), "A.md", math.MinInt64, math.MaxInt64)
	if err == nil {
		t.Fatal("ReadRange com par start/end que estoura int64 deveria falhar, nao entrar em panic")
	}
}
