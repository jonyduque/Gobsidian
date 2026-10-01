package vazamentotest

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// vazarUma deixa uma goroutine presa num canal que nada mais alcanca: a forma
// exata que o perfil goroutineleak acusa.
func vazarUma() {
	ch := make(chan struct{})
	<-ch
}

// presaNumCanalVivo fica presa num canal que o teste ainda alcanca. Isso NAO e
// vazamento para o perfil, e a trava nao pode acusa-lo.
var canalVivo = make(chan struct{})

func presaNumCanalVivo() { <-canalVivo }

func esperarBloquear() { time.Sleep(50 * time.Millisecond) }

// TestVazadasAcusaGoroutinePresa: o que a trava existe para pegar. Sem esta
// prova, um Conferir que sempre devolvesse 0 passaria por todos os pacotes.
func TestVazadasAcusaGoroutinePresa(t *testing.T) {
	go vazarUma()
	go presaNumCanalVivo()
	t.Cleanup(func() { close(canalVivo) })
	esperarBloquear()

	relato, n := Vazadas()
	if n < 1 || !strings.Contains(relato, "vazamentotest.vazarUma") {
		t.Fatalf("goroutine presa em canal inalcancavel nao foi acusada: n=%d\n%s", n, relato)
	}
	if strings.Contains(relato, "presaNumCanalVivo") {
		t.Errorf("goroutine presa em canal ALCANCAVEL foi acusada:\n%s", relato)
	}

	// A mesma goroutine, nomeada como permitida, sai do relato.
	if relato, _ := Vazadas("vazamentotest.vazarUma"); strings.Contains(relato, "vazarUma") {
		t.Errorf("permitida nao filtrou:\n%s", relato)
	}

	// E Conferir transforma o vazamento em reprovacao, sem mascarar uma
	// reprovacao que ja existia.
	if c := conferirEm(io.Discard, 0, nil); c != 1 {
		t.Errorf("Conferir(0) com vazamento = %d, quer 1", c)
	}
	if c := conferirEm(io.Discard, 3, nil); c != 3 {
		t.Errorf("Conferir(3) = %d, quer o codigo original", c)
	}
	if c := conferirEm(io.Discard, 0, []string{"vazamentotest.vazarUma"}); c != 0 {
		t.Errorf("Conferir com a vazada permitida = %d, quer 0", c)
	}
}

// O vazamento proposital do teste acima e permitido aqui, e so ele.
func TestMain(m *testing.M) {
	os.Exit(Conferir(m.Run(), "vazamentotest.vazarUma"))
}

// TestTodoPacoteComTesteConfere e o gate da trava: pacote com teste que nao
// chama Conferir no TestMain fica fora dela em silencio, e o primeiro pacote
// novo seria esse. Ligado em 2026-10-01 nos 23 pacotes com teste.
func TestTodoPacoteComTesteConfere(t *testing.T) {
	raiz, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(raiz, "go.mod")); err != nil {
		t.Fatalf("raiz do modulo nao achada a partir de %s: %v", raiz, err)
	}
	comTeste := map[string]bool{}
	confere := map[string]bool{}
	err = filepath.WalkDir(raiz, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			nome := d.Name()
			if p != raiz && (strings.HasPrefix(nome, ".") || nome == "testdata" || nome == "node_modules") {
				return filepath.SkipDir
			}
			// Diretorio com go.mod proprio e outro modulo, fora do ./... -- o
			// mesmo corte que o go faz (agent/ e um deles).
			if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil && p != raiz {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		dir := filepath.Dir(p)
		comTeste[dir] = true
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "vazamentotest.Conferir(") || strings.Contains(string(b), "os.Exit(Conferir(") {
			confere[dir] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(comTeste) < 20 {
		t.Fatalf("so %d pacotes com teste achados a partir de %s: a varredura nao esta olhando o repositorio", len(comTeste), raiz)
	}
	for dir := range comTeste {
		if !confere[dir] {
			rel, _ := filepath.Rel(raiz, dir)
			t.Errorf("%s tem teste e nenhum TestMain com vazamentotest.Conferir", filepath.ToSlash(rel))
		}
	}
}
