//go:build !windows

package ipc

import (
	"fmt"
	"os"
	"path/filepath"
)

// runtimeDir e restrictPermission sao o lado Unix (Linux e macOS) do
// transporte IPC: onde o socket mora e como a permissao dele e restringida.
// D-M7-6 fechou o transporte em si (AF_UNIX, sem compilacao condicional); o
// que muda por plataforma e so isto — ver ipc_windows.go para o lado
// Windows.
//
// runtimeDir prefere XDG_RUNTIME_DIR: em sessoes de usuario modernas no
// Linux ele ja vem 0700 e some no logout, o mesmo ciclo de vida que um
// socket de sessao quer. Sem ele — macOS nao define, e alguns Linux
// headless tambem nao — cai para um subdiretorio de os.TempDir() nomeado
// com o uid. /tmp e compartilhado entre usuarios; sem o uid no nome, dois
// usuarios da mesma maquina colidiriam no mesmo caminho de diretorio antes
// mesmo de chegar no socket.
func runtimeDir() (string, error) {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "gobsidian"), nil
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("gobsidian-%d", os.Getuid())), nil
}

// restrictPermission aplica 0600 ao arquivo do socket: so o dono le e
// escreve. E a garantia que substitui a antiga (RNF-30 reformulado pela
// Task 90) — um socket legivel por qualquer um, para um daemon que le o
// cofre, e pior que qualquer preocupacao de rede que RNF-30 tratava antes.
func restrictPermission(path string) error {
	return os.Chmod(path, 0o600)
}
