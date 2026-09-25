package instalar

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/hosts"
)

// mundoDeCofres monta um obsidian.json de fixture e as pastas que ele cita.
// Devolve o caminho do registro e a raiz onde tudo mora.
func mundoDeCofres(t *testing.T, cofres ...string) (registro, raiz string) {
	t.Helper()
	raiz = t.TempDir()
	type entrada struct {
		Path string `json:"path"`
	}
	reg := struct {
		Vaults map[string]entrada `json:"vaults"`
	}{Vaults: map[string]entrada{}}
	for i, rel := range cofres {
		p := filepath.Join(raiz, filepath.FromSlash(rel))
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		reg.Vaults[string(rune('a'+i))] = entrada{Path: p}
	}
	b, err := json.Marshal(reg)
	if err != nil {
		t.Fatal(err)
	}
	registro = filepath.Join(raiz, "obsidian.json")
	if err := os.WriteFile(registro, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return registro, raiz
}

func pasta(t *testing.T, partes ...string) string {
	t.Helper()
	p := filepath.Join(partes...)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// Um caso por linha da regra de decisao da Parte J do plano.
func TestResolverCofre(t *testing.T) {
	t.Run("regra 1: valor com separador e caminho e nunca consulta o registro", func(t *testing.T) {
		reg, raiz := mundoDeCofres(t, "Cofres/Estudo")
		cwd := pasta(t, raiz, "cwd")
		for _, v := range []string{`.\Estudo`, "./Estudo", `Cofres\Estudo`, "Cofres/Estudo", filepath.Join(raiz, "Outro"), "~/Estudo", ".Estudo"} {
			got, porNome, err := ResolverCofre(v, reg, cwd)
			if err != nil || porNome || got != v {
				t.Errorf("ResolverCofre(%q) = %q, %v, %v; quer o valor intacto, sem nome", v, got, porNome, err)
			}
		}
	})

	t.Run("regra 2: nome simples, caixa diferente e NFD casam a pasta do registro", func(t *testing.T) {
		reg, raiz := mundoDeCofres(t, "Cofres/Estudo", "Outros/Revisão")
		cwd := pasta(t, raiz, "cwd")
		casos := map[string]string{
			"Estudo":   filepath.Join(raiz, "Cofres", "Estudo"),
			"estudo":   filepath.Join(raiz, "Cofres", "Estudo"),
			"ESTUDO":   filepath.Join(raiz, "Cofres", "Estudo"),
			"Revisão": filepath.Join(raiz, "Outros", "Revisão"),
			"revisão":  filepath.Join(raiz, "Outros", "Revisão"),
		}
		for v, quer := range casos {
			got, porNome, err := ResolverCofre(v, reg, cwd)
			if err != nil || !porNome || got != quer {
				t.Errorf("ResolverCofre(%q) = %q, %v, %v; quer %q por nome", v, got, porNome, err, quer)
			}
		}
	})

	t.Run("regra 2: nome sem acento nao casa pasta acentuada", func(t *testing.T) {
		reg, raiz := mundoDeCofres(t, "Outros/Revisão")
		cwd := pasta(t, raiz, "cwd")
		_, _, err := ResolverCofre("Revisao", reg, cwd)
		var e *ErroDeCofre
		if !errors.As(err, &e) || len(e.Conhecidos) != 1 || e.Conhecidos[0] != "Revisão" {
			t.Errorf("err = %v, quer ErroDeCofre com o nome conhecido Revisão", err)
		}
	})

	t.Run("regra 3: nome do registro e pasta homonima no cwd em lugares diferentes e erro", func(t *testing.T) {
		reg, raiz := mundoDeCofres(t, "Cofres/Estudo")
		cwd := pasta(t, raiz, "cwd")
		pasta(t, cwd, "Estudo")
		_, _, err := ResolverCofre("Estudo", reg, cwd)
		var e *ErroDeCofre
		if !errors.As(err, &e) || e.NoDiretorio != filepath.Join(cwd, "Estudo") || len(e.Candidatos) != 1 {
			t.Errorf("err = %v, quer ErroDeCofre com a pasta do cwd e o cofre do registro", err)
		}
	})

	t.Run("regra 3: pasta do cwd que E o cofre do registro nao e conflito", func(t *testing.T) {
		reg, raiz := mundoDeCofres(t, "Cofres/Estudo")
		cwd := filepath.Join(raiz, "Cofres")
		got, porNome, err := ResolverCofre("Estudo", reg, cwd)
		if err != nil || !porNome || got != filepath.Join(raiz, "Cofres", "Estudo") {
			t.Errorf("= %q, %v, %v", got, porNome, err)
		}
	})

	t.Run("regra 4: dois cofres com o mesmo nome e erro com os dois caminhos", func(t *testing.T) {
		reg, raiz := mundoDeCofres(t, "A/Estudo", "B/Estudo")
		cwd := pasta(t, raiz, "cwd")
		_, _, err := ResolverCofre("Estudo", reg, cwd)
		var e *ErroDeCofre
		quer := []string{filepath.Join(raiz, "A", "Estudo"), filepath.Join(raiz, "B", "Estudo")}
		if !errors.As(err, &e) || !slices.Equal(e.Candidatos, quer) {
			t.Errorf("err = %v, quer ErroDeCofre com %v", err, quer)
		}
	})

	t.Run("regra 5: nome sem cofre e pasta no cwd continua sendo caminho", func(t *testing.T) {
		reg, raiz := mundoDeCofres(t, "Cofres/Estudo")
		cwd := pasta(t, raiz, "cwd")
		pasta(t, cwd, "Local")
		got, porNome, err := ResolverCofre("Local", reg, cwd)
		if err != nil || porNome || got != "Local" {
			t.Errorf("= %q, %v, %v; quer o valor intacto", got, porNome, err)
		}
	})

	t.Run("regra 6: nome que nao casa nada lista os conhecidos", func(t *testing.T) {
		reg, raiz := mundoDeCofres(t, "Cofres/Estudo", "Cofres/Oral")
		cwd := pasta(t, raiz, "cwd")
		_, _, err := ResolverCofre("Nada", reg, cwd)
		var e *ErroDeCofre
		if !errors.As(err, &e) || !slices.Equal(e.Conhecidos, []string{"Estudo", "Oral"}) || e.SemRegistro {
			t.Errorf("err = %v, quer ErroDeCofre com Estudo e Oral", err)
		}
	})

	t.Run("regra 6: registro ausente diz isso", func(t *testing.T) {
		raiz := t.TempDir()
		_, _, err := ResolverCofre("Estudo", filepath.Join(raiz, "obsidian.json"), raiz)
		var e *ErroDeCofre
		if !errors.As(err, &e) || !e.SemRegistro {
			t.Errorf("err = %v, quer ErroDeCofre com SemRegistro", err)
		}
	})

	t.Run("regra 6: cofre do registro cujo diretorio sumiu nao casa", func(t *testing.T) {
		reg, raiz := mundoDeCofres(t, "Cofres/Sumido", "Cofres/Estudo")
		if err := os.Remove(filepath.Join(raiz, "Cofres", "Sumido")); err != nil {
			t.Fatal(err)
		}
		cwd := pasta(t, raiz, "cwd")
		_, _, err := ResolverCofre("Sumido", reg, cwd)
		var e *ErroDeCofre
		if !errors.As(err, &e) || !slices.Equal(e.Conhecidos, []string{"Estudo"}) {
			t.Errorf("err = %v, quer ErroDeCofre so com Estudo", err)
		}
	})

	t.Run("cwd vazio nao consulta diretorio corrente nenhum", func(t *testing.T) {
		reg, _ := mundoDeCofres(t, "Cofres/Estudo")
		if _, _, err := ResolverCofre("Nada", reg, ""); err == nil {
			t.Error("nome desconhecido com cwd vazio nao deu erro")
		}
	})

	t.Run("vazio passa intacto para config.Load recusar", func(t *testing.T) {
		got, porNome, err := ResolverCofre("", "x", "y")
		if err != nil || porNome || got != "" {
			t.Errorf("= %q, %v, %v", got, porNome, err)
		}
	})
}

// J4.1: entrada escrita a mao com o nome do cofre resolve pelo registro; nome
// que nao resolve aparece como veio, em vez de virar <cwd>\Nome ou sumir.
func TestConfiguracaoAtualResolveEntradaPeloNome(t *testing.T) {
	reg, raiz := mundoDeCofres(t, "Cofres/Estudo")
	caminhoDoCofre := filepath.Join(raiz, "Cofres", "Estudo")
	outro := filepath.Join(raiz, "Outro")

	home := filepath.Join(raiz, "home")
	config := filepath.Join(home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := hosts.FundirVarias(config, []hosts.EntradaNomeada{
		{Chave: "gobsidian-estudo", Entrada: hosts.Entrada{Command: "g", Args: []string{"serve", "--vault", "Estudo"}}},
		{Chave: "gobsidian-sumido", Entrada: hosts.Entrada{Command: "g", Args: []string{"serve", "--vault", "Sumido"}}},
		{Chave: "gobsidian-outro", Entrada: hosts.Entrada{Command: "g", Args: []string{"serve", "--vault", outro}}},
	}); err != nil {
		t.Fatal(err)
	}
	amb := hosts.Ambiente{
		Home:       home,
		Existe:     func(string) bool { return false },
		TemComando: func(nome string) bool { return nome == "cursor" },
		Rodar:      func(string, ...string) error { return nil },
	}

	got := ConfiguracaoAtual(amb, reg)
	quer := []string{caminhoDoCofre, outro, "Sumido"}
	slices.Sort(quer)
	if !slices.Equal(got, quer) {
		t.Errorf("ConfiguracaoAtual = %q\nquer                %q", got, quer)
	}
}
