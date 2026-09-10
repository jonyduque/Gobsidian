package main

import (
	"encoding/json"
	"fmt"

	"github.com/jonyduque/Gobsidian/internal/boot"
	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/console"
	"github.com/jonyduque/Gobsidian/internal/search"
	"github.com/jonyduque/Gobsidian/internal/service"
	"github.com/jonyduque/Gobsidian/internal/vault"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var flags config.Flags
	var jsonOutput bool
	var limit int

	cmd := &cobra.Command{
		Use:   "search <consulta>",
		Short: "Executa busca por texto completo no cofre",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.MaxResultsSet = cmd.Flags().Changed("max-results")

			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}

			v, err := vault.New(cfg.VaultPath, vault.SeguirSymlinks(cfg.FollowSymlinks))
			if err != nil {
				return err
			}

			log := loggerDeCLI(cmd, cfg)
			idx, _, err := boot.AbrirIndice(cmd.Context(), v, cfg, log)
			if err != nil {
				return err
			}
			inv := search.NewInverted()
			inv.MarkBuilding()
			boot.PrepararBusca(cmd.Context(), v, idx, inv, cfg, log)
			defer func() { _ = inv.Close() }()

			svc := service.New(v, idx, inv, nil, service.Options{ReadOnly: cfg.ReadOnly, MaxResults: cfg.MaxResults})

			res, err := svc.Search(cmd.Context(), service.SearchOptions{
				Query: args[0],
				Limit: limit,
			})
			if err != nil {
				return err
			}

			// search e um subcomando de CLI (nao um servidor MCP JSON-RPC),
			// portanto a escrita em stdout e feita de proposito.
			out := cmd.OutOrStdout()

			if jsonOutput {
				b, err := json.MarshalIndent(res, "", "  ")
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(out, string(b))
				return nil
			}

			con := console.New(out)
			if len(res.Results) == 0 {
				con.Info("Nenhum resultado para %q", args[0])
				return nil
			}

			// Um bloco por resultado seria uma moldura a cada duas linhas; o
			// que se le aqui e uma LISTA, e a moldura serve para delimita-la,
			// nao para separar item de item.
			corpos := make([]string, 0, len(res.Results)*2)
			for _, m := range res.Results {
				corpos = append(corpos, fmt.Sprintf("  %s  %s",
					m.Path, con.Dim(fmt.Sprintf("%.2f", m.Score))))
				if m.Snippet != "" {
					corpos = append(corpos, "    "+con.Dim("... "+m.Snippet+" ..."))
				}
			}
			con.Bloco(fmt.Sprintf("%d de %d para %q", len(res.Results), res.Total, args[0]), corpos, "")
			return nil
		},
	}

	flagsDeCofre(cmd, &flags)
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "saida estruturada em formato JSON")
	cmd.Flags().IntVar(&limit, "limit", 20, "limite maximo de resultados")
	cmd.Flags().IntVar(&flags.MaxResults, "max-results", 0, "teto de resultados por consulta")
	flagsDeCache(cmd, &flags)

	return cmd
}
