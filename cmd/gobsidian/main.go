// Package main e o binario do gobsidian: `serve` levanta o servidor MCP
// sobre stdio, `doctor` diagnostica o ambiente e `version` imprime a
// identificacao do build.
//
// A divisao entre quem escreve em stdout e quem nao escreve vive aqui e e
// load-bearing: `serve` cede stdout inteiro ao JSON-RPC, enquanto `doctor` e
// `version` sao comandos de CLI e imprimem la de proposito.
package main

import (
	"fmt"
	"os"

	"github.com/jonyd/gobsidian/internal/console"
	"github.com/jonyd/gobsidian/internal/mcpsrv"
	"github.com/spf13/cobra"
)

// Injetados pelo linker. O script de build que os define e trabalho
// pendente da Task 11 (scripts/ hoje so tem check_net.ps1); ate la, builds
// locais caem no fallback "dev".
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	// Propaga a versao injetada pelo linker para o handshake MCP — sem isso
	// o cliente ve sempre "dev", mesmo em build de release.
	mcpsrv.Version = version

	root := newRootCmd()

	if err := root.Execute(); err != nil {
		// Erro vai para o stderr, e a decisao de cor sai do PROPRIO stderr.
		// Se stdout estiver redirecionado e o stderr for um terminal, o erro
		// continua colorido -- e o inverso tambem vale.
		console.New(os.Stderr).Err("%v", err)
		os.Exit(1)
	}
}

// newRootCmd monta a arvore de comandos completa.
//
// Existe separada de main() para que o teste exercite a MESMA arvore que o
// binario, e nao uma parecida. SilenceUsage e o exemplo de por que isso
// importa: com ele, um erro de configuracao nao despeja o texto de uso na
// saida -- e um teste que montasse o proprio root sem essa flag afirmaria
// sobre um programa que nao existe.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gobsidian",
		Short:         "Servidor MCP para cofres locais do Obsidian",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newServeCmd(), newDoctorCmd(), newVersionCmd(), newIndexCmd(), newSearchCmd(), newInspectCmd(), newDaemonCmd())

	// A ajuda formatada e instalada na arvore inteira. Isto nao alcanca o
	// stdout de `serve`: o cobra so imprime ajuda quando --help e pedido, e
	// nesse caso o comando nao chega a servir nada.
	console.SetupHelp(root)

	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Imprime versao, commit e data de build",
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "gobsidian %s (%s) %s\n", version, commit, buildDate)
		},
	}
}
