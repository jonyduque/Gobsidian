// Package config resolve a configuracao efetiva do produto a partir de tres
// fontes, na precedencia flag > variavel de ambiente > default. E o unico
// lugar onde essa precedencia e decidida.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cespare/xxhash/v2"
)

// Flags espelha exatamente o que a CLI aceita. cobra preenche esta struct
// e nada mais; a traducao para Config acontece em Load.
//
// ReadOnlySet e DebounceMSSet indicam se a flag correspondente foi
// explicitamente fornecida na linha de comando. Sem isso, um bool ou int
// no valor zero fica indistinguivel de "flag omitida", o que torna a
// direcao "flag desliga o que o env ligou" da precedencia flag > env >
// default inalcancavel. cobra preenche os campos de valor via BoolVar /
// IntVar; os campos *Set sao preenchidos depois do parse, lendo
// cmd.Flags().Changed(nome).
type Flags struct {
	VaultPath     string
	LogLevel      string
	ReadOnly      bool
	ReadOnlySet   bool
	DebounceMS    int
	DebounceMSSet bool
	CacheDir      string
	// EagerSearch nao tem par de variavel de ambiente, entao nao precisa do
	// companheiro *Set: sem uma segunda fonte para desempatar, o valor zero
	// (false, o padrao preguicoso) e indistinguivel de "omitida" so no
	// sentido em que os dois significam a mesma coisa.
	EagerSearch bool
}

// Config e a configuracao ja resolvida, do jeito que o resto do produto a
// consome. Ninguem abaixo desta camada volta a olhar flag ou variavel de
// ambiente: se um valor nao esta aqui, ele nao existe.
type Config struct {
	VaultPath   string
	LogLevel    slog.Level
	ReadOnly    bool
	DebounceMS  int
	CacheDir    string
	MaxResults  int
	EagerSearch bool
}

// Load resolve a configuracao com precedencia flag > env > default.
func Load(f Flags) (Config, error) {
	cfg := Defaults()

	if strings.TrimSpace(f.VaultPath) == "" {
		return Config{}, fmt.Errorf("caminho do cofre nao informado: use --vault")
	}
	abs, err := filepath.Abs(f.VaultPath)
	if err != nil {
		return Config{}, fmt.Errorf("resolvendo caminho do cofre %q: %w", f.VaultPath, err)
	}
	cfg.VaultPath = abs

	if v := os.Getenv("GOBSIDIAN_LOG_LEVEL"); v != "" {
		lvl, err := parseLevel(v)
		if err != nil {
			return Config{}, fmt.Errorf("GOBSIDIAN_LOG_LEVEL: %w", err)
		}
		cfg.LogLevel = lvl
	}
	if f.LogLevel != "" {
		lvl, err := parseLevel(f.LogLevel)
		if err != nil {
			return Config{}, fmt.Errorf("--log-level: %w", err)
		}
		cfg.LogLevel = lvl
	}

	if v := os.Getenv("GOBSIDIAN_READ_ONLY"); v != "" {
		b, err := parseReadOnly(v)
		if err != nil {
			return Config{}, fmt.Errorf("GOBSIDIAN_READ_ONLY: %w", err)
		}
		cfg.ReadOnly = b
	}
	if f.ReadOnlySet {
		cfg.ReadOnly = f.ReadOnly
	}

	if v := os.Getenv("GOBSIDIAN_DEBOUNCE_MS"); v != "" {
		n, err := parseDebounceMS(v)
		if err != nil {
			return Config{}, fmt.Errorf("GOBSIDIAN_DEBOUNCE_MS: %w", err)
		}
		cfg.DebounceMS = n
	}
	if f.DebounceMSSet {
		if err := validateDebounceMS(f.DebounceMS); err != nil {
			return Config{}, fmt.Errorf("--debounce-ms: %w", err)
		}
		cfg.DebounceMS = f.DebounceMS
	}

	cfg.CacheDir = f.CacheDir
	if cfg.CacheDir == "" {
		cfg.CacheDir = defaultCacheDir(cfg.VaultPath)
	}

	cfg.EagerSearch = f.EagerSearch

	return cfg, nil
}

func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("nivel de log desconhecido: %q (use debug, info, warn ou error)", s)
	}
}

// parseReadOnly aceita um conjunto explicito de grafias verdadeiras/falsas,
// case-insensitive, e rejeita qualquer outra coisa em vez de silenciosamente
// coagir para false (um "ture" digitado errado nao pode desligar o modo
// somente-leitura de alguem sem aviso).
func parseReadOnly(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "t", "yes", "y":
		return true, nil
	case "0", "false", "f", "no", "n":
		return false, nil
	default:
		return false, fmt.Errorf("valor desconhecido: %q (use 1, true, t, yes, y, 0, false, f, no ou n)", s)
	}
}

// parseDebounceMS converte e valida um valor de debounce vindo de texto
// (variavel de ambiente). O erro do strconv.Atoi e preservado com %w.
func parseDebounceMS(v string) (int, error) {
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("valor invalido %q (use um inteiro >= 1): %w", v, err)
	}
	if err := validateDebounceMS(n); err != nil {
		return 0, err
	}
	return n, nil
}

// validateDebounceMS aplica a mesma regra de aceitacao independente da
// origem do valor (flag ou env). Zero e recusado: sem coalescencia cada
// evento vira um lote de um caminho so, e a correlacao de rename — que
// exige uma remocao E uma criacao no MESMO lote — para de detectar
// qualquer rename. Servidor sem debounce nao e configuracao que se possa
// pedir por engano, entao a recusa mora aqui e nao no watcher.
func validateDebounceMS(n int) error {
	if n < 1 {
		return fmt.Errorf("valor invalido %d (use um inteiro >= 1)", n)
	}
	return nil
}

// VaultKey deriva um nome de arquivo estavel a partir do caminho absoluto do
// cofre. E a UNICA funcao que calcula essa chave — defaultCacheDir e
// internal/ipc.SocketPath dependem dela para nomear, respectivamente, o
// diretorio de cache e o socket IPC do mesmo cofre. Duas contas separadas do
// mesmo hash e exatamente o padrao que ja custou caro neste projeto (ver
// byAlias em CLAUDE.md): enquanto os dois calculos concordam por coincidencia
// eles nunca divergem, ate que um dos dois lados mude sozinho.
func VaultKey(vaultPath string) string {
	sum := xxhash.Sum64String(strings.ToLower(vaultPath))
	return strconv.FormatUint(sum, 16)
}

// defaultCacheDir deriva o diretorio de cache de um hash do caminho absoluto
// do cofre, sempre FORA do cofre (PRD D1).
func defaultCacheDir(vaultPath string) string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "gobsidian", VaultKey(vaultPath))
}
