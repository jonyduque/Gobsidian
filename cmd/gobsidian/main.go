package main

import (
	"fmt"
	"os"

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

	root := &cobra.Command{
		Use:           "gobsidian",
		Short:         "Servidor MCP para cofres locais do Obsidian",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newServeCmd(), newDoctorCmd(), newVersionCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "[!] %v\n", err)
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Imprime versao, commit e data de build",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "gobsidian %s (%s) %s\n", version, commit, buildDate)
		},
	}
}
