package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/instalar"
)

// TestListarCofresUnePelaGrafiaCanonica: o registro do Obsidian e o config de
// um host guardam o mesmo cofre com grafias diferentes, e a lista precisa
// mostrar UM cofre -- em 2026-09-11 o dono viu cada cofre duas vezes na lista
// do install por esse motivo.
func TestListarCofresUnePelaGrafiaCanonica(t *testing.T) {
	raiz := t.TempDir()
	estudo := filepath.Join(raiz, "Estudo")
	fora := filepath.Join(raiz, "Fora")

	lista := listarCofres(
		[]instalar.Cofre{{Caminho: estudo, Aberto: true}, {Caminho: filepath.Join(raiz, "Oral")}},
		[]string{filepath.ToSlash(estudo) + "/", fora},
	)

	if len(lista) != 3 {
		t.Fatalf("lista = %+v, quer 3 cofres (Estudo uma vez so)", lista)
	}
	porNome := map[string]cofreListado{}
	for _, c := range lista {
		porNome[c.Nome] = c
	}
	if c := porNome["Estudo"]; !c.Aberto || !c.Configurado || !c.NoObsidian {
		t.Errorf("Estudo = %+v, quer aberto, configurado e no Obsidian", c)
	}
	if c := porNome["Oral"]; c.Configurado || !c.NoObsidian {
		t.Errorf("Oral = %+v, quer no Obsidian e nao configurado", c)
	}
	// Configurado num host e desconhecido do Obsidian: some da lista seria
	// esconder exatamente o caso que se quer ver.
	if c, ok := porNome["Fora"]; !ok || !c.Configurado || c.NoObsidian {
		t.Errorf("Fora = %+v (presente=%v), quer configurado e fora do Obsidian", c, ok)
	}
}

// TestVaultsForaDoTerminalSaiEmJSON: um buffer nao e terminal, e o que sai e a
// lista JSON numa linha -- o que um script passa adiante.
func TestVaultsForaDoTerminalSaiEmJSON(t *testing.T) {
	raiz := t.TempDir()
	cofre := filepath.Join(raiz, "Estudo")
	antesObs, antesCfg := cofresDoObsidianFn, configuracaoAtualFn
	cofresDoObsidianFn = func() ([]instalar.Cofre, error) { return []instalar.Cofre{{Caminho: cofre}}, nil }
	configuracaoAtualFn = func() []string { return nil }
	defer func() { cofresDoObsidianFn, configuracaoAtualFn = antesObs, antesCfg }()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"vaults"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	linhas := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(linhas) != 1 {
		t.Fatalf("esperava uma linha de JSON, saiu:\n%s", out.String())
	}
	var lista []cofreListado
	if err := json.Unmarshal([]byte(linhas[0]), &lista); err != nil {
		t.Fatalf("a saida nao e JSON: %v\n%s", err, out.String())
	}
	if len(lista) != 1 || lista[0].Nome != "Estudo" || !lista[0].NoObsidian {
		t.Errorf("lista = %+v", lista)
	}
}

// TestVaultsComTextoMostraATabela: --texto forca a forma de ler mesmo num
// buffer.
func TestVaultsComTextoMostraATabela(t *testing.T) {
	raiz := t.TempDir()
	antesObs, antesCfg := cofresDoObsidianFn, configuracaoAtualFn
	cofresDoObsidianFn = func() ([]instalar.Cofre, error) {
		return []instalar.Cofre{{Caminho: filepath.Join(raiz, "Estudo")}}, nil
	}
	configuracaoAtualFn = func() []string { return []string{filepath.Join(raiz, "Fora")} }
	defer func() { cofresDoObsidianFn, configuracaoAtualFn = antesObs, antesCfg }()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"vaults", "--texto"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	saida := out.String()
	for _, quer := range []string{"Estudo", "Fora", "fora do Obsidian", "configurado", "gobsidian config"} {
		if !strings.Contains(saida, quer) {
			t.Errorf("faltou %q:\n%s", quer, saida)
		}
	}
}

// TestJSONETextoJuntosSaoUsoErrado: os dois forcam formas opostas.
func TestJSONETextoJuntosSaoUsoErrado(t *testing.T) {
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"vaults", "--json", "--texto"})
	err := root.Execute()
	if codigo, _ := codigoDeSaida(err); err == nil || codigo != saidaUso {
		t.Errorf("err = %v, codigo = %d; quer erro de uso (%d)", err, codigo, saidaUso)
	}
}
