//go:build !windows && !darwin

package hosts

import "path/filepath"

// Lado Linux e demais. Ver caminhos_windows.go para por que isto fica atras de
// build tag em vez de um condicional dentro da tabela.
//
// XDG_CONFIG_HOME nao entra aqui de proposito: o Claude Desktop para Linux nao
// o respeita hoje, e adivinhar um caminho que o host nao usa oferece um destino
// que nao funciona. Se isso mudar, muda aqui.
func diretorioDoClaudeDesktop(a Ambiente) string {
	return filepath.Join(a.Home, ".config", "Claude")
}

// Nao ha diretorio de instalacao canonico no Linux -- .deb, AppImage e Flatpak
// põem o aplicativo em lugares diferentes. Vazio faz Ambiente.Existe devolver
// false, e a deteccao cai para o diretorio de configuracao.
func instalacaoDoClaudeDesktop(_ Ambiente) string {
	return ""
}
