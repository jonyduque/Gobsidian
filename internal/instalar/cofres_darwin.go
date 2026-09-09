//go:build darwin

package instalar

import (
	"os"
	"path/filepath"
)

// diretorioDeConfigDoObsidian e o lado macOS.
func diretorioDeConfigDoObsidian() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "Application Support", "obsidian")
}
