//go:build windows

package index_test

import (
	"testing"

	"github.com/jonyd/gobsidian/internal/vaulttest"
)

// lockFileForTest torna o arquivo ilegivel do jeito que o Windows permite: um
// handle exclusivo, segurado ate o fim do teste.
//
// A copia local que existia aqui abria o handle e nao conferia nada — se o
// share mode nao barrasse a leitura, TestBuildRegistraArquivoIlegivel
// exercitava um cofre de dois arquivos legiveis e passava. O helper de
// internal/vaulttest prova a trava antes de devolver.
func lockFileForTest(t *testing.T, path string) {
	t.Helper()
	vaulttest.TravarExclusivo(t, path)
}
