package hosts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// ambienteFalso monta um mundo controlado: nada aqui toca a maquina de quem
// roda o teste.
//
// E o que os 1 738 linhas de instalador em PowerShell e Node nunca tiveram --
// e elas sao o codigo que edita arquivos que o usuario escreveu a mao e que
// encerra processos dele.
type ambienteFalso struct {
	Ambiente
	comandos [][]string
}

func novoAmbienteFalso(t *testing.T, existentes []string, comandos []string) *ambienteFalso {
	t.Helper()
	raiz := t.TempDir()
	af := &ambienteFalso{}
	af.Ambiente = Ambiente{
		Home:         filepath.Join(raiz, "home"),
		AppData:      filepath.Join(raiz, "appdata"),
		LocalAppData: filepath.Join(raiz, "local"),
		Existe: func(caminho string) bool {
			if caminho == "" {
				return false
			}
			return slices.Contains(existentes, caminho)
		},
		TemComando: func(nome string) bool { return slices.Contains(comandos, nome) },
		Rodar: func(nome string, args ...string) error {
			af.comandos = append(af.comandos, append([]string{nome}, args...))
			return nil
		},
	}
	return af
}

// TestTodosOsHostsTemChaveNomeEDeteccao: a tabela e consumida por flag
// (`--hosts`), por menu e por relatorio. Um host sem chave, sem nome ou sem
// deteccao apareceria como linha vazia num dos tres.
func TestTodosOsHostsTemChaveNomeEDeteccao(t *testing.T) {
	todos := Todos()
	if len(todos) != 9 {
		t.Fatalf("a tabela tem %d hosts, esperado 9 -- os mesmos que install.ps1 cobria", len(todos))
	}

	vistas := map[string]bool{}
	for _, h := range todos {
		if h.Chave == "" || h.Nome == "" {
			t.Errorf("host sem chave ou sem nome: %+v", h)
		}
		if h.detectar == nil {
			t.Errorf("host %s nao sabe se detectar", h.Chave)
		}
		if h.configurar == nil {
			t.Errorf("host %s nao sabe se configurar", h.Chave)
		}
		if vistas[h.Chave] {
			t.Errorf("chave duplicada: %s", h.Chave)
		}
		vistas[h.Chave] = true
	}

	// As nove chaves sao contrato: elas aparecem em `--hosts` e no README, e
	// renomear uma quebra a automacao de quem ja usa.
	esperadas := []string{
		"antigravity", "antigravity-ide", "claude-code", "claude-desktop",
		"codex", "cursor", "gemini-cli", "vscode", "windsurf",
	}
	if !slices.Equal(Chaves(), esperadas) {
		t.Errorf("Chaves() = %v, esperado %v", Chaves(), esperadas)
	}
}

// TestDetectarNaoOferecemHostQueNaoExiste e a licao que o instalador antigo
// deixou escrita: na maquina do dono existem ~/.cursor e ~/.codex sem que
// nenhum dos dois produtos esteja instalado -- foram criados por outra
// ferramenta. Detectar por diretorio de CONFIG ofereceria hosts inexistentes.
func TestDetectarNaoOferecemHostQueNaoExiste(t *testing.T) {
	af := novoAmbienteFalso(t, nil, nil)

	// Cria os diretorios de config que confundiriam uma deteccao ingenua.
	for _, d := range []string{".cursor", ".codex"} {
		if err := os.MkdirAll(filepath.Join(af.Home, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}

	if achados := Detectar(af.Ambiente); len(achados) != 0 {
		var nomes []string
		for _, h := range achados {
			nomes = append(nomes, h.Chave)
		}
		t.Fatalf("Detectar() ofereceu hosts que nao estao instalados: %v", nomes)
	}
}

// TestDetectarAchaOQueEstaInstalado: o outro lado. Sem ele, um Detectar que
// nunca acha nada passaria no teste acima.
func TestDetectarAchaOQueEstaInstalado(t *testing.T) {
	af := novoAmbienteFalso(t, nil, []string{"claude", "code"})
	cursorDir := filepath.Join(af.LocalAppData, "Programs", "cursor")
	af.Existe = func(caminho string) bool { return caminho == cursorDir }

	var chaves []string
	for _, h := range Detectar(af.Ambiente) {
		chaves = append(chaves, h.Chave)
	}

	for _, esperado := range []string{"claude-code", "cursor", "vscode"} {
		if !slices.Contains(chaves, esperado) {
			t.Errorf("Detectar() nao achou %s; achou %v", esperado, chaves)
		}
	}
	if slices.Contains(chaves, "windsurf") {
		t.Errorf("Detectar() achou windsurf sem ele estar instalado: %v", chaves)
	}
}

// TestConfigurarHostDeArquivoEscreveNoLugarCerto cobre os cinco hosts que se
// configuram por JSON.
func TestConfigurarHostDeArquivoEscreveNoLugarCerto(t *testing.T) {
	e := Entrada{Command: `C:\bin\gobsidian.exe`, Args: []string{"serve", "--vault", `C:\Cofre`}}

	casos := []struct {
		chave    string
		relativo []string // a partir de Home, exceto claude-desktop
	}{
		{"antigravity", []string{".gemini", "antigravity", "mcp_config.json"}},
		{"antigravity-ide", []string{".gemini", "antigravity-ide", "mcp_config.json"}},
		{"cursor", []string{".cursor", "mcp.json"}},
		{"windsurf", []string{".codeium", "windsurf", "mcp_config.json"}},
	}

	for _, c := range casos {
		t.Run(c.chave, func(t *testing.T) {
			af := novoAmbienteFalso(t, nil, nil)
			h, ok := PorChave(c.chave)
			if !ok {
				t.Fatalf("host %s sumiu da tabela", c.chave)
			}

			aviso, err := h.Configurar(af.Ambiente, e)
			if err != nil {
				t.Fatalf("Configurar() error = %v", err)
			}
			if aviso == "" {
				t.Error("Configurar() nao devolveu aviso; o usuario nao sabe que precisa reiniciar o host")
			}

			alvo := filepath.Join(append([]string{af.Home}, c.relativo...)...)
			var doc struct {
				McpServers map[string]Entrada `json:"mcpServers"`
			}
			b, err := os.ReadFile(alvo)
			if err != nil {
				t.Fatalf("nada foi escrito em %s: %v", alvo, err)
			}
			if err := json.Unmarshal(b, &doc); err != nil {
				t.Fatalf("o arquivo escrito nao e JSON: %v", err)
			}
			if doc.McpServers[ChaveDoServidor].Command != e.Command {
				t.Errorf("entrada errada em %s: %+v", alvo, doc.McpServers)
			}
		})
	}

	// claude-desktop tem caminho que varia por plataforma; confere so que
	// escreveu ALGO, no diretorio que a plataforma corrente define.
	t.Run("claude-desktop", func(t *testing.T) {
		af := novoAmbienteFalso(t, nil, nil)
		h, _ := PorChave("claude-desktop")
		if _, err := h.Configurar(af.Ambiente, e); err != nil {
			t.Fatalf("Configurar() error = %v", err)
		}
		alvo := filepath.Join(diretorioDoClaudeDesktop(af.Ambiente), "claude_desktop_config.json")
		if _, err := os.Stat(alvo); err != nil {
			t.Fatalf("nada escrito em %s (GOOS=%s): %v", alvo, runtime.GOOS, err)
		}
	})
}

// TestConfigurarHostDeCLIRemoveAntesDeAdicionar: `claude mcp add` RECUSA um
// nome que ja existe, e rodar o instalador duas vezes e o caso comum
// (atualizacao). Sem o remove, so este host falhava, com "MCP server gobsidian
// already exists in user config".
func TestConfigurarHostDeCLIRemoveAntesDeAdicionar(t *testing.T) {
	e := Entrada{Command: `C:\bin\gobsidian.exe`, Args: []string{"serve", "--vault", `C:\Cofre`}}

	for _, chave := range []string{"claude-code", "gemini-cli", "codex"} {
		t.Run(chave, func(t *testing.T) {
			af := novoAmbienteFalso(t, nil, nil)
			h, _ := PorChave(chave)
			if _, err := h.Configurar(af.Ambiente, e); err != nil {
				t.Fatalf("Configurar() error = %v", err)
			}
			if len(af.comandos) < 2 {
				t.Fatalf("esperado remove seguido de add, veio %v", af.comandos)
			}
			if !slices.Contains(af.comandos[0], "remove") {
				t.Errorf("o primeiro comando nao foi o remove: %v", af.comandos[0])
			}
			if !slices.Contains(af.comandos[1], "add") {
				t.Errorf("o segundo comando nao foi o add: %v", af.comandos[1])
			}
			// O caminho do executavel e os args do serve chegam intactos --
			// o "--" existe para que --vault nao seja lido como flag do host.
			ultimo := strings.Join(af.comandos[1], " ")
			if !strings.Contains(ultimo, e.Command) || !strings.Contains(ultimo, "--vault") {
				t.Errorf("o comando de add perdeu o executavel ou os args: %v", af.comandos[1])
			}
		})
	}
}

// TestVSCodeRecebeUmJSONNumArgumentoSo: `code --add-mcp` espera o servidor
// inteiro como UM argumento JSON.
func TestVSCodeRecebeUmJSONNumArgumentoSo(t *testing.T) {
	af := novoAmbienteFalso(t, nil, nil)
	e := Entrada{Command: `C:\bin\gobsidian.exe`, Args: []string{"serve", "--vault", `C:\Cofre`}}

	h, _ := PorChave("vscode")
	if _, err := h.Configurar(af.Ambiente, e); err != nil {
		t.Fatalf("Configurar() error = %v", err)
	}
	if len(af.comandos) != 1 || len(af.comandos[0]) != 3 {
		t.Fatalf("esperado `code --add-mcp <json>`, veio %v", af.comandos)
	}

	var def struct {
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.Unmarshal([]byte(af.comandos[0][2]), &def); err != nil {
		t.Fatalf("o argumento nao e JSON valido: %v -- %q", err, af.comandos[0][2])
	}
	if def.Name != ChaveDoServidor || def.Command != e.Command || len(def.Args) != 3 {
		t.Errorf("definicao errada: %+v", def)
	}
}
