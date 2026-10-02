package main

import (
	"strings"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/console"
)

// TestAjudaNomeiaOValorDaFlagPeloTipo: o pflag tira o nome do valor da
// PRIMEIRA palavra entre crases da descricao, e as crases aqui sao marcacao de
// codigo que o console desenha. Medido em 2026-10-01 no `graph --help` do dono:
// "--log-level debug", "--depth depth", "--include-broken include_broken" --
// um booleano com nome de valor, e um nome que nada tem a ver com o tipo.
func TestAjudaNomeiaOValorDaFlagPeloTipo(t *testing.T) {
	t.Setenv(console.VarDeEstilo, "0")
	out, errOut, err := rodar(t, "graph", "--help")
	if err != nil {
		t.Fatalf("%v\n%s", err, errOut)
	}
	linha := func(flag string) string {
		for _, l := range strings.Split(out, "\n") {
			if strings.Contains(l, flag+" ") {
				return l
			}
		}
		t.Fatalf("flag %s ausente da ajuda:\n%s", flag, out)
		return ""
	}
	for flag, quer := range map[string]string{
		"--depth":     "--depth int",
		"--log-level": "--log-level string",
		"--direction": "--direction string",
	} {
		if l := linha(flag); !strings.Contains(l, quer) {
			t.Errorf("ajuda de %s = %q, quer %q", flag, strings.TrimSpace(l), quer)
		}
	}
	// Booleano nao tem nome de valor.
	if l := linha("--include-broken"); strings.Contains(l, "--include-broken include_broken") {
		t.Errorf("booleano com nome de valor: %q", strings.TrimSpace(l))
	}
	// E a palavra que o pflag roubava continua na descricao.
	if l := linha("--log-level"); !strings.Contains(l, "debug") {
		t.Errorf("a descricao perdeu 'debug': %q", strings.TrimSpace(l))
	}
	// A --help do subcomando em portugues, e nao "help for graph" do cobra.
	if l := linha("--help"); !strings.Contains(l, "graph") || strings.Contains(l, "help for") {
		t.Errorf("--help do subcomando = %q", strings.TrimSpace(l))
	}
	// A descricao do parametro e a do schema, e nao o texto de reserva.
	if strings.Contains(out, "da tool (ver") {
		t.Errorf("parametro sem descricao na ajuda:\n%s", out)
	}
}
