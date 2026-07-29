package watcher_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/watcher"
)

func TestCorrelateRenames(t *testing.T) {
	tmp := t.TempDir()
	v, _ := vault.New(tmp)
	idx := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Setup: initial files
	content := []byte("some content")
	emptyContent := []byte("")

	os.WriteFile(filepath.Join(tmp, "note1.md"), content, 0644)
	os.WriteFile(filepath.Join(tmp, "empty1.md"), emptyContent, 0644)

	idx.Build(context.Background(), v)

	// Action: simulate rename note1.md -> note2.md
	os.Rename(filepath.Join(tmp, "note1.md"), filepath.Join(tmp, "note2.md"))
	// Simulate rename empty1.md -> empty2.md
	os.Rename(filepath.Join(tmp, "empty1.md"), filepath.Join(tmp, "empty2.md"))
	// Simulate modified file
	os.WriteFile(filepath.Join(tmp, "note3.md"), []byte("initial"), 0644)
	idx.Build(context.Background(), v)
	os.Rename(filepath.Join(tmp, "note3.md"), filepath.Join(tmp, "note4.md"))
	os.WriteFile(filepath.Join(tmp, "note4.md"), []byte("modified"), 0644)

	batch := []vault.CanonicalPath{
		"note1.md", "note2.md",
		"empty1.md", "empty2.md",
		"note3.md", "note4.md",
	}

	renames, nonRenames := watcher.CorrelateRenames(context.Background(), batch, v, idx, log)

	if len(renames) != 1 {
		t.Fatalf("Expected 1 rename, got %d", len(renames))
	}
	if renames[0].From != "note1.md" || renames[0].To != "note2.md" {
		t.Errorf("Unexpected rename correlation: %v", renames[0])
	}

	// Ensure empty and modified files fell back to nonRenames
	nonRenamesMap := make(map[vault.CanonicalPath]bool)
	for _, p := range nonRenames {
		nonRenamesMap[p] = true
	}

	expectedNonRenames := []vault.CanonicalPath{"empty1.md", "empty2.md", "note3.md", "note4.md"}
	for _, p := range expectedNonRenames {
		if !nonRenamesMap[p] {
			t.Errorf("Expected %s in nonRenames, not found", p)
		}
	}
}

func TestCorrelateRenames_AssetIsNeverCorrelated(t *testing.T) {
	tmp := t.TempDir()
	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Bytes identicos nos dois lados. Se o anexo for lido, o hash casa e a
	// correlacao acontece — que e exatamente o defeito que este teste pega.
	conteudo := []byte("# Nota\n\nconteudo qualquer\n")
	if err := os.WriteFile(filepath.Join(tmp, "origem.md"), conteudo, 0644); err != nil {
		t.Fatal(err)
	}
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	// origem.md sai do disco; diagrama.png entra com os MESMOS bytes.
	if err := os.Remove(filepath.Join(tmp, "origem.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "diagrama.png"), conteudo, 0644); err != nil {
		t.Fatal(err)
	}

	batch := []vault.CanonicalPath{"origem.md", "diagrama.png"}
	renames, nonRenames := watcher.CorrelateRenames(context.Background(), batch, v, idx, log)

	if len(renames) != 0 {
		t.Fatalf("anexo foi correlacionado como rename: %+v — o gate de classe nao "+
			"impediu a leitura, e um .png de 40 MB tocado pelo OneDrive seria lido "+
			"inteiro dentro do laco que serializa as escritas do indice", renames)
	}
	if len(nonRenames) != 2 {
		t.Errorf("nonRenames = %v, quer os dois caminhos exatamente uma vez", nonRenames)
	}
}

func TestCorrelateRenames_NoDuplicateOutput(t *testing.T) {
	tmp := t.TempDir()
	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Nota vazia indexada, depois removida: Hash != 0 mas Size == 0.
	if err := os.WriteFile(filepath.Join(tmp, "vazia.md"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}
	if err := os.Remove(filepath.Join(tmp, "vazia.md")); err != nil {
		t.Fatal(err)
	}
	// Nota vazia nova: len(data) == 0.
	if err := os.WriteFile(filepath.Join(tmp, "nova.md"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	batch := []vault.CanonicalPath{"vazia.md", "nova.md"}
	renames, nonRenames := watcher.CorrelateRenames(context.Background(), batch, v, idx, log)

	if len(renames) != 0 {
		t.Errorf("arquivos vazios nao podem correlacionar: %+v", renames)
	}
	// A assercao central: cada caminho aparece UMA vez. Com os dois lacos de
	// leitura da entrega original, isto devolvia 4 entradas, e o Apply chamava
	// Replace/Remove duas vezes por caminho.
	visto := map[vault.CanonicalPath]int{}
	for _, p := range nonRenames {
		visto[p]++
	}
	if len(nonRenames) != 2 {
		t.Fatalf("nonRenames = %v (len %d), quer 2 entradas distintas", nonRenames, len(nonRenames))
	}
	for p, n := range visto {
		if n != 1 {
			t.Errorf("%s aparece %d vezes em nonRenames; duplicata faz Replace rodar em dobro", p, n)
		}
	}
}

func TestCorrelateRenames_SingleReadPerPath(t *testing.T) {
	tmp := t.TempDir()
	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Setup: 2 notas no disco, 1 nota removida do disco mas no indice.
	if err := os.WriteFile(filepath.Join(tmp, "removida.md"), []byte("conteudo antigo"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "mantida.md"), []byte("conteudo mantido"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}
	if err := os.Remove(filepath.Join(tmp, "removida.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "nova_diferente.md"), []byte("conteudo novo diferente"), 0644); err != nil {
		t.Fatal(err)
	}

	batch := []vault.CanonicalPath{"removida.md", "mantida.md", "nova_diferente.md"}
	renames, nonRenames := watcher.CorrelateRenames(context.Background(), batch, v, idx, log)

	if len(renames) != 0 {
		t.Fatalf("esperava 0 renames, obteve %d", len(renames))
	}

	visto := map[vault.CanonicalPath]int{}
	for _, p := range nonRenames {
		visto[p]++
	}
	if len(nonRenames) != 3 {
		t.Fatalf("nonRenames = %v (len %d), quer 3 entradas distintas", nonRenames, len(nonRenames))
	}
	for p, n := range visto {
		if n != 1 {
			t.Errorf("%s aparece %d vezes em nonRenames", p, n)
		}
	}
}

func TestCorrelateRenames_WithBOM(t *testing.T) {
	tmp := t.TempDir()
	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Nota com BOM UTF-8 (\xEF\xBB\xBF)
	conteudoComBOM := []byte("\xEF\xBB\xBF# Nota com BOM\nConteudo da nota\n")
	if err := os.WriteFile(filepath.Join(tmp, "origem_bom.md"), conteudoComBOM, 0644); err != nil {
		t.Fatal(err)
	}
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	// Rename no disco: origem_bom.md -> destino_bom.md
	if err := os.Remove(filepath.Join(tmp, "origem_bom.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "destino_bom.md"), conteudoComBOM, 0644); err != nil {
		t.Fatal(err)
	}

	batch := []vault.CanonicalPath{"origem_bom.md", "destino_bom.md"}
	renames, nonRenames := watcher.CorrelateRenames(context.Background(), batch, v, idx, log)

	if len(renames) != 1 {
		t.Fatalf("esperava 1 rename para nota com BOM, obteve %d", len(renames))
	}
	if renames[0].From != "origem_bom.md" || renames[0].To != "destino_bom.md" {
		t.Errorf("rename inesperado: %+v", renames[0])
	}
	if len(nonRenames) != 0 {
		t.Errorf("nonRenames = %v, quer vazio", nonRenames)
	}
}

func TestCorrelateRenames_ReportsBacklinkCandidates(t *testing.T) {
	tmp := t.TempDir()
	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	origemContent := []byte("# Origem\n\nConteudo original\n")
	cContent := []byte("# Nota C\n\nLink para [[origem]] aqui.\n")

	if err := os.WriteFile(filepath.Join(tmp, "origem.md"), origemContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "c.md"), cContent, 0644); err != nil {
		t.Fatal(err)
	}

	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	// Rename origem.md -> destino.md
	if err := os.Remove(filepath.Join(tmp, "origem.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "destino.md"), origemContent, 0644); err != nil {
		t.Fatal(err)
	}

	batch := []vault.CanonicalPath{"origem.md", "destino.md"}
	renames, nonRenames := watcher.CorrelateRenames(context.Background(), batch, v, idx, log)

	if len(renames) != 1 {
		t.Fatalf("esperava 1 rename, obteve %d", len(renames))
	}
	if renames[0].From != "origem.md" || renames[0].To != "destino.md" {
		t.Errorf("rename retornado inesperado: %+v", renames[0])
	}
	if len(renames[0].Backlinks) != 1 {
		t.Fatalf("len(Backlinks) = %d, quer 1", len(renames[0].Backlinks))
	}
	if renames[0].Backlinks[0].From != "c.md" {
		t.Errorf("Backlinks[0].From = %s, quer c.md", renames[0].Backlinks[0].From)
	}
	if len(nonRenames) != 0 {
		t.Errorf("nonRenames = %v, quer 0 entradas", nonRenames)
	}
}

func TestCorrelateRenames_DoesNotWriteVault(t *testing.T) {
	tmp := t.TempDir()
	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	content := []byte("# Nota de teste\nConteudo para checar alteracao no disco.\n")
	if err := os.WriteFile(filepath.Join(tmp, "origem.md"), content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	// Rename no disco
	if err := os.Remove(filepath.Join(tmp, "origem.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "destino.md"), content, 0644); err != nil {
		t.Fatal(err)
	}

	// Snapshot do estado do cofre antes da chamada
	snapshotAntes := snapshotVault(t, tmp)

	batch := []vault.CanonicalPath{"origem.md", "destino.md"}
	renames, _ := watcher.CorrelateRenames(context.Background(), batch, v, idx, log)

	if len(renames) != 1 {
		t.Fatalf("esperava 1 rename, obteve %d", len(renames))
	}

	// Snapshot do estado do cofre depois da chamada
	snapshotDepois := snapshotVault(t, tmp)

	if len(snapshotAntes) != len(snapshotDepois) {
		t.Fatalf("tamanho do cofre mudou: antes %d, depois %d", len(snapshotAntes), len(snapshotDepois))
	}
	for path, fileInfoAntes := range snapshotAntes {
		fileInfoDepois, ok := snapshotDepois[path]
		if !ok {
			t.Fatalf("arquivo %s sumiu apos CorrelateRenames", path)
		}
		if fileInfoAntes.size != fileInfoDepois.size || !fileInfoAntes.mtime.Equal(fileInfoDepois.mtime) {
			t.Fatalf("arquivo %s foi modificado por CorrelateRenames: antes %+v, depois %+v", path, fileInfoAntes, fileInfoDepois)
		}
	}
}

type fileState struct {
	mtime time.Time
	size  int64
}

func snapshotVault(t *testing.T, root string) map[string]fileState {
	t.Helper()
	res := make(map[string]fileState)
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		res[rel] = fileState{mtime: info.ModTime(), size: info.Size()}
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotVault: %v", err)
	}
	return res
}
