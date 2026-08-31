package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestMoveNoteSemCofreNaoStatCaminhoRelativo é o achado B16, e o defeito era
// silencioso e capaz de servir dado de OUTRO arquivo.
//
// `MoveNote` recebe o cofre opcionalmente. No ramo `v == nil` ela fazia
// `os.Stat(string(newPath))` — e `newPath` é um caminho CANÔNICO, relativo à
// raiz do cofre. Sem cofre, ele era resolvido contra o **diretório de trabalho
// do processo**. No melhor caso falhava; no pior encontrava um arquivo
// homônimo em outro lugar do disco e copiava `ModTime` e `Size` DELE para a
// nota — metadado de um arquivo que não tem relação nenhuma com o cofre.
//
// O cenário monta exatamente isso: um arquivo com o mesmo nome canônico no CWD,
// com tamanho inconfundível.
func TestMoveNoteSemCofreNaoStatCaminhoRelativo(t *testing.T) {
	// Um cofre de verdade para construir o índice.
	root := t.TempDir()
	writeFile(t, root, "origem.md", "---\ntitle: Origem\n---\n\ntexto\n")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	ix := index.New()
	if err := ix.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	antes, ok := ix.Get("origem.md")
	if !ok {
		t.Fatal("origem.md nao entrou no indice: o cenario nao exercita nada")
	}
	tamanhoOriginal := antes.Size

	// A ISCA: um arquivo com o nome canônico do DESTINO, no diretório de
	// trabalho do processo, com tamanho que não se confunde com nada.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	isca := filepath.Join(cwd, "destino-isca-b16.md")
	iscaConteudo := make([]byte, 999_331) // tamanho inconfundível
	if err := os.WriteFile(isca, iscaConteudo, 0644); err != nil {
		t.Skipf("nao foi possivel criar a isca no CWD: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(isca) })

	// Move SEM cofre, para o caminho canônico que casa a isca.
	ix.MoveNote(nil, "origem.md", "destino-isca-b16.md")

	depois, ok := ix.Get("destino-isca-b16.md")
	if !ok {
		t.Fatal("a nota nao apareceu no destino")
	}

	if depois.Size == int64(len(iscaConteudo)) {
		t.Errorf("Size = %d, exatamente o tamanho da isca no CWD\n"+
			"o Stat resolveu o caminho canonico contra o diretorio de trabalho "+
			"e copiou o metadado de um arquivo de fora do cofre", depois.Size)
	}
	if !depois.ModTime.IsZero() {
		t.Errorf("ModTime = %v, queria zero\n"+
			"sem cofre nao da para resolver o caminho absoluto, e zerar e a "+
			"resposta honesta para \"nao sei\" — ela forca reindexacao", depois.ModTime)
	}
	// O conteúdo indexado tem de continuar sendo o da nota, não o da isca.
	if depois.Size != tamanhoOriginal && depois.Size != 0 {
		t.Logf("Size = %d (era %d antes do move)", depois.Size, tamanhoOriginal)
	}
}
