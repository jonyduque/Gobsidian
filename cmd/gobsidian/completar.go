// completar.go liga o carapace a arvore de comandos.
//
// O cobra sozinho gera completion para bash, fish, powershell e zsh, e completa
// NOME de comando e de flag. O que faltava era duas coisas: o nushell, que ele
// nao conhece, e o VALOR de uma flag -- `--hosts <Tab>` nao oferecia nada, e a
// lista de hosts validos e uma constante deste binario.
package main

import (
	"github.com/carapace-sh/carapace"
	"github.com/jonyd/gobsidian/internal/hosts"
	"github.com/jonyd/gobsidian/internal/instalar"
	"github.com/spf13/cobra"
)

// instalarCompletion registra os completadores de VALOR e liga o carapace.
//
// # Por que os valores vem daqui, e nao de uma lista escrita no completion
//
// A lista de hosts, os niveis de log e os cofres do Obsidian ja existem no
// codigo. Escreve-los de novo num arquivo de completion seria a segunda copia
// de um fato -- e a copia menos consultada e a que fica errada, que e a
// armadilha registrada no CLAUDE.md. Aqui cada completador CHAMA a fonte:
// hosts.Chaves(), instalar.CofresDoObsidian().
//
// # O custo, medido antes de entrar
//
// Dois modulos novos (carapace e carapace-shlex) e +2,14 MB no binario, medido
// em 2026-09-09 contra um programa minimo com cobra. O carapace puxa `net`,
// `net/url` e `net/netip` transitivamente; nenhum pacote NOSSO os importa, que
// e o que a RNF-30 cobra -- a mesma situacao do SDK de MCP, registrada no
// CLAUDE.md.
func instalarCompletion(raiz *cobra.Command) {
	porNome := map[string]*cobra.Command{}
	for _, c := range raiz.Commands() {
		porNome[c.Name()] = c
	}

	// --hosts: as chaves reais, com o nome do host como descricao.
	//
	// `none` entra na lista porque ele e um valor VALIDO da flag -- quem quer
	// instalar sem tocar em host nenhum precisa descobrir isso, e o Tab e onde
	// se descobre.
	valoresDeHost := func() carapace.Action {
		pares := []string{"none", "nao configura nenhum host"}
		for _, h := range hosts.Todos() {
			pares = append(pares, h.Chave, h.Nome)
		}
		return carapace.ActionValuesDescribed(pares...)
	}

	// --vault: os cofres que o proprio Obsidian conhece, mais o caminho livre.
	//
	// ActionDirectories no fim, e nao no lugar: quem tem cofre fora do registro
	// do Obsidian continua completando caminho normalmente.
	valoresDeCofre := func() carapace.Action {
		return carapace.ActionCallback(func(carapace.Context) carapace.Action {
			cofres, err := instalar.CofresDoObsidian(instalar.CaminhoDoRegistroDoObsidian())
			if err != nil || len(cofres) == 0 {
				return carapace.ActionDirectories()
			}
			var pares []string
			for _, c := range cofres {
				nota := "cofre do Obsidian"
				if c.Aberto {
					nota = "aberto agora"
				}
				pares = append(pares, c.Caminho, nota)
			}
			return carapace.Batch(
				carapace.ActionValuesDescribed(pares...),
				carapace.ActionDirectories(),
			).ToA()
		})
	}

	niveisDeLog := func() carapace.Action {
		return carapace.ActionValuesDescribed(
			"debug", "tudo, inclusive o que so interessa depurando",
			"info", "o padrao",
			"warn", "so o que pede atencao",
			"error", "so falha",
		)
	}

	for nome, cmd := range porNome {
		flags := carapace.ActionMap{}
		if cmd.Flags().Lookup("vault") != nil {
			flags["vault"] = valoresDeCofre()
		}
		if cmd.Flags().Lookup("hosts") != nil {
			flags["hosts"] = valoresDeHost()
		}
		if cmd.Flags().Lookup("log-level") != nil {
			flags["log-level"] = niveisDeLog()
		}
		if cmd.Flags().Lookup("cache-dir") != nil {
			flags["cache-dir"] = carapace.ActionDirectories()
		}
		if cmd.Flags().Lookup("install-dir") != nil {
			flags["install-dir"] = carapace.ActionDirectories()
		}
		if len(flags) > 0 {
			carapace.Gen(cmd).FlagCompletion(flags)
		}
		_ = nome
	}

	carapace.Gen(raiz)
}
