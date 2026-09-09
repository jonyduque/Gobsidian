//go:build darwin

package hosts

import "path/filepath"

// Lado macOS. Ver caminhos_windows.go para por que isto fica atras de build
// tag em vez de um condicional dentro da tabela.
func diretorioDoClaudeDesktop(a Ambiente) string {
	return filepath.Join(a.Home, "Library", "Application Support", "Claude")
}

func instalacaoDoClaudeDesktop(_ Ambiente) string {
	return "/Applications/Claude.app"
}
