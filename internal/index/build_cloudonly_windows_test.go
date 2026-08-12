//go:build windows

package index_test

import (
	"context"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestBuildNotaSomenteNuvemNaoDerrubaIndexacao cobre o panic que este arquivo
// existe para impedir.
//
// build.go manda placeholder de nuvem para o coletor com `parsed{entry: e}` e
// `note` NIL, de proposito: ler o arquivo dispararia download sincrono. O
// coletor entrava no ramo `r.entry.IsNote` — verdadeiro, e um `.md` — e
// desreferenciava `r.note.Title`. Um unico `.md` nao hidratado do OneDrive
// derrubava a indexacao inteira no boot, e o cofre em OneDrive e cenario
// suportado (docs/WINDOWS.md).
//
// Sobreviveu porque nenhum teste montava a condicao: o unico que tentava usa
// FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS, que nao e gravavel, e esta pulado.
// FILE_ATTRIBUTE_OFFLINE e gravavel por SetFileAttributes e vault.IsCloudOnly
// tambem o aceita — e por isso este teste consegue existir.
//
// So Windows porque o atributo e do NTFS; em outros sistemas IsCloudOnly
// devolve false por construcao e nao ha condicao para montar.
func TestBuildNotaSomenteNuvemNaoDerrubaIndexacao(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "comum.md", "# Titulo comum\n\ncorpo\n")
	writeFile(t, root, "nuvem.md", "# Titulo da nuvem\n\ncorpo\n")

	p, err := windows.UTF16PtrFromString(vault.LongPath(root + `\nuvem.md`))
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Skipf("nao foi possivel marcar FILE_ATTRIBUTE_OFFLINE: %v", err)
	}
	t.Cleanup(func() {
		_ = windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_NORMAL)
	})

	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	// Antes da correcao esta linha nao devolvia erro: entrava em panic.
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	nuvem, ok := idx.Get("nuvem.md")
	if !ok {
		t.Fatal("a nota somente-nuvem nao entrou no indice")
	}

	// Guarda da montagem. Sem o atributo pegando, o teste passaria a exercitar
	// uma nota comum e nao afirmaria nada sobre o caminho que ele cobre.
	if !nuvem.CloudOnly {
		t.Fatal("a nota nao ficou marcada CloudOnly; o atributo nao pegou")
	}

	// Title vazio e Hash zero sao a afirmacao de que o arquivo NAO foi lido —
	// ele tem um H1 no disco. Ler seria o download sincrono que a regra existe
	// para evitar.
	if nuvem.Title != "" {
		t.Fatalf("Title = %q, quer vazio; o placeholder foi lido e parseado", nuvem.Title)
	}
	if nuvem.Hash != 0 {
		t.Fatalf("Hash = %d, quer 0; o placeholder foi lido", nuvem.Hash)
	}
	if len(nuvem.Headings) != 0 {
		t.Fatalf("Headings = %d, quer 0; o placeholder foi parseado", len(nuvem.Headings))
	}

	// A nota comum precisa continuar completa: uma correcao que salvasse o
	// placeholder esvaziando todo mundo passaria nas assercoes acima.
	comum, ok := idx.Get("comum.md")
	if !ok {
		t.Fatal("a nota comum sumiu do indice")
	}
	if comum.Title != "Titulo comum" {
		t.Fatalf("Title da nota comum = %q, quer %q", comum.Title, "Titulo comum")
	}
	if comum.Hash == 0 {
		t.Fatal("Hash da nota comum = 0; ela deixou de ser lida")
	}

	if n := idx.NoteCount(); n != 2 {
		t.Fatalf("NoteCount = %d, quer 2", n)
	}
}
