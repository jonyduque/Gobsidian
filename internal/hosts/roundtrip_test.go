package hosts_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/hosts"
)

// O corpus e sobre o CONTEUDO que a fusao promete atravessar, e nao sobre os
// nove hosts: `Fundir` recebe o caminho por parametro e nao sabe de qual host
// ele e -- nove copias do mesmo caso provariam nove vezes a mesma coisa.
//
// Cada entrada aqui e uma forma que ja apareceu, ou que o Go 1.27 passou a
// tratar diferente. Ver TestFundirAtravessaOResto para o que se cobra de cada
// uma.
var corpus = []struct {
	nome string
	json string
}{
	{
		nome: "config real, com preferencias e outro servidor ao lado",
		json: `{
  "preferences": {"theme": "dark", "locale": "pt-BR"},
  "mcpServers": {
    "outro": {"command": "node", "args": ["x.js"]}
  }
}`,
	},
	{
		nome: "sem mcpServers",
		json: `{"preferences": {"theme": "dark"}}`,
	},
	{
		nome: "arquivo vazio como objeto",
		json: `{}`,
	},
	{
		nome: "acento e escape unicode no valor",
		json: `{"nota": "Ação e coração", "escapado": "Ação", "mcpServers": {}}`,
	},
	{
		nome: "numero que nao cabe em float64",
		json: `{"grande": 123456789012345678901234567890, "preciso": 0.1000000000000000055511151231257827, "mcpServers": {}}`,
	},
	{
		nome: "aninhamento fundo e array heterogeneo",
		json: `{"a": {"b": {"c": [1, "dois", null, true, {"d": []}]}}, "mcpServers": {}}`,
	},
	{
		nome: "chave vazia e chave com ponto",
		json: `{"": 1, "a.b": 2, "mcpServers": {}}`,
	},
	{
		// O v2 recusa chave duplicada; o v1 classico ficava com a ultima. Sob a
		// 1.27 o v1 roda sobre o v2, e o que este caso cobra e que a recusa NAO
		// vire sobrescrita: Fundir tem de sair sem tocar no arquivo.
		nome: "chave duplicada",
		json: `{"x": 1, "x": 2, "mcpServers": {}}`,
	},
}

// TestFundirAtravessaOResto e a cobranca do contrato de Fundir: tudo que nao e
// a entrada do gobsidian sai igual ao que entrou.
//
// Existe por causa do Go 1.27, que passou a implementar o encoding/json v1
// sobre a maquina do v2 (GOEXPERIMENT=nojsonv2 restaura a antiga). A
// compatibilidade e a intencao declarada, e este pacote escreve nos configs MCP
// REAIS do usuario -- ele ja reescreveu seis arquivos da maquina do dono. O que
// torna aquilo recuperavel e o .gobsidian-backup, nao a certeza de que o
// round-trip e fiel. Esta certeza e este teste.
//
// A comparacao e sobre o VALOR compactado, e nao sobre o arquivo inteiro:
// Fundir reindenta e ordena as chaves de topo por construcao (mapa + Marshal).
// Compactar remove essa diferenca e mantem a que interessa -- um byte de
// conteudo que mudou, um numero reformatado, um U+FFFD no lugar de um byte.
func TestFundirAtravessaOResto(t *testing.T) {
	for _, c := range corpus {
		t.Run(c.nome, func(t *testing.T) {
			caminho := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(caminho, []byte(c.json), 0o600); err != nil {
				t.Fatal(err)
			}

			antes := map[string]json.RawMessage{}
			legivel := json.Unmarshal([]byte(c.json), &antes) == nil

			err := hosts.Fundir(caminho, hosts.Entrada{Command: "gobsidian", Args: []string{"serve"}})
			if !legivel {
				if err == nil {
					t.Fatal("Fundir aceitou um JSON que ela mesma nao consegue ler: sobrescrever destroi o que havia")
				}
				depois, errLer := os.ReadFile(caminho)
				if errLer != nil {
					t.Fatal(errLer)
				}
				if string(depois) != c.json {
					t.Errorf("Fundir recusou e MESMO ASSIM escreveu:\n%s", depois)
				}
				return
			}
			if err != nil {
				t.Fatalf("Fundir: %v", err)
			}

			bruto, err := os.ReadFile(caminho)
			if err != nil {
				t.Fatal(err)
			}
			depois := map[string]json.RawMessage{}
			if err := json.Unmarshal(bruto, &depois); err != nil {
				t.Fatalf("Fundir gravou algo que nao se le de volta: %v\n%s", err, bruto)
			}

			for chave, valorAntes := range antes {
				if chave == "mcpServers" {
					continue
				}
				valorDepois, ok := depois[chave]
				if !ok {
					t.Errorf("a chave %q sumiu da fusao", chave)
					continue
				}
				if a, d := compactar(t, valorAntes), compactar(t, valorDepois); !bytes.Equal(a, d) {
					t.Errorf("a chave %q mudou de valor:\nantes:  %s\ndepois: %s", chave, a, d)
				}
			}

			// A entrada do gobsidian entrou, e os servidores que ja existiam
			// continuam la.
			var servidoresDepois map[string]json.RawMessage
			if err := json.Unmarshal(depois["mcpServers"], &servidoresDepois); err != nil {
				t.Fatalf("mcpServers nao voltou como objeto: %v", err)
			}
			if _, ok := servidoresDepois[hosts.ChaveDoServidor]; !ok {
				t.Errorf("a entrada do gobsidian nao foi escrita")
			}
			if bruta, ok := antes["mcpServers"]; ok {
				var servidoresAntes map[string]json.RawMessage
				if json.Unmarshal(bruta, &servidoresAntes) == nil {
					for nome, v := range servidoresAntes {
						if nome == hosts.ChaveDoServidor {
							continue
						}
						d, ok := servidoresDepois[nome]
						if !ok {
							t.Errorf("o servidor %q sumiu", nome)
							continue
						}
						if a, dd := compactar(t, v), compactar(t, d); !bytes.Equal(a, dd) {
							t.Errorf("o servidor %q mudou:\nantes:  %s\ndepois: %s", nome, a, dd)
						}
					}
				}
			}
		})
	}
}

// TestFundirPreservaBytesNaoUTF8 fica de fora do corpus porque a resposta certa
// depende do que a stdlib faz, e o teste existe para MEDIR isso, nao para
// impor. As duas respostas sao aceitaveis; a inaceitavel e a terceira.
//
//	recusar             -- o arquivo fica intacto, e o usuario e avisado
//	atravessar          -- os bytes saem como entraram
//	trocar por U+FFFD   -- INACEITAVEL: corrompe em silencio o config do usuario
func TestFundirPreservaBytesNaoUTF8(t *testing.T) {
	// 0xFF nunca e UTF-8 valido.
	original := "{\"nota\": \"a\xffb\", \"mcpServers\": {}}"
	caminho := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(caminho, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	err := hosts.Fundir(caminho, hosts.Entrada{Command: "gobsidian", Args: []string{"serve"}})
	bruto, errLer := os.ReadFile(caminho)
	if errLer != nil {
		t.Fatal(errLer)
	}

	if err != nil {
		if string(bruto) != original {
			t.Errorf("Fundir recusou e mesmo assim escreveu:\n%s", bruto)
		}
		t.Logf("MEDIDO: Fundir RECUSA byte nao-UTF-8 (%v) e deixa o arquivo intacto", err)
		return
	}

	if bytes.Contains(bruto, []byte("�")) {
		t.Errorf("Fundir trocou o byte invalido por U+FFFD: o config do usuario foi corrompido em silencio\n%s", bruto)
		return
	}
	if !bytes.Contains(bruto, []byte{0xff}) {
		t.Errorf("o byte 0xFF sumiu sem virar U+FFFD; o valor foi alterado de outro jeito\n%s", bruto)
		return
	}
	t.Logf("MEDIDO: Fundir ATRAVESSA byte nao-UTF-8 sem alterar")
}

// TestFundirChaveDuplicada mede o que a toolchain faz com chave repetida. O v1
// classico ficava com a ultima; o v2 recusa. As duas sao seguras aqui -- o que
// nao pode e Fundir aceitar o arquivo e perder uma das entradas de mcpServers.
func TestFundirChaveDuplicada(t *testing.T) {
	original := `{"x": 1, "x": 2, "mcpServers": {"outro": {"command": "node"}}}`
	caminho := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(caminho, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	err := hosts.Fundir(caminho, hosts.Entrada{Command: "gobsidian", Args: []string{"serve"}})
	bruto, errLer := os.ReadFile(caminho)
	if errLer != nil {
		t.Fatal(errLer)
	}
	if err != nil {
		if string(bruto) != original {
			t.Errorf("Fundir recusou e mesmo assim escreveu:\n%s", bruto)
		}
		t.Logf("MEDIDO: Fundir RECUSA chave duplicada (%v)", err)
		return
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(bruto, &doc); err != nil {
		t.Fatalf("Fundir gravou algo que nao se le de volta: %v", err)
	}
	var srv map[string]json.RawMessage
	if err := json.Unmarshal(doc["mcpServers"], &srv); err != nil {
		t.Fatal(err)
	}
	if _, ok := srv["outro"]; !ok {
		t.Errorf("o servidor que ja existia sumiu numa fusao que foi ACEITA")
	}
	t.Logf("MEDIDO: Fundir ACEITA chave duplicada e preserva mcpServers")
}

// TestFundirValorNaoObjetoEmMcpServersNaoDestroi cobre a outra porta de
// destruicao: mcpServers com forma inesperada. Fundir tem de recusar, e nao
// substituir por um objeto novo.
func TestFundirValorNaoObjetoEmMcpServers(t *testing.T) {
	original := `{"mcpServers": "isto nao e um objeto", "preferences": {"a": 1}}`
	caminho := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(caminho, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := hosts.Fundir(caminho, hosts.Entrada{Command: "gobsidian"}); err == nil {
		t.Error("Fundir aceitou mcpServers que nao e objeto")
	}
	bruto, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	if string(bruto) != original {
		t.Errorf("o arquivo foi alterado apesar da recusa:\n%s", bruto)
	}
}

func compactar(t *testing.T, v json.RawMessage) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, v); err != nil {
		// Nem tudo compacta (bytes invalidos); comparar cru ainda responde.
		return v
	}
	return buf.Bytes()
}
