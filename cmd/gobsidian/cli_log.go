package main

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/jonyduque/Gobsidian/internal/config"
)

// loggerDeCLI e o logger dos subcomandos de CLI (search, index, inspect):
// escreve em cmd.ErrOrStderr() para que teste capture, e cala tudo abaixo de
// Warn a menos que o operador tenha pedido um nivel — um subcomando que
// imprime "servidor pronto" a cada chamada polui o terminal de quem so
// queria o resultado.
func loggerDeCLI(cmd *cobra.Command, cfg config.Config) *slog.Logger {
	nivel := slog.LevelWarn
	if cfg.LogLevelExplicito {
		nivel = cfg.LogLevel
	}
	return slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: nivel}))
}
