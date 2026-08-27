//go:build windows

package writer_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/writer"
)

// TestSweepAlcancaCaminhoAlemDeMaxPath fixa o alcance real da varredura — e
// documenta um achado da auditoria que se provou FALSO.
//
// P11 dizia que `SweepStaleTempFiles` varria sem o prefixo `\?\` e por isso
// pulava, em silêncio, subárvores além de MAX_PATH: o temporário de uma escrita
// interrompida ficaria lá para sempre, aparecendo no Explorer do dono sem
// explicação.
//
// Sondado em 2026-08-27: **o pacote `os` do Go aplica o prefixo sozinho**
// (`fixLongPath`). `MkdirAll`, `WriteFile` e `WalkDir` alcançaram 318
// caracteres sem prefixo nenhum. A correção chegou a ser escrita — prefixar a
// raiz — e a prova de mutação a reprovou: trocá-la por `vault.LongPath`, que
// para raiz curta é identidade e portanto o comportamento antigo, deixou este
// teste PASSANDO. Guarda que não muda resultado foi removida.
//
// O teste fica porque a premissa merece ficar fixada: se uma versão futura do
// Go, ou uma troca de `os` por syscall direta, quebrar esse alcance, é aqui que
// aparece.
func TestSweepAlcancaCaminhoAlemDeMaxPath(t *testing.T) {
	raiz := t.TempDir()

	// Monta uma árvore que passa de 260 caracteres. Cada nível usa o prefixo
	// longo para PODER ser criado — o teste não pode falhar na montagem do
	// cenário e ser lido como falha da regra.
	fundo := raiz
	for len(fundo) < 300 {
		fundo = filepath.Join(fundo, strings.Repeat("d", 40))
	}
	if err := os.MkdirAll(fundo, 0755); err != nil {
		t.Skipf("nao foi possivel criar arvore profunda (%d chars): %v", len(fundo), err)
	}

	temp := filepath.Join(fundo, writer.TempFilePrefix+"orfao")
	if err := os.WriteFile(temp, []byte("lixo"), 0644); err != nil {
		t.Skipf("nao foi possivel criar o temporario profundo: %v", err)
	}

	// Sem isto o teste passaria mesmo que a árvore não tivesse sido criada.
	if _, err := os.Stat(temp); err != nil {
		t.Fatalf("o temporario profundo nao existe: o cenario nao se montou: %v", err)
	}
	t.Logf("caminho com %d caracteres", len(temp))

	varr, err := writer.SweepStaleTempFiles(context.Background(), raiz)
	if err != nil {
		t.Fatalf("SweepStaleTempFiles: %v", err)
	}
	if varr.Inacessiveis != 0 {
		t.Errorf("%d subarvores inacessiveis: a varredura nao entrou na arvore profunda", varr.Inacessiveis)
	}
	if varr.Removidos != 1 {
		t.Errorf("removidos = %d, queria 1", varr.Removidos)
	}
	if _, err := os.Stat(temp); err == nil {
		t.Error("o temporario profundo SOBROU depois da varredura: ele ficaria no cofre para sempre")
	}
}
