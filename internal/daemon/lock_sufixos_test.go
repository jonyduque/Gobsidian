package daemon

import (
	"path/filepath"
	"testing"
)

// lock_sufixos_test.go testa EhArquivoDeTrava e a conta unica que sufixoTrava
// e sufixoTravaDeEscuta representam. E pacote daemon (nao daemon_test) porque
// exercita os dois identificadores nao exportados diretamente -- ver
// lock_test.go para os testes de caixa-preta via EnsureStarted.

func TestEhArquivoDeTravaCobreAsDuasTravas(t *testing.T) {
	sock := "abc.sock"
	casos := map[string]bool{
		sock + sufixoTrava:         true,
		sock + sufixoTravaDeEscuta: true,
		sock:                       false,
		sock + ".log":              false,
		"abc.lock.txt":             false,
	}
	for nome, quer := range casos {
		if got := EhArquivoDeTrava(nome); got != quer {
			t.Errorf("EhArquivoDeTrava(%q) = %v, quer %v", nome, got, quer)
		}
	}
	// As constantes sao o que lockPath e ComLockDeEscuta usam de verdade.
	p, err := lockPath(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !EhArquivoDeTrava(filepath.Base(p)) {
		t.Fatalf("lockPath produz %q, que EhArquivoDeTrava nao reconhece", p)
	}
}
