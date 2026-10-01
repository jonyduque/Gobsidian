package console

import (
	"os"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/vazamentotest"
)

// TestMain reprova o pacote se sobrar goroutine vazada. Ver vazamentotest.
func TestMain(m *testing.M) { os.Exit(vazamentotest.Conferir(m.Run())) }
