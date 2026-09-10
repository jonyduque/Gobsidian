package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jonyduque/Gobsidian/internal/boot"
	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/console"
	"github.com/jonyduque/Gobsidian/internal/vault"
	"github.com/spf13/cobra"
)

type indexSummaryJSON struct {
	VaultPath  string `json:"vault_path"`
	Origin     string `json:"origin"`
	Notes      int    `json:"notes"`
	Assets     int    `json:"assets"`
	Tags       int    `json:"tags"`
	TotalSize  int64  `json:"total_size"`
	DurationMS int64  `json:"duration_ms"`
}

func newIndexCmd() *cobra.Command {
	var flags config.Flags
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Constroi o indice do cofre e exibe um resumo",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(flags)
			if err != nil {
				return err
			}

			v, err := vault.New(cfg.VaultPath, vault.SeguirSymlinks(cfg.FollowSymlinks))
			if err != nil {
				return err
			}

			log := loggerDeCLI(cmd, cfg)
			start := time.Now()
			idx, origem, err := boot.AbrirIndice(cmd.Context(), v, cfg, log)
			if err != nil {
				return err
			}
			dur := time.Since(start)

			notes := idx.NoteCount()
			assets := idx.AssetCount()
			tags := len(idx.Tags("", 1))
			size := idx.TotalSize()

			// index e um subcomando de CLI (nao um servidor MCP JSON-RPC),
			// portanto a escrita em stdout e feita de proposito.
			out := cmd.OutOrStdout()

			if jsonOutput {
				res := indexSummaryJSON{
					VaultPath:  cfg.VaultPath,
					Origin:     origem,
					Notes:      notes,
					Assets:     assets,
					Tags:       tags,
					TotalSize:  size,
					DurationMS: dur.Milliseconds(),
				}
				b, err := json.MarshalIndent(res, "", "  ")
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(out, string(b))
				return nil
			}

			con := console.New(out)
			con.OK("Indexacao concluida em %d ms", dur.Milliseconds())
			con.Campos("Indice", []console.Campo{
				console.Campof("origem", "%s", origem),
				console.Campof("notas", "%d", notes),
				console.Campof("anexos", "%d", assets),
				console.Campof("tags", "%d", tags),
				{Chave: "tamanho", Valor: fmt.Sprintf("%d", size), Nota: "bytes"},
			})
			return nil
		},
	}

	flagsDeCofre(cmd, &flags)
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "saida estruturada em formato JSON")
	flagsDeCache(cmd, &flags)

	return cmd
}
