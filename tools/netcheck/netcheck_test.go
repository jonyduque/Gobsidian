package netcheck_test

import (
	"testing"

	"github.com/jonyd/gobsidian/tools/netcheck"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestNetCheck(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), netcheck.Analyzer, "a", "tcp", "redevar", "unixok", "outronet")
}

// TestNetcheckRecusaRedeVariavel isola o caso que a Task 90 existe para
// cobrir: net.Dial com a rede vinda de uma variável, não de um literal. É
// o teste nomeado na prova de mutação — se a condição em run() que compara
// ehConstante(arg0)/valorDe(arg0) for neutralizada, este teste, e só ele
// precisa falhar, é quem tem de acusar.
func TestNetcheckRecusaRedeVariavel(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), netcheck.Analyzer, "redevar")
}
