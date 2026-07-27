package main

import (
	"fmt"
	"os"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var flags config.Flags

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostica o ambiente: permissoes, OneDrive, MAX_PATH, casing",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Sem esta linha, --read-only nao chega a Config e a verificacao
			// de permissao de escrita roda mesmo quando o usuario pediu para
			// nao rodar. Toda chamada a config.Load precisa preencher os
			// companheiros das flags que o comando expoe.
			flags.ReadOnlySet = cmd.Flags().Changed("read-only")

			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}

			results := doctor.Run(cmd.Context(), cfg)

			// doctor imprime em stdout de proposito: e um comando de CLI,
			// nao um servidor. Nenhum JSON-RPC trafega aqui.
			//
			// O erro de escrita e descartado explicitamente, e nao por
			// esquecimento: se o proprio relatorio nao sai, nao sobra canal
			// para reclamar disso.
			out := cmd.OutOrStdout()
			for _, r := range results {
				_, _ = fmt.Fprintf(out, "%s %s\n", r.Status.Marker(), r.Name)
				if r.Detail != "" {
					_, _ = fmt.Fprintf(out, "     %s\n", r.Detail)
				}
			}

			code := doctor.ExitCode(results)
			if code != 0 {
				_, _ = fmt.Fprintln(out, "[!] Ha falhas bloqueantes acima")
				os.Exit(code)
			}
			_, _ = fmt.Fprintln(out, "[OK] Ambiente apto")
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "nao verifica permissao de escrita")

	return cmd
}
