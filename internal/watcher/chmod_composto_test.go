package watcher

import (
	"io"
	"log/slog"
	"testing"

	"github.com/fsnotify/fsnotify"
)

// TestChmodCompostoNaoDerrubaOEvento é o achado M11.
//
// O teste era `e.Op&Chmod == Chmod`, verdadeiro para QUALQUER máscara que
// contenha Chmod. Um `Write|Chmod` — máscara composta, que é o padrão dos
// backends kqueue — era descartado inteiro, e a nota só voltava ao índice no
// próximo boot.
//
// O Windows contém o dano porque o backend dele raramente compõe máscaras, mas
// linux e darwin são alvos declarados e **não têm reconciliação por overflow**
// (kqueue não emite ErrEventOverflow), então lá não há rede de segurança
// nenhuma: o conteúdo simplesmente não entra.
func TestChmodCompostoNaoDerrubaOEvento(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	root := t.TempDir()

	casos := []struct {
		nome   string
		op     fsnotify.Op
		emitir bool
		motivo DropReason
		porque string
	}{
		{
			nome: "chmod sozinho e descartado", op: fsnotify.Chmod,
			emitir: false, motivo: DropChmod,
			porque: "mudanca de permissao pura nao muda conteudo e e muito comum",
		},
		{
			nome: "write com chmod e emitido", op: fsnotify.Write | fsnotify.Chmod,
			emitir: true,
			porque: "a mascara traz uma mudanca de CONTEUDO junto; descartar perde a edicao",
		},
		{
			nome: "create com chmod e emitido", op: fsnotify.Create | fsnotify.Chmod,
			emitir: true,
			porque: "idem: a nota nova entraria no indice so no proximo boot",
		},
		{
			nome: "write puro continua emitido", op: fsnotify.Write,
			emitir: true,
			porque: "contrapeso: a correcao nao pode ter mexido no caminho comum",
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			ev := fsnotify.Event{Name: root + string('/') + "nota.md", Op: c.op}
			_, emitiu, motivo := filter(ev, root, log)
			if emitiu != c.emitir {
				t.Errorf("emitiu = %v, queria %v (motivo=%q)\n%s", emitiu, c.emitir, motivo, c.porque)
			}
			if !c.emitir && motivo != c.motivo {
				t.Errorf("motivo = %q, queria %q", motivo, c.motivo)
			}
		})
	}
}
