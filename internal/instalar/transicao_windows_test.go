package instalar

import (
	"os"
	"testing"
)

// TestDaemonNoDiretorioIgnoraACaixaNoWindows: no Windows `C:\Cofre` e
// `c:\cofre\` sao a mesma pasta, e um daemon registrado com uma grafia tem de
// ser achado pela outra. So aqui: em sistema de arquivos que distingue caixa,
// sao pastas diferentes (ver o caso nativo em transicao_test.go).
func TestDaemonNoDiretorioIgnoraACaixaNoWindows(t *testing.T) {
	dir := t.TempDir()
	liberar, err := Registrar(dir, `C:\Cofre`, "daemon", ModoDaemon, "v1")
	if err != nil {
		t.Fatalf("Registrar(daemon) error = %v", err)
	}
	defer liberar()
	pid, ok := daemonNoDiretorio(dir, `c:\cofre\`)
	if !ok || pid != os.Getpid() {
		t.Fatalf("daemonNoDiretorio() = (%d, %v), esperado (%d, true) -- mesma pasta com outra caixa e barra final", pid, ok, os.Getpid())
	}
}
