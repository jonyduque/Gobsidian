// Command netcheck e o binario que `go vet -vettool` carrega para cobrar o
// RNF-30: nenhum pacote sob internal/ ou cmd/ importa net ou net/*. A regra em
// si vive em tools/netcheck; aqui so ha o embrulho de linha de comando que o
// scripts/check_net.ps1 compila e invoca nos tres GOOS.
package main

import (
	"github.com/jonyd/gobsidian/tools/netcheck"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(netcheck.Analyzer)
}
