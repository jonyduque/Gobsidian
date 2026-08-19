package main

import (
	"encoding/json"
	"fmt"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/console"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
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
			flags.ReadOnlySet = cmd.Flags().Changed("read-only")
			flags.DebounceMSSet = cmd.Flags().Changed("debounce-ms")
			flags.MaxResultsSet = cmd.Flags().Changed("max-results")

			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}

			v, err := vault.New(cfg.VaultPath)
			if err != nil {
				return err
			}

			idx := index.New()
			if err := idx.Build(cmd.Context(), v); err != nil {
				return err
			}

			inv := search.NewInverted()
			for _, p := range idx.NotePaths() {
				_ = inv.Update(cmd.Context(), v, p)
			}

			svc := service.New(v, idx, inv, nil, service.Options{ReadOnly: cfg.ReadOnly})

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

			con.OK("%d resultado(s) para %q (total: %d):", len(res.Results), args[0], res.Total)
			for _, m := range res.Results {
				con.Item("%s (score: %.2f)", m.Path, m.Score)
				if m.Snippet != "" {
					con.Detail("... %s ...", m.Snippet)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "saida estruturada em formato JSON")
	cmd.Flags().IntVar(&limit, "limit", 20, "limite maximo de resultados")
	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "desabilita verificacoes de escrita")
	cmd.Flags().IntVar(&flags.DebounceMS, "debounce-ms", 0, "janela de coalescencia de eventos do watcher")
	cmd.Flags().IntVar(&flags.MaxResults, "max-results", 0, "teto de resultados por consulta")

	return cmd
}
