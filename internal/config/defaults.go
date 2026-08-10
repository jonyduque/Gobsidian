package config

import "log/slog"

// Todos os valores padrao do produto vivem aqui. Nenhum outro arquivo
// declara um default; procurar por um valor magico deve terminar neste arquivo.
const (
	DefaultDebounceMS = 250
	DefaultMaxResults = 50
	MaxResultsCeiling = 500
)

// Defaults devolve a Config antes de qualquer flag ou variavel de ambiente
// ser aplicada. VaultPath fica vazio de proposito: ele nao tem default
// defensavel, e Load exige que alguem o forneca.
func Defaults() Config {
	return Config{
		LogLevel:    slog.LevelInfo,
		ReadOnly:    false,
		DebounceMS:  DefaultDebounceMS,
		MaxResults:  DefaultMaxResults,
		EagerSearch: false,
	}
}
