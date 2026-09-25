package main

import (
	"github.com/jonyduque/Gobsidian/internal/textos"
	"github.com/spf13/cobra"

	"github.com/jonyduque/Gobsidian/internal/config"
)

// flagsDeCofre registra as flags que todo subcomando que abre um cofre
// aceita. Seis arquivos registravam --vault e --follow-symlinks com o mesmo
// texto; quando o texto mudou, mudou em cinco.
func flagsDeCofre(cmd *cobra.Command, f *config.Flags) {
	cmd.Flags().StringVar(&f.VaultPath, "vault", "", textos.FlagVault)
	cmd.Flags().BoolVar(&f.FollowSymlinks, "follow-symlinks", false,
		textos.FlagFollowSymlinks)
}

// flagsDeCache registra as flags de quem le ou grava o cache de indice.
func flagsDeCache(cmd *cobra.Command, f *config.Flags) {
	cmd.Flags().StringVar(&f.CacheDir, "cache-dir", "", textos.FlagCacheDir)
	cmd.Flags().StringVar(&f.LogLevel, "log-level", "", textos.FlagLogLevel)
}
