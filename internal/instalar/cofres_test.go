package instalar

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// TestCofresDoObsidianDescartaOQueNaoExisteMais: o registro do Obsidian guarda
// cofres que o usuario ja apagou. Oferecer um deles faria a instalacao terminar
// apontando para o nada, e o erro so apareceria na primeira busca.
func TestCofresDoObsidianDescartaOQueNaoExisteMais(t *testing.T) {
	raiz := t.TempDir()
	vivo := filepath.Join(raiz, "cofre-vivo")
	aberto := filepath.Join(raiz, "cofre-aberto")
	for _, d := range []string{vivo, aberto} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	sumido := filepath.Join(raiz, "cofre-que-foi-apagado")

	registro := filepath.Join(raiz, "obsidian.json")
	conteudo := `{"vaults":{
		"a":{"path":` + aspas(sumido) + `,"open":false,"ts":1},
		"b":{"path":` + aspas(vivo) + `,"open":false,"ts":2},
		"c":{"path":` + aspas(aberto) + `,"open":true,"ts":3}
	}}`
	if err := os.WriteFile(registro, []byte(conteudo), 0o600); err != nil {
		t.Fatal(err)
	}

	cofres, err := CofresDoObsidian(registro)
	if err != nil {
		t.Fatalf("CofresDoObsidian() error = %v", err)
	}
	if len(cofres) != 2 {
		t.Fatalf("CofresDoObsidian() = %d cofres, esperado 2: %+v", len(cofres), cofres)
	}
	// O aberto vem primeiro: e o palpite mais provavel, e a ordem tem de ser
	// estavel porque o usuario escolhe por numero.
	if !cofres[0].Aberto || cofres[0].Caminho != aberto {
		t.Errorf("o cofre aberto nao veio primeiro: %+v", cofres)
	}
	for _, c := range cofres {
		if c.Caminho == sumido {
			t.Errorf("um cofre inexistente foi oferecido: %+v", cofres)
		}
	}
}

// TestCofresDoObsidianSemRegistroNaoEhErro: nem todo mundo tem Obsidian
// instalado, e isso nao pode abortar a instalacao do servidor.
func TestCofresDoObsidianSemRegistroNaoEhErro(t *testing.T) {
	cofres, err := CofresDoObsidian(filepath.Join(t.TempDir(), "nao-existe.json"))
	if err != nil {
		t.Fatalf("CofresDoObsidian() sem registro virou erro: %v", err)
	}
	if len(cofres) != 0 {
		t.Errorf("CofresDoObsidian() = %+v, esperado vazio", cofres)
	}
}

// TestCofresDoObsidianRegistroInvalidoEhErro: um registro ilegivel e diferente
// de um registro ausente, e a mensagem para o usuario tem de dizer qual dos
// dois aconteceu.
func TestCofresDoObsidianRegistroInvalidoEhErro(t *testing.T) {
	registro := filepath.Join(t.TempDir(), "obsidian.json")
	if err := os.WriteFile(registro, []byte("{isto nao e json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CofresDoObsidian(registro); err == nil {
		t.Fatal("CofresDoObsidian() aceitou um registro invalido")
	}
}

// aspas cita um caminho para dentro de um literal JSON.
//
// strconv.Quote, e nao escape a mao: as regras de escape de string sao as
// mesmas em Go e em JSON para o que aparece num caminho de arquivo, e escrever
// a segunda versao delas aqui seria uma conta a mais para errar.
func aspas(s string) string { return strconv.Quote(s) }
