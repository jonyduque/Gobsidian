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
type Flags struct {
	VaultPath  string
	LogLevel   string
	ReadOnly   bool
	DebounceMS int
	CacheDir   string
}

type Config struct {
	VaultPath  string
	LogLevel   slog.Level
	ReadOnly   bool
	DebounceMS int
	CacheDir   string
	MaxResults int
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
		cfg.ReadOnly = v == "1" || strings.EqualFold(v, "true")
	}
	if f.ReadOnly {
		cfg.ReadOnly = true
	}

	if v := os.Getenv("GOBSIDIAN_DEBOUNCE_MS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return Config{}, fmt.Errorf("GOBSIDIAN_DEBOUNCE_MS invalido: %q", v)
		}
		cfg.DebounceMS = n
	}
	if f.DebounceMS > 0 {
		cfg.DebounceMS = f.DebounceMS
	}

	cfg.CacheDir = f.CacheDir
	if cfg.CacheDir == "" {
		cfg.CacheDir = defaultCacheDir(cfg.VaultPath)
	}

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

// defaultCacheDir deriva o diretorio de cache de um hash do caminho absoluto
// do cofre, sempre FORA do cofre (PRD D1).
func defaultCacheDir(vaultPath string) string {
	sum := xxhash.Sum64String(strings.ToLower(vaultPath))
	name := strconv.FormatUint(sum, 16)

	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "gobsidian", name)
}
