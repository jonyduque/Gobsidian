package main

import (
	"os"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/console"
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
			flags.DebounceMSSet = cmd.Flags().Changed("debounce-ms")
			flags.MaxResultsSet = cmd.Flags().Changed("max-results")

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
			// A cor sai do writer do comando, nao de os.Stdout: quem faz
			// `gobsidian doctor > relatorio.txt` recebe um arquivo limpo.
			con := console.New(cmd.OutOrStdout())
			for _, r := range results {
				switch r.Status {
				case doctor.StatusOK:
					con.OK("%s", r.Name)
				case doctor.StatusWarn:
					con.Warn("%s", r.Name)
				default:
					con.Err("%s", r.Name)
				}
				if r.Detail != "" {
					con.Detail("%s", r.Detail)
				}
			}

			code := doctor.ExitCode(results)
			if code != 0 {
				con.Err("Ha falhas bloqueantes acima")
				os.Exit(code)
			}
			con.OK("Ambiente apto")
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "nao verifica permissao de escrita")
	cmd.Flags().IntVar(&flags.DebounceMS, "debounce-ms", 0, "janela de coalescencia de eventos do watcher")
	cmd.Flags().IntVar(&flags.MaxResults, "max-results", 0, "teto de resultados por consulta")
	cmd.Flags().BoolVar(&flags.FollowSymlinks, "follow-symlinks", false,
		"segue symlink dentro do cofre; o padrao recusa, porque o confinamento nao alcanca o alvo")

	return cmd
}
