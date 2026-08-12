//go:build windows

package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// marcarSomenteNuvem poe FILE_ATTRIBUTE_OFFLINE no arquivo e restaura no fim.
//
// E o unico atributo que serve: FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS nao e
// gravavel por SetFileAttributes — e o motivo de TestReadNoteCloudOnlyFails
// estar pulado —, e vault.IsCloudOnly aceita os dois. So Windows porque o
// atributo e do NTFS; fora dele IsCloudOnly devolve false por construcao e nao
// ha condicao para montar.
func marcarSomenteNuvem(t *testing.T, abs string) {
	t.Helper()
	p, err := windows.UTF16PtrFromString(vault.LongPath(abs))
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Skipf("nao foi possivel marcar FILE_ATTRIBUTE_OFFLINE: %v", err)
	}
	t.Cleanup(func() {
		_ = windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_NORMAL)
	})
}

// TestReplaceNotaSomenteNuvemEntraComoNotaSemAbrirOArquivo cobre a divergencia
// da Task 95 do lado que estava errado.
//
// Replace registrava o placeholder como Asset, e idx.Get devolvia false para
// um `.md` — a mesma entrada que Build registra como Note com CloudOnly.
//
// A prova de que o arquivo NAO foi aberto e mecanica, nao inferida do conteudo:
// o teste segura um handle exclusivo (dwShareMode = 0) sobre ele, e confere
// primeiro que esse handle de fato barra uma leitura. Com a trava provada, um
// Replace que abrisse o arquivo devolveria erro de compartilhamento; ele
// devolvendo nil e a afirmacao.
func TestReplaceNotaSomenteNuvemEntraComoNotaSemAbrirOArquivo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "nuvem.md", "---\naliases: [Nuvem]\n---\n# Titulo da nuvem\n\ncorpo\n")
	abs := filepath.Join(root, "nuvem.md")
	marcarSomenteNuvem(t, abs)

	p, err := windows.UTF16PtrFromString(vault.LongPath(abs))
	if err != nil {
		t.Fatal(err)
	}
	// Leitura E escrita: medido nesta maquina, um handle exclusivo que pede so
	// GENERIC_READ nao barra o os.Open do vault.ReadAll. Pedindo os dois, barra
	// — e a guarda logo abaixo confere isso em vez de confiar na regra.
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Skipf("nao foi possivel abrir o arquivo em modo exclusivo: %v", err)
	}
	t.Cleanup(func() { _ = windows.CloseHandle(h) })

	// Guarda da trava. Sem ela, um handle que nao barrasse nada faria o resto
	// do teste passar sem provar coisa nenhuma sobre abrir o arquivo.
	if _, err := os.ReadFile(abs); err == nil {
		t.Fatal("o handle exclusivo nao barrou a leitura; a prova de 'nao abriu' seria vazia")
	}

	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ix := index.New()
	if err := ix.Replace(context.Background(), v, "nuvem.md"); err != nil {
		t.Fatalf("Replace: %v — o arquivo foi aberto, e abrir um placeholder "+
			"dispara download sincrono", err)
	}

	n, ok := ix.Get("nuvem.md")
	if !ok {
		t.Fatal("Get devolveu false: Replace registrou o `.md` como Asset, nao como Note")
	}

	// Guarda da montagem: sem o atributo pegando, o teste mediria uma nota comum.
	if !n.CloudOnly {
		t.Fatal("a nota nao ficou marcada CloudOnly; o atributo nao pegou")
	}
	if n.Title != "" {
		t.Fatalf("Title = %q, quer vazio; o placeholder foi lido e parseado", n.Title)
	}
	if n.Hash != 0 {
		t.Fatalf("Hash = %d, quer 0; o placeholder foi lido", n.Hash)
	}
	if len(n.Headings) != 0 {
		t.Fatalf("Headings = %d, quer 0; o placeholder foi parseado", len(n.Headings))
	}
	if len(n.Aliases) != 0 {
		t.Fatalf("Aliases = %v, quer vazio; o frontmatter foi lido", n.Aliases)
	}

	if a := ix.AssetCount(); a != 0 {
		t.Fatalf("AssetCount = %d, quer 0; o `.md` entrou como anexo", a)
	}
	if c := ix.NoteCount(); c != 1 {
		t.Fatalf("NoteCount = %d, quer 1", c)
	}
}

// TestConstrucoesDoIndiceConcordamComPlaceholderDeNuvem e o mesmo diferencial
// campo a campo de TestConstrucoesDoIndiceConcordamCampoACampo, sobre o cofre
// que carrega a entrada onde as duas construcoes divergiam.
//
// Antes da Task 95 ele reprovava em Get: true por Build, false por Replace.
func TestConstrucoesDoIndiceConcordamComPlaceholderDeNuvem(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "comum.md", "# Comum\n\nVer [[nuvem]].\n")
	writeFile(t, root, "nuvem.md", "# Titulo da nuvem\n\ncorpo\n")
	writeFile(t, root, "Anexos/diagrama.png", "\x89PNG")
	marcarSomenteNuvem(t, filepath.Join(root, "nuvem.md"))

	// Guarda da montagem, antes de comparar: se o atributo nao pegasse, este
	// teste viraria uma copia do cross-platform e nao afirmaria nada sobre
	// placeholder de nuvem.
	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ixGuarda := index.New()
	if err := ixGuarda.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	n, ok := ixGuarda.Get("nuvem.md")
	if !ok || !n.CloudOnly {
		t.Fatal("a nota nao ficou marcada CloudOnly; o atributo nao pegou")
	}

	compararConstrucoes(t, root)
}
