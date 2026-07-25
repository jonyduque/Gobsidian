package netcheck_test

import (
	"testing"

	"github.com/jonyd/gobsidian/tools/netcheck"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestNetCheck(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), netcheck.Analyzer, "a")
}
