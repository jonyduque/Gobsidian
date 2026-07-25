// Package netcheck implementa a verificação de RNF-30: nenhum pacote do
// produto importa rede. Ver PRD §6.4 para a formulação completa da garantia.
package netcheck

import (
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "netcheck",
	Doc:  "reporta importacao de pacotes de rede em codigo do produto",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if !isNetwork(path) {
				continue
			}
			pass.Reportf(imp.Pos(), "pacote de rede proibido: %s", path)
		}
	}
	return nil, nil
}

func isNetwork(path string) bool {
	return path == "net" || strings.HasPrefix(path, "net/")
}
