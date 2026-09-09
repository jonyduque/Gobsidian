//go:build !windows && !darwin

package instalar

import (
	"os"
	"path/filepath"
)

// diretorioDeConfigDoObsidian e o lado Linux. XDG_CONFIG_HOME vale aqui: o
// Obsidian no Linux o respeita, ao contrario do caminho de config do Claude
// Desktop (ver internal/hosts/caminhos_other.go).
func diretorioDeConfigDoObsidian() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "obsidian")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "obsidian")
}
