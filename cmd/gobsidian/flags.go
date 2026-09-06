package main

import (
	"github.com/spf13/cobra"

	"github.com/jonyd/gobsidian/internal/config"
)

// flagsDeCofre registra as flags que todo subcomando que abre um cofre
// aceita. Seis arquivos registravam --vault e --follow-symlinks com o mesmo
// texto; quando o texto mudou, mudou em cinco.
func flagsDeCofre(cmd *cobra.Command, f *config.Flags) {
	cmd.Flags().StringVar(&f.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
	cmd.Flags().BoolVar(&f.FollowSymlinks, "follow-symlinks", false,
		"segue symlink dentro do cofre; o padrao recusa, porque o confinamento nao alcanca o alvo")
}

// flagsDeCache registra as flags de quem le ou grava o cache de indice.
func flagsDeCache(cmd *cobra.Command, f *config.Flags) {
	cmd.Flags().StringVar(&f.CacheDir, "cache-dir", "", "diretorio do cache de indice")
	cmd.Flags().StringVar(&f.LogLevel, "log-level", "", "debug, info, warn ou error")
}
