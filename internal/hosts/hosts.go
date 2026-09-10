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
	//
	// Recebe a LISTA de entradas porque a instalacao configura N cofres desde
	// 2026-09-09. Lista vazia e uma escolha valida: significa "tire as minhas
	// entradas e nao ponha nenhuma".
	configurar func(Ambiente, []EntradaNomeada) (string, error)

	// arquivo diz ONDE mora a configuracao deste host, quando ela mora num
	// arquivo que este pacote escreve. Nulo nos hosts de CLI: la a
	// configuracao e do proprio CLI, e adivinhar o formato dela e o que
	// configurar por CLI existe para evitar.
	arquivo func(Ambiente) string
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
			arquivo: func(a Ambiente) string {
				return filepath.Join(diretorioDoClaudeDesktop(a), "claude_desktop_config.json")
			},
			configurar: func(a Ambiente, ens []EntradaNomeada) (string, error) {
				alvo := filepath.Join(diretorioDoClaudeDesktop(a), "claude_desktop_config.json")
				return "Reinicie o Claude Desktop para carregar o servidor.", FundirVarias(alvo, ens)
			},
		},
		{
			Chave:    "claude-code",
			Nome:     "Claude Code (CLI)",
			detectar: func(a Ambiente) bool { return a.TemComando("claude") },
			configurar: func(a Ambiente, ens []EntradaNomeada) (string, error) {
				// O proprio CLI escreve a config dele -- melhor do que adivinhar
				// o formato e o arquivo, que mudam entre versoes.
				//
				// O remove vem antes porque `claude mcp add` RECUSA um nome que
				// ja existe: sem ele, rodar o instalador duas vezes falhava so
				// neste host, com "MCP server gobsidian already exists".
				for _, chave := range chavesAPodar(ens) {
					_ = a.Rodar("claude", "mcp", "remove", chave, "--scope", "user")
				}
				return "Registrado no Claude Code.", registrarPorCLI(a, ens, func(en EntradaNomeada) []string {
					return append([]string{"mcp", "add", en.Chave, "--scope", "user", "--", en.Command}, en.Args...)
				}, "claude")
			},
		},
		{
			Chave:    "gemini-cli",
			Nome:     "Gemini CLI",
			detectar: func(a Ambiente) bool { return a.TemComando("gemini") },
			configurar: func(a Ambiente, ens []EntradaNomeada) (string, error) {
				// O remove do gemini assume escopo de PROJETO por padrao, entao
				// o --scope tem de ser explicito nos dois lados.
				for _, chave := range chavesAPodar(ens) {
					_ = a.Rodar("gemini", "mcp", "remove", chave, "--scope", "user")
				}
				return "Registrado no Gemini CLI.", registrarPorCLI(a, ens, func(en EntradaNomeada) []string {
					return append([]string{"mcp", "add", en.Chave, en.Command}, en.Args...)
				}, "gemini")
			},
		},
		{
			Chave: "antigravity",
			Nome:  "Antigravity",
			detectar: func(a Ambiente) bool {
				return a.Existe(filepath.Join(a.LocalAppData, "Programs", "Antigravity"))
			},
			arquivo: func(a Ambiente) string { return filepath.Join(a.Home, ".gemini", "antigravity", "mcp_config.json") },
			configurar: func(a Ambiente, ens []EntradaNomeada) (string, error) {
				alvo := filepath.Join(a.Home, ".gemini", "antigravity", "mcp_config.json")
				return "Reinicie o Antigravity.", FundirVarias(alvo, ens)
			},
		},
		{
			Chave: "antigravity-ide",
			Nome:  "Antigravity IDE",
			detectar: func(a Ambiente) bool {
				return a.Existe(filepath.Join(a.LocalAppData, "Programs", "Antigravity IDE"))
			},
			arquivo: func(a Ambiente) string { return filepath.Join(a.Home, ".gemini", "antigravity-ide", "mcp_config.json") },
			configurar: func(a Ambiente, ens []EntradaNomeada) (string, error) {
				alvo := filepath.Join(a.Home, ".gemini", "antigravity-ide", "mcp_config.json")
				return "Reinicie o Antigravity IDE.", FundirVarias(alvo, ens)
			},
		},
		{
			Chave:    "codex",
			Nome:     "Codex CLI",
			detectar: func(a Ambiente) bool { return a.TemComando("codex") },
			configurar: func(a Ambiente, ens []EntradaNomeada) (string, error) {
				for _, chave := range chavesAPodar(ens) {
					_ = a.Rodar("codex", "mcp", "remove", chave)
				}
				return "Registrado em ~/.codex/config.toml.", registrarPorCLI(a, ens, func(en EntradaNomeada) []string {
					return append([]string{"mcp", "add", en.Chave, "--", en.Command}, en.Args...)
				}, "codex")
			},
		},
		{
			Chave:    "vscode",
			Nome:     "VS Code",
			detectar: func(a Ambiente) bool { return a.TemComando("code") },
			configurar: func(a Ambiente, ens []EntradaNomeada) (string, error) {
				for _, en := range ens {
					def, err := definicaoParaVSCode(en)
					if err != nil {
						return "", err
					}
					if err := a.Rodar("code", "--add-mcp", def); err != nil {
						return "", err
					}
				}
				return "Registrado na configuracao de usuario do VS Code.", nil
			},
		},
		{
			Chave: "cursor",
			Nome:  "Cursor",
			detectar: func(a Ambiente) bool {
				return a.TemComando("cursor") || a.Existe(filepath.Join(a.LocalAppData, "Programs", "cursor"))
			},
			arquivo: func(a Ambiente) string { return filepath.Join(a.Home, ".cursor", "mcp.json") },
			configurar: func(a Ambiente, ens []EntradaNomeada) (string, error) {
				return "Reinicie o Cursor.", FundirVarias(filepath.Join(a.Home, ".cursor", "mcp.json"), ens)
			},
		},
		{
			Chave: "windsurf",
			Nome:  "Windsurf",
			detectar: func(a Ambiente) bool {
				return a.Existe(filepath.Join(a.Home, ".codeium", "windsurf"))
			},
			arquivo: func(a Ambiente) string { return filepath.Join(a.Home, ".codeium", "windsurf", "mcp_config.json") },
			configurar: func(a Ambiente, ens []EntradaNomeada) (string, error) {
				alvo := filepath.Join(a.Home, ".codeium", "windsurf", "mcp_config.json")
				return "Reinicie o Windsurf.", FundirVarias(alvo, ens)
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
func (h Host) Configurar(a Ambiente, ens []EntradaNomeada) (aviso string, err error) {
	if h.configurar == nil {
		return "", fmt.Errorf("host %s nao sabe se configurar", h.Chave)
	}
	return h.configurar(a, ens)
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

// chavesAPodar lista as chaves que um host de CLI precisa remover antes de
// registrar de novo.
//
// Sao as chaves NOVAS -- porque `mcp add` de vários CLIs recusa nome que já
// existe -- mais ChaveDoServidor, que e o que uma instalacao anterior de um
// cofre so deixou. Um CLI nao permite listar o que esta la, entao esta e a
// poda possivel: ela cobre o caminho de ida (um cofre -> varios) e a
// reconfiguracao do mesmo conjunto.
//
// O que ela NAO cobre e o caminho de volta com nomes diferentes: quem
// configurou "gobsidian-a" e depois so "gobsidian-b" fica com "gobsidian-a"
// no CLI ate remove-la a mao. Os hosts de arquivo JSON nao tem esse limite --
// la a poda le o que existe.
func chavesAPodar(ens []EntradaNomeada) []string {
	chaves := []string{ChaveDoServidor}
	for _, en := range ens {
		if en.Chave != ChaveDoServidor {
			chaves = append(chaves, en.Chave)
		}
	}
	return chaves
}

// registrarPorCLI roda um `mcp add` por entrada, parando no primeiro erro.
//
// montarArgs existe porque os tres CLIs querem a mesma coisa em ordens
// diferentes; o LAÇO e a parte comum, e e ele que estava escrito tres vezes.
func registrarPorCLI(a Ambiente, ens []EntradaNomeada, montarArgs func(EntradaNomeada) []string, comando string) error {
	for _, en := range ens {
		if err := a.Rodar(comando, montarArgs(en)...); err != nil {
			return err
		}
	}
	return nil
}

// EntradasAtuais devolve as entradas NOSSAS que ja estao na configuracao deste
// host.
//
// Vazio para host de CLI, e isso NAO e uma lacuna a preencher depois: ler a
// configuracao do Claude Code, do Gemini CLI ou do Codex exigiria adivinhar um
// formato que muda entre versoes, que e exatamente o que escrever pelo CLI
// existe para evitar. Quem chama trata "nao sei" como "nao sei", e nao como
// "nao ha".
func (h Host) EntradasAtuais(a Ambiente) []EntradaNomeada {
	if h.arquivo == nil {
		return nil
	}
	ens, err := LerEntradas(h.arquivo(a))
	if err != nil {
		return nil
	}
	return ens
}
