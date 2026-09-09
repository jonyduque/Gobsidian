//go:build windows

package hosts

import "path/filepath"

// diretorioDoClaudeDesktop e instalacaoDoClaudeDesktop sao o lado Windows dos
// caminhos que variam por plataforma. Tudo o mais na tabela de hosts e igual
// nas tres -- ou porque o host tem CLI proprio, ou porque o caminho e sob o
// home do usuario.
//
// Atras de build tag, em arquivo separado, e nunca `if runtime.GOOS ==` dentro
// da tabela: e a regra do CLAUDE.md, e ela existe para que a tabela continue
// legivel quando um quarto caminho aparecer.
func diretorioDoClaudeDesktop(a Ambiente) string {
	return filepath.Join(a.AppData, "Claude")
}

// instalacaoDoClaudeDesktop e onde o aplicativo em si fica. A deteccao olha os
// dois porque uma instalacao recem-feita ainda nao tem diretorio de config.
func instalacaoDoClaudeDesktop(a Ambiente) string {
	return filepath.Join(a.LocalAppData, "AnthropicClaude")
}
