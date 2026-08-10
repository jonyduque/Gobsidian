package daemon

import (
	"os"
	"testing"
)

// lock_internal_test.go testa adquirirLock e lockPath de dentro do pacote:
// as duas sao white-box de proposito, porque o comportamento que importa
// aqui -- o arquivo de lock some quando liberar() e chamado -- nao da para
// observar so pelo resultado de EnsureStarted (ver lock_test.go para os
// testes de caixa-preta via EnsureStarted).

func TestAdquirirLockSegundaChamadaPerde(t *testing.T) {
	vault := t.TempDir()

	adquiriu1, liberar1, err := adquirirLock(vault)
	if err != nil {
		t.Fatalf("primeira adquirirLock: %v", err)
	}
	if !adquiriu1 {
		t.Fatal("primeira chamada deveria vencer a corrida (adquiriu1 = true)")
	}
	t.Cleanup(func() {
		if liberar1 != nil {
			liberar1()
		}
	})

	adquiriu2, liberar2, err := adquirirLock(vault)
	if err != nil {
		t.Fatalf("segunda adquirirLock: %v", err)
	}
	if adquiriu2 {
		t.Fatal("segunda chamada nao deveria vencer a corrida com o lock ainda ativo (adquiriu2 = true)")
	}
	if liberar2 != nil {
		liberar2()
	}
}

func TestAdquirirLockLiberarPermiteNovaTentativa(t *testing.T) {
	vault := t.TempDir()

	adquiriu1, liberar1, err := adquirirLock(vault)
	if err != nil || !adquiriu1 {
		t.Fatalf("primeira adquirirLock: adquiriu=%v err=%v", adquiriu1, err)
	}
	liberar1()

	path, err := lockPath(vault)
	if err != nil {
		t.Fatalf("lockPath: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("arquivo de lock ainda existe apos liberar(): err=%v", err)
	}

	adquiriu2, liberar2, err := adquirirLock(vault)
	if err != nil {
		t.Fatalf("segunda adquirirLock apos liberar: %v", err)
	}
	if !adquiriu2 {
		t.Fatal("segunda chamada deveria vencer a corrida apos o lock anterior ser liberado")
	}
	if liberar2 != nil {
		liberar2()
	}
}
