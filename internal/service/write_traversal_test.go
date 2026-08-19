package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEscritaRecusaTravessiaComSeparadorDoWindows e o teste que faltava.
//
// A forma com barra invertida NAO e coberta hoje, e e a unica que escapa: em
// Linux a barra invertida nao e separador, entao o mesmo caso de teste roda nos
// dois sistemas e so significa alguma coisa no Windows. Ele fica sem
// runtime.GOOS de proposito — a asserção "recusou" vale nos dois, e no Linux
// ela e trivialmente verdadeira. Um teste que so roda no Windows e um teste que
// ninguem ve reprovar.
func TestEscritaRecusaTravessiaComSeparadorDoWindows(t *testing.T) {
	cofreRoot := t.TempDir()
	writeFile(t, cofreRoot, "existente.md", "conteudo")
	svc := newTestService(t, cofreRoot)

	casos := []string{
		`..\..\x.md`,
		`..\x.md`,
		`sub\..\..\x.md`,
		`../../x.md`,
		`/etc/passwd`,
		`C:\Windows\Temp\x.md`,
		"COM1",
		"nota\x00.md",
	}
	for _, c := range casos {
		t.Run(c, func(t *testing.T) {
			_, err := svc.CreateNote(context.Background(), CreateNoteRequest{Path: c, Content: "x"})
			if err == nil {
				t.Fatalf("CreateNote(%q) devolveu sucesso", c)
			}
			// A asserção mais forte: nada foi escrito FORA do cofre.
			// Sem isto, um erro devolvido depois da escrita passaria.
			entradas, _ := os.ReadDir(filepath.Dir(cofreRoot))
			for _, e := range entradas {
				if strings.HasSuffix(e.Name(), ".md") {
					t.Fatalf("CreateNote(%q) gravou %q fora do cofre", c, e.Name())
				}
			}
			// E o mesmo pelo destino de note_move.
			_, err = svc.MoveNote(context.Background(), MoveNoteRequest{From: "existente.md", To: c})
			if err == nil {
				t.Fatalf("MoveNote(to=%q) devolveu sucesso", c)
			}
		})
	}
}
