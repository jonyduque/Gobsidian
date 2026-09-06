package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jonyd/gobsidian/internal/boot"
	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/console"
	"github.com/jonyd/gobsidian/internal/vault"
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
			con.Item("Origem: %s", origem)
			con.Item("Notas: %d", notes)
			con.Item("Anexos: %d", assets)
			con.Item("Tags: %d", tags)
			con.Item("Tamanho total: %d bytes", size)
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "saida estruturada em formato JSON")
	cmd.Flags().BoolVar(&flags.FollowSymlinks, "follow-symlinks", false,
		"segue symlink dentro do cofre; o padrao recusa, porque o confinamento nao alcanca o alvo")
	cmd.Flags().StringVar(&flags.CacheDir, "cache-dir", "", "diretorio do cache de indice")
	cmd.Flags().StringVar(&flags.LogLevel, "log-level", "", "debug, info, warn ou error")

	return cmd
}
