//go:build !windows

package service_test

import (
	"os"
	"testing"
)

// tornaInescrivel impede a escrita de `arquivo` dentro de `dir`.
//
// Em Unix quem manda e a permissao do DIRETORIO, nao a do arquivo. O writer
// deste projeto e atomico: escreve num temporario e faz rename por cima. O
// rename precisa de permissao de escrita no diretorio e NAO consulta a
// permissao do arquivo de destino — entao `chmod 0400 b.md`, que era o que o
// teste fazia, nao impede nada aqui. O cenario de falha parcial simplesmente
// nao acontecia, e o teste reprovava logo no `if err == nil`.
//
// 0500 mantem leitura e travessia, e tira a criacao do temporario. O arquivo em
// si nao e tocado: quem falha e o passo de escrita, que e o ponto do cenario.
func tornaInescrivel(t *testing.T, dir, _ string) {
	t.Helper()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat %s: %v", dir, err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("Chmod %s: %v", dir, err)
	}
	// Sem o restauro, o t.TempDir() do proprio testing nao consegue remover o
	// diretorio no fim e o teste falha na limpeza, nao no que ele mede.
	t.Cleanup(func() { _ = os.Chmod(dir, info.Mode().Perm()) })
}
