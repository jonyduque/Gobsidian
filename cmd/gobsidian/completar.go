// completar.go liga o carapace a arvore de comandos.
//
// O cobra sozinho gera completion para bash, fish, powershell e zsh, e completa
// NOME de comando e de flag. O que faltava era duas coisas: o nushell, que ele
// nao conhece, e o VALOR de uma flag -- `--hosts <Tab>` nao oferecia nada, e a
// lista de hosts validos e uma constante deste binario.
package main

import (
	"github.com/carapace-sh/carapace"
	"github.com/jonyduque/Gobsidian/internal/hosts"
	"github.com/jonyduque/Gobsidian/internal/instalar"
	"github.com/jonyduque/Gobsidian/internal/textos"
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
	// Todos os comandos da arvore, e nao so os do primeiro nivel: os das
	// tools moram em grupos (`note read`), e --vault neles completa igual.
	var todos []*cobra.Command
	var andar func(c *cobra.Command)
	andar = func(c *cobra.Command) {
		for _, f := range c.Commands() {
			todos = append(todos, f)
			andar(f)
		}
	}
	andar(raiz)

	// --hosts: as chaves reais, com o nome do host como descricao.
	//
	// `none` entra na lista porque ele e um valor VALIDO da flag -- quem quer
	// instalar sem tocar em host nenhum precisa descobrir isso, e o Tab e onde
	// se descobre.
	valoresDeHost := func() carapace.Action {
		pares := []string{"none", textos.CompletarNenhumHost}
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
			return carapace.Batch(
				carapace.ActionValuesDescribed(paresDeCofre(cofres)...),
				carapace.ActionDirectories(),
			).ToA()
		})
	}

	niveisDeLog := func() carapace.Action {
		return carapace.ActionValuesDescribed(
			"debug", textos.CompletarLogDebug,
			"info", textos.CompletarLogInfo,
			"warn", textos.CompletarLogWarn,
			"error", textos.CompletarLogError,
		)
	}

	for _, cmd := range todos {
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
	}

	carapace.Gen(raiz)
}

// paresDeCofre monta os pares valor/descricao de --vault: primeiro o NOME de
// cada cofre, descrito pelo caminho, e depois os caminhos.
//
// Nome que o registro repete nao e oferecido como nome, so pelos caminhos:
// --vault recusaria o nome ambiguo, e oferecer no Tab o que a flag recusa e
// ensinar o erro. Quem decide o que e ambiguo e instalar.CofresPorNome, a
// mesma conta de instalar.ResolverCofre.
func paresDeCofre(cofres []instalar.Cofre) []string {
	var pares []string
	for _, g := range instalar.CofresPorNome(cofres) {
		if len(g.Caminhos) == 1 {
			pares = append(pares, g.Nome, g.Caminhos[0])
		}
	}
	for _, c := range cofres {
		nota := textos.CompletarCofre
		if c.Aberto {
			nota = textos.CompletarCofreAberto
		}
		pares = append(pares, c.Caminho, nota)
	}
	return pares
}
