### Task 2: Configuração

**Files:**
- Create: `internal/config/config.go`, `internal/config/defaults.go`, `internal/config/config_test.go`

**Interfaces:**
- Consumes: nada
- Produces: `config.Config` com campos `VaultPath string`, `LogLevel slog.Level`, `ReadOnly bool`, `DebounceMS int`, `CacheDir string`, `MaxResults int`; `config.Load(flags Flags) (Config, error)`; `config.Defaults() Config`

- [ ] **Step 1: Escrever o teste de precedência**

`internal/config/config_test.go`:

```go
package config_test

import (
	"log/slog"
	"testing"

	"github.com/jonyd/gobsidian/internal/config"
)

func TestLoadPrecedence(t *testing.T) {
	tests := []struct {
		name  string
		flags config.Flags
		env   map[string]string
		want  func(config.Config) bool
	}{
		{
			name:  "default log level is info",
			flags: config.Flags{VaultPath: "C:/vault"},
			want:  func(c config.Config) bool { return c.LogLevel == slog.LevelInfo },
		},
		{
			name:  "env overrides default",
			flags: config.Flags{VaultPath: "C:/vault"},
			env:   map[string]string{"GOBSIDIAN_LOG_LEVEL": "debug"},
			want:  func(c config.Config) bool { return c.LogLevel == slog.LevelDebug },
		},
		{
			name:  "flag overrides env",
			flags: config.Flags{VaultPath: "C:/vault", LogLevel: "error"},
			env:   map[string]string{"GOBSIDIAN_LOG_LEVEL": "debug"},
			want:  func(c config.Config) bool { return c.LogLevel == slog.LevelError },
		},
		{
			name:  "read only defaults to false",
			flags: config.Flags{VaultPath: "C:/vault"},
			want:  func(c config.Config) bool { return !c.ReadOnly },
		},
		{
			name:  "debounce defaults to 250ms",
			flags: config.Flags{VaultPath: "C:/vault"},
			want:  func(c config.Config) bool { return c.DebounceMS == 250 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			got, err := config.Load(tt.flags)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if !tt.want(got) {
				t.Errorf("Load() = %+v, falhou a condicao do caso", got)
			}
		})
	}
}

func TestLoadRejectsEmptyVault(t *testing.T) {
	if _, err := config.Load(config.Flags{}); err == nil {
		t.Fatal("Load() sem vault deveria falhar")
	}
}
```

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/config/ -v`
Esperado: FAIL — `undefined: config.Flags`.

- [ ] **Step 3: Implementar**

`internal/config/defaults.go`:

```go
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
```

`internal/config/config.go`:

```go
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
```

- [ ] **Step 4: Rodar para confirmar que passa**

Run: `go test -race ./internal/config/ -v`
Esperado: PASS, seis casos.

- [ ] **Step 5: Commit**

```bash
git add internal/config
git commit -m "feat(config): load configuration with flag over env over default precedence"
```

---

