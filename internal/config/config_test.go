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
