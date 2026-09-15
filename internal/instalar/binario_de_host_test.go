package instalar

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/hosts"
)

// executavelFalso cria um arquivo que exec.LookPath aceita como executavel nas
// tres plataformas: extensao .exe para o Windows, bit de execucao para as
// outras.
func executavelFalso(t *testing.T, caminho string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caminho, []byte("binario"), 0o755); err != nil {
		t.Fatal(err)
	}
	return caminho
}

// TestEntradasDeOutroBinarioApontaSoAsQueNaoRodamOInstalado e G7.1, na forma
// medida em 2026-09-14: um host rodando o binario instalado, outro rodando um
// binario de C:\Program Files. O Cursor e o host da fixture porque o config
// dele fica sob o home nas tres plataformas.
func TestEntradasDeOutroBinarioApontaSoAsQueNaoRodamOInstalado(t *testing.T) {
	raiz := t.TempDir()
	instalado := executavelFalso(t, filepath.Join(raiz, "instalado", "gobsidian.exe"))
	antigo := executavelFalso(t, filepath.Join(raiz, "Program Files", "gobsidian", "gobsidian.exe"))
	sumido := filepath.Join(raiz, "removido", "gobsidian.exe")

	home := filepath.Join(raiz, "home")
	config := filepath.Join(home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	// Um servidor de outro produto, com o MESMO executavel antigo: nao e nosso,
	// e nao entra na conta.
	outro := `{"mcpServers":{"outro":{"command":` + jsonString(antigo) + `,"args":[]}}}`
	if err := os.WriteFile(config, []byte(outro), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := hosts.FundirVarias(config, []hosts.EntradaNomeada{
		{Chave: "gobsidian-estudo", Entrada: hosts.Entrada{Command: instalado, Args: []string{"serve", "--vault", "E"}}},
		{Chave: "gobsidian-oral", Entrada: hosts.Entrada{Command: antigo, Args: []string{"serve", "--vault", "O"}}},
		{Chave: "gobsidian-revisao", Entrada: hosts.Entrada{Command: sumido, Args: []string{"serve", "--vault", "R"}}},
	}); err != nil {
		t.Fatal(err)
	}

	amb := hosts.Ambiente{
		Home:       home,
		Existe:     func(string) bool { return false },
		TemComando: func(nome string) bool { return nome == "cursor" },
		Rodar:      func(string, ...string) error { return nil },
	}

	got := EntradasDeOutroBinario(amb, instalado)
	chaves := make([]string, 0, len(got))
	for _, e := range got {
		chaves = append(chaves, e.Chave)
	}
	if !slices.Equal(chaves, []string{"gobsidian-oral", "gobsidian-revisao"}) {
		t.Fatalf("entradas de outro binario = %+v, esperado oral (binario antigo) e revisao (comando sumido)", got)
	}
	if got[0].Host != "Cursor" || !mesmoExecutavel(got[0].Executavel, antigo) {
		t.Errorf("oral = %+v, esperado host Cursor e executavel %s", got[0], antigo)
	}
	if got[1].Executavel != "" || got[1].Comando != sumido {
		t.Errorf("revisao = %+v, esperado comando %s sem executavel resolvido", got[1], sumido)
	}
}

// TestVersaoDoExecutavelVemSoDaPresencaDeUmProcessoDele: a versao sai de um
// processo vivo daquele executavel que registrou presenca. Processo sem
// presenca -- o v1.5.1 medido -- nao da versao, e o doctor diz que nao mediu.
func TestVersaoDoExecutavelVemSoDaPresencaDeUmProcessoDele(t *testing.T) {
	raiz := t.TempDir()
	novo := executavelFalso(t, filepath.Join(raiz, "novo", "gobsidian.exe"))
	antigo := executavelFalso(t, filepath.Join(raiz, "antigo", "gobsidian.exe"))
	terceiro := executavelFalso(t, filepath.Join(raiz, "terceiro", "gobsidian.exe"))

	vivos := []Presenca{{PID: 10, Versao: "v1.8.1"}, {PID: 20, Versao: "v1.9.0"}}
	processos := []ProcessoDoSistema{
		{PID: 20, Executavel: terceiro},
		{PID: 10, Executavel: novo},
		{PID: 30, Executavel: antigo},
	}

	if v := VersaoDoExecutavel(novo, vivos, processos); v != "v1.8.1" {
		t.Errorf("versao do novo = %q, esperado v1.8.1 (pid 10)", v)
	}
	if v := VersaoDoExecutavel(antigo, vivos, processos); v != "" {
		t.Errorf("versao do antigo = %q, esperado vazio: o pid 30 nao tem presenca", v)
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
