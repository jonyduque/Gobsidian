package main

import (
	"os"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/ipc"
)

// TestMain isola o diretorio de runtime da suite: servePonte registra presenca
// e disca o socket do cofre, e o arquivo `serve.<pid>.presenca` caia no
// %LocalAppData% do usuario a cada rodada. Ver ipc.RodarComRuntimeIsolado.
func TestMain(m *testing.M) { os.Exit(ipc.RodarComRuntimeIsolado(m)) }
