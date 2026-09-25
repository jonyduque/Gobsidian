package main

import (
	"bufio"
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/console"
)

// Medido pelo dono em 2026-09-25, com `gobsidian vaults`: respondeu Sim a
// "Manter esta configuracao?" e a lista de hosts apareceu mesmo assim. O
// "manter" so valia para os cofres; escolherHosts rodava em seguida de
// qualquer jeito, e com --yes a fatia nula de hosts configurava TODOS os
// detectados. Manter e nao alterar nada.
func TestManterConfiguracaoNaoPerguntaNemAlteraHosts(t *testing.T) {
	antesAtual, antesHosts, antesTerminal := configuracaoAtualFn, escolherHostsFn, terminalInterativoFn
	t.Cleanup(func() {
		configuracaoAtualFn, escolherHostsFn, terminalInterativoFn = antesAtual, antesHosts, antesTerminal
	})
	configuracaoAtualFn = func() []string { return []string{`C:\Cofres\Estudo`} }
	terminalInterativoFn = func() bool { return false }
	perguntouHosts := false
	escolherHostsFn = func(*console.Stream, *bufio.Reader, *opcoesDeInstalacao) ([]string, error) {
		perguntouHosts = true
		return nil, nil
	}

	for _, c := range []struct {
		nome    string
		o       opcoesDeInstalacao
		entrada string
	}{
		{"respondeu sim", opcoesDeInstalacao{}, "s\n"},
		{"enter no padrao", opcoesDeInstalacao{}, "\n"},
		{"com --yes", opcoesDeInstalacao{sim: true}, ""},
	} {
		t.Run(c.nome, func(t *testing.T) {
			perguntouHosts = false
			var saida bytes.Buffer
			e, err := escolherConfiguracao(console.NewPlain(&saida), bufio.NewReader(strings.NewReader(c.entrada)), &c.o)
			if err != nil {
				t.Fatalf("escolherConfiguracao: %v", err)
			}
			if perguntouHosts {
				t.Error("pediu os hosts depois de o usuario manter a configuracao")
			}
			if !e.manter || e.hosts == nil || len(e.hosts) != 0 {
				t.Errorf("escolha = %+v; quer manter, com hosts vazio e NAO nulo (nulo configura todos os detectados)", e)
			}
			if !slices.Equal(e.cofres, []string{`C:\Cofres\Estudo`}) {
				t.Errorf("cofres = %q", e.cofres)
			}
		})
	}

	t.Run("respondeu nao pergunta os hosts", func(t *testing.T) {
		perguntouHosts = false
		o := opcoesDeInstalacao{vault: ""}
		var saida bytes.Buffer
		// "n" recusa manter; a lista de cofres, sem terminal, e digitada: "1".
		_, _ = escolherConfiguracao(console.NewPlain(&saida), bufio.NewReader(strings.NewReader("n\n1\n")), &o)
		if !perguntouHosts {
			t.Errorf("recusou manter e os hosts nao foram pedidos; saida:\n%s", saida.String())
		}
	})

	t.Run("--hosts explicito nao oferece manter", func(t *testing.T) {
		perguntouHosts = false
		o := opcoesDeInstalacao{hostsCSV: "claude-desktop", sim: true}
		var saida bytes.Buffer
		e, err := escolherConfiguracao(console.NewPlain(&saida), bufio.NewReader(strings.NewReader("")), &o)
		if err != nil {
			t.Fatalf("escolherConfiguracao: %v", err)
		}
		if e.manter || !perguntouHosts {
			t.Errorf("com --hosts: manter=%v, hosts pedidos=%v; quer a configuracao pedida na linha de comando", e.manter, perguntouHosts)
		}
	})
}
