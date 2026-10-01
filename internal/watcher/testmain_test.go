package watcher

import (
	"os"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/vazamentotest"
)

func TestMain(m *testing.M) { os.Exit(vazamentotest.Conferir(m.Run())) }
