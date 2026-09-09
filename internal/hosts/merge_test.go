package hosts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func entradaDeTeste() Entrada {
	return Entrada{Command: `C:\bin\gobsidian.exe`, Args: []string{"serve", "--vault", `C:\Cofre`}}
}

// TestFundirPreservaTudoQueJaEstavaLa e a razao de existir deste arquivo.
//
// O `claude_desktop_config.json` da maquina do dono carrega `preferences` e
// OUTRO servidor MCP ao lado do gobsidian. Sobrescrever apagaria os dois. O
// instalador em PowerShell ja fazia a fusao certa; o que faltava era um teste
// que reprovasse se alguem trocasse a fusao por uma escrita.
func TestFundirPreservaTudoQueJaEstavaLa(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "claude_desktop_config.json")
	original := `{
  "preferences": {"tema": "escuro", "aninhado": {"lista": [1, 2, 3]}},
  "mcpServers": {
    "outro-servidor": {"command": "outro.exe", "args": ["--x"]}
  }
}`
	if err := os.WriteFile(caminho, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Fundir(caminho, entradaDeTeste()); err != nil {
		t.Fatalf("Fundir() error = %v", err)
	}

	var doc struct {
		Preferences struct {
			Tema     string `json:"tema"`
			Aninhado struct {
				Lista []int `json:"lista"`
			} `json:"aninhado"`
		} `json:"preferences"`
		McpServers map[string]Entrada `json:"mcpServers"`
	}
	lerJSON(t, caminho, &doc)

	if doc.Preferences.Tema != "escuro" {
		t.Errorf("preferences.tema virou %q; a fusao destruiu configuracao do usuario", doc.Preferences.Tema)
	}
	if len(doc.Preferences.Aninhado.Lista) != 3 {
		t.Errorf("preferences.aninhado.lista virou %v; campos que este pacote nao conhece tem de atravessar intactos", doc.Preferences.Aninhado.Lista)
	}
	if outro, ok := doc.McpServers["outro-servidor"]; !ok || outro.Command != "outro.exe" {
		t.Errorf("o outro servidor MCP do usuario sumiu: %+v", doc.McpServers)
	}
	nosso, ok := doc.McpServers[ChaveDoServidor]
	if !ok {
		t.Fatalf("a entrada do gobsidian nao foi escrita: %+v", doc.McpServers)
	}
	if nosso.Command != `C:\bin\gobsidian.exe` || strings.Join(nosso.Args, " ") != `serve --vault C:\Cofre` {
		t.Errorf("entrada do gobsidian errada: %+v", nosso)
	}
}

// TestFundirSubstituiAEntradaAntigaSemDuplicar: rodar o instalador duas vezes e
// o caso comum (atualizacao). A segunda passada tem de ATUALIZAR, nao acumular.
func TestFundirSubstituiAEntradaAntigaSemDuplicar(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "mcp.json")

	antiga := Entrada{Command: `C:\velho\gobsidian.exe`, Args: []string{"serve", "--vault", `C:\Antigo`}}
	if err := Fundir(caminho, antiga); err != nil {
		t.Fatalf("primeira Fundir(): %v", err)
	}
	if err := Fundir(caminho, entradaDeTeste()); err != nil {
		t.Fatalf("segunda Fundir(): %v", err)
	}

	var doc struct {
		McpServers map[string]Entrada `json:"mcpServers"`
	}
	lerJSON(t, caminho, &doc)

	if len(doc.McpServers) != 1 {
		t.Fatalf("mcpServers tem %d entradas, esperado 1: %+v", len(doc.McpServers), doc.McpServers)
	}
	if doc.McpServers[ChaveDoServidor].Command != `C:\bin\gobsidian.exe` {
		t.Errorf("a segunda passada nao atualizou o comando: %+v", doc.McpServers[ChaveDoServidor])
	}
}

// TestFundirCriaOArquivoEODiretorio: host recem-instalado ainda nao tem
// configuracao nenhuma.
func TestFundirCriaOArquivoEODiretorio(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "sub", "dir", "mcp.json")
	if err := Fundir(caminho, entradaDeTeste()); err != nil {
		t.Fatalf("Fundir() error = %v", err)
	}
	var doc struct {
		McpServers map[string]Entrada `json:"mcpServers"`
	}
	lerJSON(t, caminho, &doc)
	if _, ok := doc.McpServers[ChaveDoServidor]; !ok {
		t.Errorf("arquivo criado sem a entrada: %+v", doc)
	}
}

// TestFundirDeixaBackupAntesDeEscrever: o arquivo e do usuario e nao tem copia.
func TestFundirDeixaBackupAntesDeEscrever(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "mcp.json")
	original := `{"mcpServers":{"outro":{"command":"o.exe","args":[]}}}`
	if err := os.WriteFile(caminho, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Fundir(caminho, entradaDeTeste()); err != nil {
		t.Fatalf("Fundir() error = %v", err)
	}

	backup, err := os.ReadFile(caminho + SufixoDeBackup)
	if err != nil {
		t.Fatalf("o backup nao foi criado: %v", err)
	}
	if string(backup) != original {
		t.Errorf("o backup nao tem o conteudo ANTERIOR:\n%s", backup)
	}
}

// TestFundirNaoSobrescreveJSONInvalido e o caso que separa "cuidadoso" de
// "destrutivo".
//
// Um JSON invalido pode ser um arquivo que o usuario esta editando agora, ou um
// formato que este pacote nao entende. Nos dois casos, escrever por cima
// destroi o que havia. O instalador antigo, em PowerShell, deixava
// ConvertFrom-Json lancar -- e aqui o comportamento e explicito e testado.
func TestFundirNaoSobrescreveJSONInvalido(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "mcp.json")
	quebrado := `{"mcpServers": {` // truncado de proposito
	if err := os.WriteFile(caminho, []byte(quebrado), 0o600); err != nil {
		t.Fatal(err)
	}

	err := Fundir(caminho, entradaDeTeste())
	if err == nil {
		t.Fatal("Fundir() aceitou um JSON invalido")
	}

	depois, lerErr := os.ReadFile(caminho)
	if lerErr != nil {
		t.Fatalf("lendo o arquivo depois: %v", lerErr)
	}
	if string(depois) != quebrado {
		t.Errorf("o arquivo invalido foi ALTERADO:\n%s", depois)
	}
	// E o backup existe, para quem quiser voltar.
	if _, err := os.Stat(caminho + SufixoDeBackup); err != nil {
		t.Errorf("nao houve backup do arquivo invalido: %v", err)
	}
}

// TestFundirEmArquivoVazioNaoQuebra: alguns hosts criam o arquivo vazio na
// primeira execucao.
func TestFundirEmArquivoVazioNaoQuebra(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "mcp.json")
	if err := os.WriteFile(caminho, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Fundir(caminho, entradaDeTeste()); err != nil {
		t.Fatalf("Fundir() em arquivo vazio: %v", err)
	}
	var doc struct {
		McpServers map[string]Entrada `json:"mcpServers"`
	}
	lerJSON(t, caminho, &doc)
	if _, ok := doc.McpServers[ChaveDoServidor]; !ok {
		t.Errorf("entrada nao escrita em arquivo vazio: %+v", doc)
	}
}

func lerJSON(t *testing.T, caminho string, alvo any) {
	t.Helper()
	b, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("lendo %s: %v", caminho, err)
	}
	if err := json.Unmarshal(b, alvo); err != nil {
		t.Fatalf("o arquivo gravado nao e JSON valido: %v\n%s", err, b)
	}
}
