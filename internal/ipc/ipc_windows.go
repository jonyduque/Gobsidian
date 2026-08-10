//go:build windows

package ipc

import (
	"fmt"
	"os"
	"path/filepath"
)

// runtimeDir e restrictPermission sao o lado Windows do transporte IPC: onde
// o socket mora e como a permissao dele e restringida. D-M7-6 fechou o
// transporte em si (AF_UNIX, sem compilacao condicional, requer Windows 10
// 1803+); o que muda por plataforma e so isto — ver ipc_unix.go para o lado
// Unix.
//
// runtimeDir usa o mesmo diretorio base que config.defaultCacheDir ja usa
// para o cache — %LocalAppData%, dentro do perfil do usuario. Nao ha
// equivalente Windows a XDG_RUNTIME_DIR; o perfil do usuario e o analogo
// mais proximo porque a ACL dele ja restringe acesso ao proprio usuario e a
// administradores, sem nada que este pacote precise configurar.
func runtimeDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolvendo %%LocalAppData%%: %w", err)
	}
	return filepath.Join(base, "gobsidian", "run"), nil
}

// restrictPermission e no-op no Windows: o arquivo do socket herda a ACL do
// diretorio que o contem (decisao 4 da Task 91), e runtimeDir aponta para
// dentro do perfil do usuario. Nao ha chmod nesta plataforma — e por isso a
// verificacao real e um teste que tenta abrir o socket como outro usuario,
// nao uma chamada aqui.
func restrictPermission(string) error { return nil }
