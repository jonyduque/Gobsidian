package hosts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

// Ambiente e tudo que a deteccao consulta do mundo de fora.
//
// Struct com funcoes, e nao chamadas diretas a os/exec: e o que torna a tabela
// de hosts TESTAVEL. Os 1 738 linhas de instalador que este pacote substitui
// nao tinham um unico teste, e eram o codigo que edita arquivos que o usuario
// escreveu a mao.
type Ambiente struct {
	Home         string // %USERPROFILE% no Windows, $HOME no resto
	AppData      string // %APPDATA%; vazio fora do Windows
	LocalAppData string // %LOCALAPPDATA%; vazio fora do Windows

	// Existe responde se um caminho existe. Injetavel para o teste montar um
	// mundo sem tocar na maquina.
	Existe func(caminho string) bool
	// TemComando responde se um executavel esta no PATH.
	TemComando func(nome string) bool
	// Rodar executa um comando externo. Os hosts que tem CLI proprio se
	// configuram assim -- e melhor que adivinhar um formato de arquivo que
	// muda entre versoes.
	Rodar func(nome string, args ...string) error
}

// AmbienteReal le o mundo de verdade.
func AmbienteReal() Ambiente {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return Ambiente{
		Home:         home,
		AppData:      os.Getenv("APPDATA"),
		LocalAppData: os.Getenv("LOCALAPPDATA"),
		Existe: func(caminho string) bool {
			if caminho == "" {
				return false
			}
			_, err := os.Stat(caminho)
			return err == nil
		},
		TemComando: func(nome string) bool {
			_, err := exec.LookPath(nome)
			return err == nil
		},
		Rodar: func(nome string, args ...string) error {
			saida, err := exec.Command(nome, args...).CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s: %w (%s)", nome, err, saida)
			}
			return nil
		},
	}
}

// Host e um destino de configuracao.
type Host struct {
	Chave string
	Nome  string

	// detectar diz se o host esta instalado nesta maquina.
	//
	// A deteccao olha para o EXECUTAVEL ou para a instalacao, nunca so para o
	// diretorio de configuracao. O instalador antigo registrava por que: na
	// maquina do dono existem ~/.cursor e ~/.codex sem que nenhum dos dois
	// produtos esteja instalado -- foram criados por outra ferramenta.
	// Detectar por diretorio de config ofereceria hosts que nao existem.
	detectar func(Ambiente) bool

	// configurar registra o gobsidian. Devolve o aviso que o usuario precisa
	// ler depois (tipicamente "reinicie o host").
	configurar func(Ambiente, Entrada) (string, error)
}

// Todos devolve a tabela inteira, em ordem estavel.
//
// Ordem fixa, e nao mapa: a lista vai para a tela do usuario, que escolhe por
// ela. Ordem que muda entre execucoes faz alguem marcar o host errado.
func Todos() []Host {
	hosts := []Host{
		{
			Chave: "claude-desktop",
			Nome:  "Claude Desktop",
			detectar: func(a Ambiente) bool {
				return a.Existe(diretorioDoClaudeDesktop(a)) || a.Existe(instalacaoDoClaudeDesktop(a))
			},
			configurar: func(a Ambiente, e Entrada) (string, error) {
				alvo := filepath.Join(diretorioDoClaudeDesktop(a), "claude_desktop_config.json")
				return "Reinicie o Claude Desktop para carregar o servidor.", Fundir(alvo, e)
			},
		},
		{
			Chave:    "claude-code",
			Nome:     "Claude Code (CLI)",
			detectar: func(a Ambiente) bool { return a.TemComando("claude") },
			configurar: func(a Ambiente, e Entrada) (string, error) {
				// O proprio CLI escreve a config dele -- melhor do que adivinhar
				// o formato e o arquivo, que mudam entre versoes.
				//
				// O remove vem antes porque `claude mcp add` RECUSA um nome que
				// ja existe: sem ele, rodar o instalador duas vezes falhava so
				// neste host, com "MCP server gobsidian already exists".
				_ = a.Rodar("claude", "mcp", "remove", ChaveDoServidor, "--scope", "user")
				args := append([]string{"mcp", "add", ChaveDoServidor, "--scope", "user", "--", e.Command}, e.Args...)
				return "Registrado no escopo de usuario.", a.Rodar("claude", args...)
			},
		},
		{
			Chave:    "gemini-cli",
			Nome:     "Gemini CLI",
			detectar: func(a Ambiente) bool { return a.TemComando("gemini") },
			configurar: func(a Ambiente, e Entrada) (string, error) {
				// O remove do gemini assume escopo de PROJETO por padrao, entao
				// o --scope tem de ser explicito nos dois lados.
				_ = a.Rodar("gemini", "mcp", "remove", ChaveDoServidor, "--scope", "user")
				args := append([]string{"mcp", "add", ChaveDoServidor, e.Command}, e.Args...)
				args = append(args, "--scope", "user")
				return "Registrado no escopo de usuario.", a.Rodar("gemini", args...)
			},
		},
		{
			Chave: "antigravity",
			Nome:  "Antigravity",
			detectar: func(a Ambiente) bool {
				return a.Existe(filepath.Join(a.LocalAppData, "Programs", "Antigravity"))
			},
			configurar: func(a Ambiente, e Entrada) (string, error) {
				alvo := filepath.Join(a.Home, ".gemini", "antigravity", "mcp_config.json")
				return "Reinicie o Antigravity.", Fundir(alvo, e)
			},
		},
		{
			Chave: "antigravity-ide",
			Nome:  "Antigravity IDE",
			detectar: func(a Ambiente) bool {
				return a.Existe(filepath.Join(a.LocalAppData, "Programs", "Antigravity IDE"))
			},
			configurar: func(a Ambiente, e Entrada) (string, error) {
				alvo := filepath.Join(a.Home, ".gemini", "antigravity-ide", "mcp_config.json")
				return "Reinicie o Antigravity IDE.", Fundir(alvo, e)
			},
		},
		{
			Chave:    "codex",
			Nome:     "Codex CLI",
			detectar: func(a Ambiente) bool { return a.TemComando("codex") },
			configurar: func(a Ambiente, e Entrada) (string, error) {
				_ = a.Rodar("codex", "mcp", "remove", ChaveDoServidor)
				args := append([]string{"mcp", "add", ChaveDoServidor, "--", e.Command}, e.Args...)
				return "Registrado em ~/.codex/config.toml.", a.Rodar("codex", args...)
			},
		},
		{
			Chave:    "vscode",
			Nome:     "VS Code",
			detectar: func(a Ambiente) bool { return a.TemComando("code") },
			configurar: func(a Ambiente, e Entrada) (string, error) {
				def, err := definicaoParaVSCode(e)
				if err != nil {
					return "", err
				}
				return "Registrado na configuracao de usuario do VS Code.", a.Rodar("code", "--add-mcp", def)
			},
		},
		{
			Chave: "cursor",
			Nome:  "Cursor",
			detectar: func(a Ambiente) bool {
				return a.TemComando("cursor") || a.Existe(filepath.Join(a.LocalAppData, "Programs", "cursor"))
			},
			configurar: func(a Ambiente, e Entrada) (string, error) {
				return "Reinicie o Cursor.", Fundir(filepath.Join(a.Home, ".cursor", "mcp.json"), e)
			},
		},
		{
			Chave: "windsurf",
			Nome:  "Windsurf",
			detectar: func(a Ambiente) bool {
				return a.Existe(filepath.Join(a.Home, ".codeium", "windsurf"))
			},
			configurar: func(a Ambiente, e Entrada) (string, error) {
				alvo := filepath.Join(a.Home, ".codeium", "windsurf", "mcp_config.json")
				return "Reinicie o Windsurf.", Fundir(alvo, e)
			},
		},
	}
	sort.SliceStable(hosts, func(i, j int) bool { return hosts[i].Chave < hosts[j].Chave })
	return hosts
}

// Detectar devolve os hosts instalados nesta maquina.
func Detectar(a Ambiente) []Host {
	var achados []Host
	for _, h := range Todos() {
		if h.detectar != nil && h.detectar(a) {
			achados = append(achados, h)
		}
	}
	return achados
}

// PorChave acha um host pela chave. Devolve false quando a chave nao existe --
// o chamador precisa distinguir "nao instalado" de "nao existe esse host", e as
// duas coisas pedem mensagens diferentes.
func PorChave(chave string) (Host, bool) {
	for _, h := range Todos() {
		if h.Chave == chave {
			return h, true
		}
	}
	return Host{}, false
}

// Configurar registra o gobsidian neste host.
func (h Host) Configurar(a Ambiente, e Entrada) (aviso string, err error) {
	if h.configurar == nil {
		return "", fmt.Errorf("host %s nao sabe se configurar", h.Chave)
	}
	return h.configurar(a, e)
}

// Chaves devolve as chaves de todos os hosts conhecidos, para mensagens de
// ajuda e validacao de flag.
func Chaves() []string {
	var chaves []string
	for _, h := range Todos() {
		chaves = append(chaves, h.Chave)
	}
	return chaves
}
