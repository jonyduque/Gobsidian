package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Injetados pelo linker. Ver scripts/build.ps1.
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
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
