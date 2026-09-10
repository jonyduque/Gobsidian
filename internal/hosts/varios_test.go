package hosts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/hosts"
)

func lerServidores(t *testing.T, caminho string) map[string]json.RawMessage {
	t.Helper()
	bruto, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(bruto, &doc); err != nil {
		t.Fatalf("config nao volta como JSON: %v\n%s", err, bruto)
	}
	var srv map[string]json.RawMessage
	if len(doc["mcpServers"]) > 0 {
		if err := json.Unmarshal(doc["mcpServers"], &srv); err != nil {
			t.Fatalf("mcpServers nao e objeto: %v", err)
		}
	}
	return srv
}

func entradaDe(binario, cofre string) hosts.Entrada {
	return hosts.Entrada{Command: binario, Args: []string{"serve", "--vault", cofre}}
}

// Um cofre continua saindo sob "gobsidian". E a promessa de compatibilidade:
// quem ja usa o produto nao tem a chave do config trocada de graca.
func TestFundirVariasUmCofreMantemAChaveAntiga(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "c.json")
	err := hosts.FundirVarias(caminho, []hosts.EntradaNomeada{
		{Chave: hosts.ChaveDoServidor, Entrada: entradaDe("gob", "/a")},
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := lerServidores(t, caminho)
	if _, ok := srv[hosts.ChaveDoServidor]; !ok {
		t.Errorf("a chave %q nao esta no config: %v", hosts.ChaveDoServidor, srv)
	}
	if len(srv) != 1 {
		t.Errorf("config tem %d servidores, queria 1: %v", len(srv), srv)
	}
}

func TestFundirVariasEscreveUmaEntradaPorCofre(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "c.json")
	entradas := []hosts.EntradaNomeada{
		{Chave: "gobsidian-estudo", Entrada: entradaDe("gob", "/x/Estudo")},
		{Chave: "gobsidian-trabalho", Entrada: entradaDe("gob", "/x/Trabalho")},
	}
	if err := hosts.FundirVarias(caminho, entradas); err != nil {
		t.Fatal(err)
	}
	srv := lerServidores(t, caminho)
	for _, en := range entradas {
		if _, ok := srv[en.Chave]; !ok {
			t.Errorf("faltou a entrada %q: %v", en.Chave, srv)
		}
	}
	if len(srv) != 2 {
		t.Errorf("config tem %d servidores, queria 2: %v", len(srv), srv)
	}
}

// A poda e o que faz a escolha do usuario valer: quem tinha tres cofres e
// escolheu um termina com UM, e nao com um novo mais dois mortos.
func TestFundirVariasPodaAsNossasQueSobraram(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "c.json")
	if err := hosts.FundirVarias(caminho, []hosts.EntradaNomeada{
		{Chave: "gobsidian-a", Entrada: entradaDe("gob", "/x/A")},
		{Chave: "gobsidian-b", Entrada: entradaDe("gob", "/x/B")},
	}); err != nil {
		t.Fatal(err)
	}

	if err := hosts.FundirVarias(caminho, []hosts.EntradaNomeada{
		{Chave: hosts.ChaveDoServidor, Entrada: entradaDe("gob", "/x/A")},
	}); err != nil {
		t.Fatal(err)
	}

	srv := lerServidores(t, caminho)
	if len(srv) != 1 {
		t.Errorf("sobrou entrada morta: %v", srv)
	}
	if _, ok := srv[hosts.ChaveDoServidor]; !ok {
		t.Errorf("a entrada nova nao esta la: %v", srv)
	}
}

// Nenhum cofre escolhido tira as NOSSAS entradas e nao toca em mais nada.
func TestFundirVariasSemEntradasSoTiraAsNossas(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "c.json")
	original := `{"preferences":{"theme":"dark"},"mcpServers":{"gobsidian":{"command":"gob"},"outro":{"command":"node"}}}`
	if err := os.WriteFile(caminho, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := hosts.FundirVarias(caminho, nil); err != nil {
		t.Fatal(err)
	}

	srv := lerServidores(t, caminho)
	if _, ok := srv[hosts.ChaveDoServidor]; ok {
		t.Errorf("a nossa entrada continua la: %v", srv)
	}
	if _, ok := srv["outro"]; !ok {
		t.Errorf("o servidor de outro produto foi removido: %v", srv)
	}
	bruto, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(bruto, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["preferences"]; !ok {
		t.Errorf("as preferencias do usuario sumiram: %s", bruto)
	}
}

func TestLerEntradasDevolveSoAsNossas(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "c.json")
	original := `{"mcpServers":{
		"gobsidian-estudo":{"command":"gob","args":["serve","--vault","/x/Estudo"]},
		"outro":{"command":"node","args":["serve","--vault","/nao/e/nosso"]}
	}}`
	if err := os.WriteFile(caminho, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	ens, err := hosts.LerEntradas(caminho)
	if err != nil {
		t.Fatal(err)
	}
	if len(ens) != 1 {
		t.Fatalf("LerEntradas = %+v, queria so a nossa", ens)
	}
	if got := hosts.CofreDaEntrada(ens[0].Entrada); got != "/x/Estudo" {
		t.Errorf("CofreDaEntrada = %q, queria /x/Estudo", got)
	}
}

func TestLerEntradasArquivoAusenteNaoEhErro(t *testing.T) {
	ens, err := hosts.LerEntradas(filepath.Join(t.TempDir(), "nao-existe.json"))
	if err != nil {
		t.Fatalf("arquivo ausente virou erro: %v", err)
	}
	if len(ens) != 0 {
		t.Errorf("LerEntradas = %+v, queria nada", ens)
	}
}

func TestEhNossaNaoCasaServidorDeOutroProduto(t *testing.T) {
	casos := map[string]bool{
		"gobsidian":        true,
		"gobsidian-estudo": true,
		"outro":            false,
		"gobsidiano":       false, // prefixo sem hifen NAO e nosso
		"meu-gobsidian":    false,
		"gobsidian-":       true,
	}
	for chave, quer := range casos {
		if got := hosts.EhNossa(chave); got != quer {
			t.Errorf("EhNossa(%q) = %v, queria %v", chave, got, quer)
		}
	}
}
