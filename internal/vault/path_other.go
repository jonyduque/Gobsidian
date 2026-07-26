//go:build !windows

package vault

// validatePlatformPath nao tem o que rejeitar fora do Windows.
//
// Nomes de dispositivo nao existem, e ponto ou espaco no fim de um componente
// sao bytes legitimos que o sistema de arquivos preserva. Rejeita-los aqui
// tornaria notas reais inalcancaveis.
func validatePlatformPath(string) error { return nil }
