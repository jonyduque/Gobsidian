// completion_extra.go poe no `completion` os shells que o cobra nao gera.
package main

import (
	"fmt"

	"github.com/carapace-sh/carapace"
	"github.com/spf13/cobra"
)

// shellsDoCarapace sao os que o cobra NAO cobre, com a instrucao de carga de
// cada um.
//
// O cobra gera bash, fish, powershell e zsh, e esses quatro ja recebem os
// valores registrados em completar.go -- eles passam pelo protocolo
// `__complete`, que e o mesmo que o carapace alimenta. Conferido em
// 2026-09-10: `gobsidian __complete serve --vault ""` devolve os cofres do
// Obsidian. Duplicar aqui os quatro seria oferecer duas fontes para o mesmo
// shell.
//
// O que faltava eram estes, e o nushell em especial: sem entrada no
// `completion` ele so existia atras do comando OCULTO `_carapace`, que ninguem
// descobre. Foi assim que o dono reportou a falta.
var shellsDoCarapace = []struct {
	nome         string
	comoCarregar string
}{
	{"nushell", "Acrescente ao seu config.nu:\n\n" +
		"  let gobsidian_completer = {|spans| gobsidian _carapace nushell ...$spans | from json }\n" +
		"  $env.config.completions.external = { enable: true, completer: $gobsidian_completer }"},
	{"elvish", "Acrescente ao seu rc.elv:\n\n  eval (gobsidian completion elvish | slurp)"},
	{"ion", "Acrescente ao seu initrc:\n\n  eval $(gobsidian completion ion)"},
	{"oil", "Acrescente ao seu oshrc:\n\n  source <(gobsidian completion oil)"},
	{"tcsh", "Acrescente ao seu .tcshrc:\n\n  eval `gobsidian completion tcsh`"},
	{"xonsh", "Acrescente ao seu .xonshrc:\n\n  exec($(gobsidian completion xonsh))"},
	{"cmd_clink", "Salve a saida em um arquivo .lua dentro do diretorio de scripts do clink."},
	{"bash_ble", "Acrescente ao seu .bashrc, DEPOIS de carregar o ble.sh:\n\n  source <(gobsidian completion bash_ble)"},
}

// acrescentarShellsDoCarapace pendura um subcomando por shell sob `completion`.
//
// Sem isto, `gobsidian completion` listava quatro shells e o nushell nao
// aparecia em lugar nenhum -- o unico caminho era adivinhar o nome do comando
// oculto. Um recurso que existe e nao se descobre nao existe para quem procura.
func acrescentarShellsDoCarapace(raiz *cobra.Command) {
	var completion *cobra.Command
	for _, c := range raiz.Commands() {
		if c.Name() == "completion" {
			completion = c
			break
		}
	}
	if completion == nil {
		// O cobra so cria `completion` quando ha subcomandos e ele nao foi
		// desabilitado. Nao ter onde pendurar nao e motivo para derrubar o
		// binario: quem perde e a completacao, e o resto continua servindo.
		return
	}

	for _, s := range shellsDoCarapace {
		cmd := &cobra.Command{
			Use:   s.nome,
			Short: "Generate the autocompletion script for " + s.nome,
			Long:  "Gera o script de autocompletar para " + s.nome + ".\n\n" + s.comoCarregar,
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				// Snippet le a arvore inteira, entao o alvo e a RAIZ, e nao o
				// subcomando que esta imprimindo.
				texto, err := carapace.Gen(raiz).Snippet(cmd.Name())
				if err != nil {
					return fmt.Errorf("gerando o script de %s: %w", cmd.Name(), err)
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), texto)
				return err
			},
		}
		completion.AddCommand(cmd)
	}
}
