//go:build !windows

package vault

// IsCloudOnly e sempre falso fora do Windows. Dropbox e Google Drive tem
// mecanismos equivalentes em macOS, mas a deteccao deles nao e por atributo
// de arquivo e fica fora da v1.
func IsCloudOnly(string) bool { return false }
