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

// TestVivosComAnteriorVeOsDoisDiretorios: o `doctor` do binario novo listava os
// processos v1.8.1 como "sem presenca", porque eles registram no diretorio
// antigo (medido em 2026-09-14).
func TestVivosComAnteriorVeOsDoisDiretorios(t *testing.T) {
	base := t.TempDir()
	runtimeNovo := filepath.Join(base, "novo")
	antigo := filepath.Join(base, "antigo")

	original := diretorioAntigoFn
	t.Cleanup(func() { diretorioAntigoFn = original })

	liberarNovo, err := Registrar(runtimeNovo, `C:\Cofre`, "daemon", ModoDaemon, "v2")
	if err != nil {
		t.Fatalf("Registrar(novo) error = %v", err)
	}
	defer liberarNovo()
	liberarAntigo, err := Registrar(antigo, `C:\Cofre`, "serve", ModoEmProcesso, "v1")
	if err != nil {
		t.Fatalf("Registrar(antigo) error = %v", err)
	}
	defer liberarAntigo()

	t.Run("presenca do diretorio antigo entra", func(t *testing.T) {
		diretorioAntigoFn = func() string { return antigo }
		vivos, err := VivosComAnterior(runtimeNovo)
		if err != nil {
			t.Fatalf("VivosComAnterior() error = %v", err)
		}
		if len(vivos) != 2 || vivos[0].Versao != "v2" || vivos[1].Versao != "v1" {
			t.Fatalf("vivos = %+v, esperado a presenca v2 do novo e a v1 do antigo", vivos)
		}
	})

	t.Run("mesmo diretorio nao conta duas vezes", func(t *testing.T) {
		diretorioAntigoFn = func() string { return runtimeNovo + string(filepath.Separator) }
		vivos, err := VivosComAnterior(runtimeNovo)
		if err != nil || len(vivos) != 1 {
			t.Fatalf("VivosComAnterior() = (%+v, %v), esperado so a presenca do novo", vivos, err)
		}
	})

	t.Run("diretorio antigo ausente nao e erro", func(t *testing.T) {
		diretorioAntigoFn = func() string { return filepath.Join(base, "nunca-existiu") }
		vivos, err := VivosComAnterior(runtimeNovo)
		if err != nil || len(vivos) != 1 {
			t.Fatalf("VivosComAnterior() = (%+v, %v), esperado so a presenca do novo, sem erro", vivos, err)
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
