package config

import "log/slog"

// Todos os valores padrao do produto vivem aqui. Nenhum outro arquivo
// declara um default; procurar por um valor magico deve terminar neste arquivo.
const (
	DefaultDebounceMS = 250
	DefaultMaxResults = 50
	MaxResultsCeiling = 500
)

func Defaults() Config {
	return Config{
		LogLevel:   slog.LevelInfo,
		ReadOnly:   false,
		DebounceMS: DefaultDebounceMS,
		MaxResults: DefaultMaxResults,
	}
}
