package service

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// travessiaUniversal sao os caminhos que escapam do cofre em QUALQUER sistema.
var travessiaUniversal = []string{
	`../../x.md`,
	`/etc/passwd`,
	"nota\x00.md",
}

// travessiaSoNoWindows sao os caminhos que escapam SO no Windows.
//
// Em Linux e macOS a barra invertida e caractere legal de NOME de arquivo, nao
// separador: `..\x.md` e um unico componente chamado literalmente `..\x.md`, e
// criar isso dentro do cofre e comportamento correto. `COM1` e nome de
// dispositivo so no Windows; em Linux e um nome comum.
var travessiaSoNoWindows = []string{
	`..\..\x.md`,
	`..\x.md`,
	`sub\..\..\x.md`,
	`C:\Windows\Temp\x.md`,
	"COM1",
}

// TestEscritaRecusaTravessiaComSeparadorDoWindows cobre a forma com barra
// invertida, que e a que escapa.
//
// Fora do Windows o caso nao e PULADO, ele troca de asserção: onde o nome e
// legal, o produto tem de aceita-lo e gravar dentro do cofre. Recusar ali
// tornaria inalcancavel uma nota de nome legitimo.
func TestEscritaRecusaTravessiaComSeparadorDoWindows(t *testing.T) {
	for _, c := range travessiaUniversal {
		t.Run("universal/"+c, func(t *testing.T) {
			exigeRecusa(t, c)
		})
	}

	for _, c := range travessiaSoNoWindows {
		t.Run("windows/"+c, func(t *testing.T) {
			if runtime.GOOS == "windows" {
				exigeRecusa(t, c)
				return
			}
			exigeAceitacaoDentroDoCofre(t, c)
		})
	}
}

// exigeRecusa prova que o caminho e recusado E que nada foi escrito fora do
// cofre — um erro devolvido DEPOIS da escrita passaria so pela primeira metade.
func exigeRecusa(t *testing.T, caminho string) {
	t.Helper()
	cofreRoot := t.TempDir()
	writeFile(t, cofreRoot, "existente.md", "conteudo")
	svc := newTestService(t, cofreRoot)

	if _, err := svc.CreateNote(context.Background(), CreateNoteRequest{Path: caminho, Content: "x"}); err == nil {
		t.Fatalf("CreateNote(%q) devolveu sucesso", caminho)
	}
	exigeNadaForaDoCofre(t, cofreRoot, caminho)

	// E o mesmo pelo destino de note_move, que constroi o caminho por outra via.
	if _, err := svc.MoveNote(context.Background(), MoveNoteRequest{From: "existente.md", To: caminho}); err == nil {
		t.Fatalf("MoveNote(to=%q) devolveu sucesso", caminho)
	}
	exigeNadaForaDoCofre(t, cofreRoot, caminho)
}

// exigeAceitacaoDentroDoCofre e o contrapeso: sem ele, recusar barra invertida
// em todo sistema passaria no teste.
func exigeAceitacaoDentroDoCofre(t *testing.T, caminho string) {
	t.Helper()
	cofreRoot := t.TempDir()
	svc := newTestService(t, cofreRoot)

	if _, err := svc.CreateNote(context.Background(), CreateNoteRequest{Path: caminho, Content: "x"}); err != nil {
		t.Fatalf("CreateNote(%q) recusou um nome de arquivo LEGAL em %s: %v",
			caminho, runtime.GOOS, err)
	}
	if _, err := os.Stat(filepath.Join(cofreRoot, caminho)); err != nil {
		t.Fatalf("CreateNote(%q) devolveu sucesso e o arquivo nao esta dentro do cofre: %v",
			caminho, err)
	}
	exigeNadaForaDoCofre(t, cofreRoot, caminho)
}

// exigeNadaForaDoCofre confere que nenhum `.md` apareceu no diretorio ACIMA da
// raiz do cofre — o alvo de toda travessia com `..`.
func exigeNadaForaDoCofre(t *testing.T, cofreRoot, caminho string) {
	t.Helper()
	entradas, err := os.ReadDir(filepath.Dir(cofreRoot))
	if err != nil {
		t.Fatalf("lendo o diretorio acima do cofre: %v", err)
	}
	for _, e := range entradas {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Fatalf("caso %q gravou %q fora do cofre", caminho, e.Name())
		}
	}
}
