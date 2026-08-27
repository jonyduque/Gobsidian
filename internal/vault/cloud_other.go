//go:build !windows

package vault

import "io/fs"

// IsCloudOnly e sempre falso fora do Windows. Dropbox e Google Drive tem
// mecanismos equivalentes em macOS, mas a deteccao deles nao e por atributo
// de arquivo e fica fora da v1.
func IsCloudOnly(string) bool { return false }

// IsCloudOnlyInfo acompanha IsCloudOnly: fora do Windows nao ha atributo de
// placeholder para consultar, nem no caminho nem no FileInfo.
func IsCloudOnlyInfo(fs.FileInfo) bool { return false }
