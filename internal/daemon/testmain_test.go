package daemon

import (
	"os"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/ipc"
)

// TestMain isola o diretorio de runtime da suite: socket, trava e log do
// daemon derivam de ipc.SocketPath, e sem isto cada cofre de t.TempDir()
// deixava travas no %LocalAppData% do usuario. Ver ipc.RodarComRuntimeIsolado.
//
// Vale tambem para o ajudante de trava_test.go, que reexecuta este binario: o
// processo filho passa por aqui e ganha o proprio desvio.
func TestMain(m *testing.M) { os.Exit(ipc.RodarComRuntimeIsolado(m)) }
