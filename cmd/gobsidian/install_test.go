package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/instalar"
	"github.com/spf13/cobra"
)

// TestSemArgumentosSoAutoinstalaComAsDuasCondicoes cobre a decisao D-11 nos
// QUATRO cruzamentos.
//
// A condicao de terminal nao e zelo: um host MCP que invocasse o binario sem
// argumento dispararia uma instalacao no meio de uma sessao -- mexendo no PATH
// e na configuracao de hosts do usuario. Nenhum host faz isso hoje (todos
// passam `serve --vault`), e o custo de supor que nunca farao e alto demais
// para o beneficio de nao escrever esta guarda.
func TestSemArgumentosSoAutoinstalaComAsDuasCondicoes(t *testing.T) {
	casos := []struct {
		nome        string
		instalado   bool
		erro        error
		interativo  bool
		esperaAjuda bool
	}{
		{"instalado e interativo", true, nil, true, true},
		{"instalado e nao interativo", true, nil, false, true},
		{"nao instalado e NAO interativo", false, instalar.ErrSemManifesto, false, true},
		{"nao instalado e interativo -> instala", false, instalar.ErrSemManifesto, true, false},
		{"erro ao consultar cai na ajuda", false, errors.New("disco ilegivel"), true, true},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			origInst, origTerm, origRodar := estaInstaladoFn, terminalInterativoFn, rodarInstalacaoFn
			estaInstaladoFn = func() (bool, error) { return c.instalado, c.erro }
			terminalInterativoFn = func() bool { return c.interativo }

			// A instalacao REAL nunca e alcancada por este teste.
			//
			// Na primeira versao ela era, e o resultado foi concreto: o teste
			// leu o registro do Obsidian, escolheu um cofre, detectou seis
			// hosts de IA e reescreveu a configuracao dos seis para apontar
			// para o binario de TESTE, alem de mexer no PATH do usuario. O que
			// D-11 decide e SE instala; que a instalacao funciona e assunto de
			// internal/instalar.
			instalou := false
			rodarInstalacaoFn = func(context.Context, *cobra.Command, *opcoesDeInstalacao, string) error {
				instalou = true
				return nil
			}
			t.Cleanup(func() {
				estaInstaladoFn, terminalInterativoFn, rodarInstalacaoFn = origInst, origTerm, origRodar
			})

			root := newRootCmd()
			var saida bytes.Buffer
			root.SetOut(&saida)
			root.SetErr(&saida)
			root.SetIn(strings.NewReader("\n"))
			root.SetArgs(nil)

			_ = root.Execute()

			texto := saida.String()
			mostrouAjuda := strings.Contains(texto, "Usage") || strings.Contains(texto, "Uso") ||
				strings.Contains(texto, "gobsidian [command]") || strings.Contains(texto, "Available Commands")

			if c.esperaAjuda && !mostrouAjuda {
				t.Fatalf("esperava a ajuda, veio:\n%s", texto)
			}
			if !c.esperaAjuda && mostrouAjuda {
				t.Fatalf("mostrou a ajuda quando devia tentar instalar:\n%s", texto)
			}
			if c.esperaAjuda && instalou {
				t.Fatal("chamou a instalacao quando devia so mostrar a ajuda")
			}
			if !c.esperaAjuda {
				if !instalou {
					t.Fatalf("nao chamou a instalacao:\n%s", texto)
				}
				if !strings.Contains(texto, "ainda nao esta instalado") {
					t.Fatalf("nao anunciou a autoinstalacao antes de comecar:\n%s", texto)
				}
			}
		})
	}
}

// TestSubcomandosDoInstaladorEstaoRegistrados: um comando que existe no codigo
// e nao esta na arvore e codigo morto com aparencia de funcionalidade.
func TestSubcomandosDoInstaladorEstaoRegistrados(t *testing.T) {
	root := newRootCmd()
	registrados := map[string]bool{}
	for _, c := range root.Commands() {
		registrados[c.Name()] = true
	}
	for _, nome := range []string{"install", "update", "path", "vaults"} {
		if !registrados[nome] {
			t.Errorf("subcomando %q nao esta registrado na arvore", nome)
		}
	}
}

// TestPathExigeUmaDasDuasFlags: `--add` e `--remove` sao exclusivos, e nenhum
// dos dois e um no-op silencioso.
func TestPathExigeUmaDasDuasFlags(t *testing.T) {
	for _, args := range [][]string{{"path"}, {"path", "--add", "--remove"}} {
		root := newRootCmd()
		var saida bytes.Buffer
		root.SetOut(&saida)
		root.SetErr(&saida)
		root.SetArgs(args)
		if err := root.Execute(); err == nil {
			t.Errorf("`gobsidian %s` foi aceito; esperado erro", strings.Join(args, " "))
		}
	}
}
