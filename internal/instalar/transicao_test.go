package instalar

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDaemonNoDiretorioAchaSoDaemonVivoDoMesmoCofre e G2.3.
//
// Cada caso usa o proprio diretorio: a presenca e um arquivo por papel e PID, e
// o mesmo processo de teste nao consegue registrar dois `daemon` no mesmo
// diretorio -- a primeira redacao deste teste tentou e a segunda trava foi
// recusada, como devia.
func TestDaemonNoDiretorioAchaSoDaemonVivoDoMesmoCofre(t *testing.T) {
	t.Run("ponte do mesmo cofre nao e daemon anterior", func(t *testing.T) {
		dir := t.TempDir()
		liberar, err := Registrar(dir, `C:\Cofre`, "serve", ModoPonte, "v1")
		if err != nil {
			t.Fatalf("Registrar(serve) error = %v", err)
		}
		defer liberar()
		if _, ok := daemonNoDiretorio(dir, `C:\Cofre`); ok {
			t.Fatal("uma ponte foi tomada por daemon de versao anterior")
		}
	})

	t.Run("daemon de outro cofre nao conta", func(t *testing.T) {
		dir := t.TempDir()
		liberar, err := Registrar(dir, `C:\Outro`, "daemon", ModoDaemon, "v1")
		if err != nil {
			t.Fatalf("Registrar(daemon de outro cofre) error = %v", err)
		}
		defer liberar()
		if _, ok := daemonNoDiretorio(dir, `C:\Cofre`); ok {
			t.Fatal("o daemon de OUTRO cofre foi tomado pelo deste")
		}
	})

	t.Run("daemon vivo do mesmo cofre conta, e morto deixa de contar", func(t *testing.T) {
		dir := t.TempDir()
		liberar, err := Registrar(dir, `C:\Cofre`, "daemon", ModoDaemon, "v1")
		if err != nil {
			t.Fatalf("Registrar(daemon) error = %v", err)
		}
		pid, ok := daemonNoDiretorio(dir, `c:\cofre\`)
		if !ok || pid != os.Getpid() {
			t.Fatalf("daemonNoDiretorio() = (%d, %v), esperado (%d, true) -- mesma pasta com outra caixa e barra final", pid, ok, os.Getpid())
		}
		liberar()
		if _, ok := daemonNoDiretorio(dir, `C:\Cofre`); ok {
			t.Fatal("daemon encerrado continuou contando")
		}
	})
}

// TestLimparVarreODiretorioAntigo e G2.4: o lixo do diretorio de runtime de
// antes de 2026-09-14 continua coberto pela limpeza.
func TestLimparVarreODiretorioAntigo(t *testing.T) {
	base := t.TempDir()
	runtimeNovo := filepath.Join(base, "novo")
	antigo := filepath.Join(base, "antigo")
	for _, d := range []string{runtimeNovo, antigo} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	lixo := filepath.Join(antigo, "0123456789abcdef.sock.lock")
	if err := os.WriteFile(lixo, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	original := diretorioAntigoFn
	diretorioAntigoFn = func() string { return antigo }
	t.Cleanup(func() { diretorioAntigoFn = original })

	r, err := Limpar(runtimeNovo, filepath.Join(base, "cache"), true)
	if err != nil {
		t.Fatalf("Limpar() error = %v", err)
	}
	if _, err := os.Stat(lixo); !os.IsNotExist(err) {
		t.Fatalf("trava orfa do diretorio antigo sobreviveu a limpeza (relatorio %+v)", r)
	}
}
