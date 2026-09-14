package doctor

import (
	"os"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/ipc"
)

// TestMain isola o diretorio de runtime da suite: a sonda do daemon resolve o
// socket por ipc.SocketPath. Ver ipc.RodarComRuntimeIsolado.
func TestMain(m *testing.M) { os.Exit(ipc.RodarComRuntimeIsolado(m)) }
