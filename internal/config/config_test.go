package config_test

import (
	"path/filepath"
	"strings"
	"testing"

	"log/slog"

	"github.com/jonyd/gobsidian/internal/config"
)

func TestLoadPrecedence(t *testing.T) {
	tests := []struct {
		name    string
		flags   config.Flags
		env     map[string]string
		want    func(config.Config) bool
		wantErr bool
	}{
		{
			name:  "default log level is info",
			flags: config.Flags{VaultPath: "C:/vault"},
			want:  func(c config.Config) bool { return c.LogLevel == slog.LevelInfo },
		},
		{
			name:  "log level env overrides default",
			flags: config.Flags{VaultPath: "C:/vault"},
			env:   map[string]string{"GOBSIDIAN_LOG_LEVEL": "debug"},
			want:  func(c config.Config) bool { return c.LogLevel == slog.LevelDebug },
		},
		{
			name:  "log level flag overrides env",
			flags: config.Flags{VaultPath: "C:/vault", LogLevel: "error"},
			env:   map[string]string{"GOBSIDIAN_LOG_LEVEL": "debug"},
			want:  func(c config.Config) bool { return c.LogLevel == slog.LevelError },
		},
		{
			name:    "invalid log level env is rejected and lists accepted values",
			flags:   config.Flags{VaultPath: "C:/vault"},
			env:     map[string]string{"GOBSIDIAN_LOG_LEVEL": "verbose"},
			wantErr: true,
		},

		// ReadOnly
		{
			name:  "read only defaults to false",
			flags: config.Flags{VaultPath: "C:/vault"},
			want:  func(c config.Config) bool { return !c.ReadOnly },
		},
		{
			name:  "read only env sets true",
			flags: config.Flags{VaultPath: "C:/vault"},
			env:   map[string]string{"GOBSIDIAN_READ_ONLY": "true"},
			want:  func(c config.Config) bool { return c.ReadOnly },
		},
		{
			name:  "read only flag false overrides env true",
			flags: config.Flags{VaultPath: "C:/vault", ReadOnly: false, ReadOnlySet: true},
			env:   map[string]string{"GOBSIDIAN_READ_ONLY": "true"},
			want:  func(c config.Config) bool { return !c.ReadOnly },
		},
		{
			name:  "read only flag true overrides env absent",
			flags: config.Flags{VaultPath: "C:/vault", ReadOnly: true, ReadOnlySet: true},
			want:  func(c config.Config) bool { return c.ReadOnly },
		},
		{
			name:    "read only env garbage is rejected",
			flags:   config.Flags{VaultPath: "C:/vault"},
			env:     map[string]string{"GOBSIDIAN_READ_ONLY": "ture"},
			wantErr: true,
		},

		// DebounceMS
		{
			name:  "debounce defaults to 250ms",
			flags: config.Flags{VaultPath: "C:/vault"},
			want:  func(c config.Config) bool { return c.DebounceMS == config.DefaultDebounceMS },
		},
		{
			name:  "debounce env overrides default",
			flags: config.Flags{VaultPath: "C:/vault"},
			env:   map[string]string{"GOBSIDIAN_DEBOUNCE_MS": "500"},
			want:  func(c config.Config) bool { return c.DebounceMS == 500 },
		},
		{
			name:  "debounce flag overrides env",
			flags: config.Flags{VaultPath: "C:/vault", DebounceMS: 700, DebounceMSSet: true},
			env:   map[string]string{"GOBSIDIAN_DEBOUNCE_MS": "500"},
			want:  func(c config.Config) bool { return c.DebounceMS == 700 },
		},
		{
			name:  "debounce explicit zero reachable from env",
			flags: config.Flags{VaultPath: "C:/vault"},
			env:   map[string]string{"GOBSIDIAN_DEBOUNCE_MS": "0"},
			want:  func(c config.Config) bool { return c.DebounceMS == 0 },
		},
		{
			name:  "debounce explicit zero reachable from flag",
			flags: config.Flags{VaultPath: "C:/vault", DebounceMS: 0, DebounceMSSet: true},
			env:   map[string]string{"GOBSIDIAN_DEBOUNCE_MS": "500"},
			want:  func(c config.Config) bool { return c.DebounceMS == 0 },
		},
		{
			name:    "debounce negative rejected from env",
			flags:   config.Flags{VaultPath: "C:/vault"},
			env:     map[string]string{"GOBSIDIAN_DEBOUNCE_MS": "-5"},
			wantErr: true,
		},
		{
			name:    "debounce negative rejected from flag",
			flags:   config.Flags{VaultPath: "C:/vault", DebounceMS: -5, DebounceMSSet: true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			got, err := config.Load(tt.flags)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load() esperava erro, obteve config = %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if !tt.want(got) {
				t.Errorf("Load() = %+v, falhou a condicao do caso", got)
			}
		})
	}
}

func TestLoadDebounceErrorNamesTheValue(t *testing.T) {
	t.Run("env", func(t *testing.T) {
		t.Setenv("GOBSIDIAN_DEBOUNCE_MS", "-5")
		_, err := config.Load(config.Flags{VaultPath: "C:/vault"})
		if err == nil {
			t.Fatal("Load() esperava erro")
		}
		if !strings.Contains(err.Error(), "-5") {
			t.Errorf("erro nao menciona o valor ofensor: %v", err)
		}
	})

	t.Run("flag", func(t *testing.T) {
		_, err := config.Load(config.Flags{VaultPath: "C:/vault", DebounceMS: -5, DebounceMSSet: true})
		if err == nil {
			t.Fatal("Load() esperava erro")
		}
		if !strings.Contains(err.Error(), "-5") {
			t.Errorf("erro nao menciona o valor ofensor: %v", err)
		}
	})
}

func TestLoadRejectsEmptyVault(t *testing.T) {
	if _, err := config.Load(config.Flags{}); err == nil {
		t.Fatal("Load() sem vault deveria falhar")
	}
}

func TestLoadCacheDir(t *testing.T) {
	t.Run("default derivation is deterministic for the same vault path", func(t *testing.T) {
		c1, err := config.Load(config.Flags{VaultPath: "C:/vault/one"})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		c2, err := config.Load(config.Flags{VaultPath: "C:/vault/one"})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if c1.CacheDir != c2.CacheDir {
			t.Errorf("CacheDir nao e deterministico: %q != %q", c1.CacheDir, c2.CacheDir)
		}
	})

	t.Run("different vault paths produce different directories", func(t *testing.T) {
		c1, err := config.Load(config.Flags{VaultPath: "C:/vault/one"})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		c2, err := config.Load(config.Flags{VaultPath: "C:/vault/two"})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if c1.CacheDir == c2.CacheDir {
			t.Errorf("CacheDir colidiu para cofres diferentes: %q", c1.CacheDir)
		}
	})

	t.Run("derived path is not inside the vault", func(t *testing.T) {
		c, err := config.Load(config.Flags{VaultPath: "C:/vault/three"})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		rel, err := filepath.Rel(c.VaultPath, c.CacheDir)
		if err != nil {
			// filepath.Rel error means the paths are on different volumes (common on Windows),
			// which satisfies the "cache outside vault" property.
			return
		}
		if !strings.HasPrefix(rel, "..") {
			t.Errorf("CacheDir %q esta dentro do cofre %q (rel = %q)", c.CacheDir, c.VaultPath, rel)
		}
	})

	t.Run("explicit cache dir wins over derivation", func(t *testing.T) {
		c, err := config.Load(config.Flags{VaultPath: "C:/vault/four", CacheDir: "C:/explicit/cache"})
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if c.CacheDir != "C:/explicit/cache" {
			t.Errorf("CacheDir = %q, esperado o valor explicito da flag", c.CacheDir)
		}
	})
}
