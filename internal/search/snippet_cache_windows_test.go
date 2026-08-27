//go:build windows

package search_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestSnippetCacheNaoAbreNotaSomenteNuvem prova que o cache não moveu a saída
// antecipada do placeholder de nuvem.
//
// A prova é possível porque vault.IsCloudOnly aceita FILE_ATTRIBUTE_OFFLINE, e
// esse atributo — ao contrário de RECALL_ON_DATA_ACCESS, que é o motivo de
// TestReadNoteCloudOnlyFails estar pulado — é gravável por SetFileAttributes.
// O arquivo no disco tem o termo; se GenerateSnippet chegasse a abri-lo, o
// trecho viria preenchido. Ele vindo vazio é a afirmação.
//
// A segunda asserção é a que o cache introduziu: entrada de placeholder no
// cache faria a regra "nunca abre" depender de já ter aberto uma vez.
// É teste só de Windows porque o atributo é do NTFS.
func TestSnippetCacheNaoAbreNotaSomenteNuvem(t *testing.T) {
	root := t.TempDir()
	const conteudo = "alvo aaaaaaaaaaaaaaaaaaaa"
	caminho := filepath.Join(root, "nuvem.md")
	if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := windows.UTF16PtrFromString(caminho)
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
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}

	// Guarda da montagem: sem o atributo pegando, o teste mediria uma nota comum.
	nota, ok := idx.Get("nuvem.md")
	if !ok {
		t.Fatal("nota nao entrou no indice")
	}
	if !nota.CloudOnly {
		t.Fatal("a nota nao ficou marcada CloudOnly; o atributo nao pegou e o teste " +
			"passaria a medir o caminho comum")
	}

	ix := search.NewInverted()
	ix.Add("nuvem.md", search.Analyze(conteudo))
	cache := search.NewSnippetCache(16)

	trecho, err := search.GenerateSnippet(context.Background(), v, ix, idx, "nuvem.md",
		search.NovosTermosDeTrecho([]string{"alvo"}), 240, cache)
	if err != nil {
		t.Fatalf("GenerateSnippet: %v", err)
	}
	if trecho.Text != "" {
		t.Fatalf("trecho = %q, quer vazio; o placeholder foi aberto e a leitura "+
			"dispararia download sincrono", trecho.Text)
	}
	if strings.Contains(trecho.Text, "alvo") {
		t.Fatalf("trecho traz o conteudo do disco: %q", trecho.Text)
	}
	if cache.Len() != 0 {
		t.Fatalf("cache.Len() = %d, quer 0; placeholder de nuvem nao pode ganhar entrada", cache.Len())
	}
}
