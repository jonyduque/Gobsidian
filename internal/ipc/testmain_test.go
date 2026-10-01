package ipc

import (
	"os"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/vazamentotest"
)

// TestMain isola o diretorio de runtime da suite: ver RodarComRuntimeIsolado.
// E um por diretorio, e nao um por pacote de teste -- ipc e ipc_test compilam
// no mesmo binario, e o Go aceita um TestMain so.
func TestMain(m *testing.M) { os.Exit(vazamentotest.Conferir(RodarComRuntimeIsolado(m))) }
