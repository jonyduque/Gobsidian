package main

import (
	"strings"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/console"
	"github.com/spf13/pflag"
)

// TestLinhaDeFalhaComecaComMaiuscula: o erro de um pacote-folha (config,
// selfupdate) chega cru, em minuscula, e a frase que abre a linha e decidida
// na hora de imprimir.
func TestLinhaDeFalhaComecaComMaiuscula(t *testing.T) {
	casos := map[string]string{
		"caminho do cofre não informado": "Caminho do cofre não informado",
		"ação direta":                    "Ação direta",
		"`--idle-seconds` precisa":       "`--idle-seconds` precisa",
		"":                               "",
	}
	for entra, quer := range casos {
		if got := comMaiuscula(entra); got != quer {
			t.Errorf("comMaiuscula(%q) = %q, quer %q", entra, got, quer)
		}
	}
}

// TestAutocompletarRecebeTextoSemMarcacao: o shell mostra o resumo e a
// descricao de flag direto, sem console; "**gobsidian**" apareceria com os
// asteriscos.
func TestAutocompletarRecebeTextoSemMarcacao(t *testing.T) {
	root := newRootCmd()
	console.TirarMarcacaoDaArvore(root)

	var sujos []string
	for _, c := range root.Commands() {
		if strings.Contains(c.Short, "*") {
			sujos = append(sujos, c.Name()+": "+c.Short)
		}
		c.Flags().VisitAll(func(f *pflag.Flag) {
			if strings.Contains(f.Usage, "**") || strings.Contains(f.Usage, "*PATH*") {
				sujos = append(sujos, c.Name()+" --"+f.Name+": "+f.Usage)
			}
		})
	}
	if len(sujos) > 0 {
		t.Errorf("sobrou marcacao no que o shell mostra:\n%s", strings.Join(sujos, "\n"))
	}
}
