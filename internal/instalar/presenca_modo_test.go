package instalar

import "testing"

// TestPresencaGravaOModo prova que o modo chega ao arquivo e volta na leitura.
// Sem ele o `doctor` volta a nao distinguir ponte de gravador, que era o
// defeito medido em 2026-09-14.
func TestPresencaGravaOModo(t *testing.T) {
	dir := t.TempDir()
	liberar, err := Registrar(dir, `C:\Cofre`, "serve", ModoPonte, "v1")
	if err != nil {
		t.Fatalf("Registrar() error = %v", err)
	}
	defer liberar()

	vivos, err := Vivos(dir)
	if err != nil {
		t.Fatalf("Vivos() error = %v", err)
	}
	if len(vivos) != 1 {
		t.Fatalf("Vivos() = %+v, esperado um processo", vivos)
	}
	if vivos[0].Modo != ModoPonte {
		t.Fatalf("modo lido = %q, esperado %q", vivos[0].Modo, ModoPonte)
	}
}
