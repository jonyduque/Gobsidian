//go:build windows

package instalar

import (
	"os"
	"path/filepath"
)

// diretorioDeConfigDoObsidian e o lado Windows: %APPDATA%\obsidian.
func diretorioDeConfigDoObsidian() string {
	return filepath.Join(os.Getenv("APPDATA"), "obsidian")
}
