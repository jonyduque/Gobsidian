//go:build !windows

package vault

// LongPath e identidade fora do Windows: nao ha MAX_PATH a contornar.
func LongPath(abs string) string { return abs }
