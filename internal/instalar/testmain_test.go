package instalar

import (
	"os"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/ipc"
)

// TestMain isola o diretorio de runtime da suite: a trava global de instalacao
// e a presenca moram em ipc.RuntimeDir, e `instalacao.lock` caia no
// %LocalAppData% do usuario a cada rodada. Ver ipc.RodarComRuntimeIsolado.
func TestMain(m *testing.M) { os.Exit(ipc.RodarComRuntimeIsolado(m)) }
