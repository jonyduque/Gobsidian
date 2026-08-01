package main

import (
	"github.com/jonyd/gobsidian/tools/netcheck"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(netcheck.Analyzer)
}
