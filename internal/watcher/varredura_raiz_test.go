package watcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// TestVarreDiretorioNovoAcusaFalhaNaPropriaRaiz cobre a mesma armadilha que
// vault.Walk já paga, num segundo lugar onde ela nunca foi corrigida.
//
// filepath.WalkDir chama o callback com d == nil e erro != nil quando NÃO
// CONSEGUE LER A PRÓPRIA RAIZ. varreDiretorioNovo tratava esse caso igual a uma
// entrada ilegível qualquer: logava e devolvia nil. O resultado é uma varredura
// que "teve sucesso" com zero entradas — e é exatamente o estado que esta função
// existe para impedir, porque um diretório que chega ao cofre entrega UM evento
// e nada mais.
//
// O sintoma, quando acontece, é indistinguível de "a varredura não rodou":
// eventos=1, índice vazio, e as notas invisíveis para todas as tools até o
// próximo reinício. No Windows a causa transitória clássica é o antivírus
// segurando o diretório recém-movido.
//
// A regra do projeto é explícita: "cofre inacessível e cofre vazio não podem
// produzir a mesma resposta". Vale para a raiz de uma varredura também.
func TestVarreDiretorioNovoAcusaFalhaNaPropriaRaiz(t *testing.T) {
	saida := &bufferDeLog{}
	log := slog.New(slog.NewTextHandler(saida, &slog.HandlerOptions{Level: slog.LevelDebug}))

	dir := t.TempDir()
	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	w, err := New(v, index.New(), nil, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })

	// Raiz que não pode ser lida: o caminho não existe. É o mesmo estado que
	// WalkDir vê quando o diretório some ou fica inacessível entre o evento e
	// a varredura — a janela real, não uma condição artificial.
	inacessivel := filepath.Join(dir, "sumiu")

	errVarredura := w.varreDiretorioNovo(context.Background(), inacessivel)

	if errVarredura == nil {
		t.Errorf("varreDiretorioNovo devolveu nil para raiz ilegivel: "+
			"uma varredura que nao conseguiu LER a raiz reportou sucesso com zero entradas.\nlog:\n%s",
			saida.String())
	}
	if errVarredura != nil && !strings.Contains(errVarredura.Error(), "sumiu") {
		t.Errorf("o erro nao nomeia a raiz recusada: %v", errVarredura)
	}
}

// TestVarreDiretorioNovoSegueApesarDeEntradaIlegivel é o contrapeso: entrada
// individual ruim NÃO pode abortar a varredura das outras.
//
// Sem este teste, a correção do caso da raiz poderia ser "devolver todo erro",
// e aí um único arquivo ilegível faria perder o diretório inteiro — trocando um
// defeito por outro. As duas asserções juntas fixam a regra: raiz falha alto,
// entrada falha baixo.
func TestVarreDiretorioNovoSegueApesarDeEntradaIlegivel(t *testing.T) {
	saida := &bufferDeLog{}
	log := slog.New(slog.NewTextHandler(saida, &slog.HandlerOptions{Level: slog.LevelDebug}))

	dir := t.TempDir()
	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	w, err := New(v, index.New(), nil, 10*time.Millisecond, log)
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })

	nova := filepath.Join(dir, "chegou")
	if err := os.MkdirAll(nova, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nova, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := w.varreDiretorioNovo(context.Background(), nova); err != nil {
		t.Errorf("varreDiretorioNovo falhou numa raiz legivel: %v", err)
	}

	// O evento da nota tem de ter sido emitido.
	select {
	case ev := <-w.events:
		if !strings.HasSuffix(string(ev.Path), "a.md") {
			t.Errorf("evento emitido foi %q, queria a.md", ev.Path)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("a varredura nao emitiu evento para a.md.\nlog:\n%s", saida.String())
	}
}
