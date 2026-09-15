//go:build windows

package ipc

import (
	"fmt"
	"os"
	"path/filepath"
)

// runtimeDirDoSistema e restrictPermission sao o lado Windows do transporte
// IPC: onde o socket mora e como a permissao dele e restringida. D-M7-6 fechou
// o transporte em si (AF_UNIX, sem compilacao condicional, requer Windows 10
// 1803+); o que muda por plataforma e so isto -- ver ipc_unix.go para o lado
// Unix.
//
// # Por que %USERPROFILE%\.gobsidian\run, e nao %LOCALAPPDATA%
//
// Ate 2026-09-14 o diretorio era %LOCALAPPDATA%\gobsidian\run. Medido naquele
// dia com tools/sondahost, dentro dos processos que o Claude Desktop 1.52386.6
// cria (pacote MSIX com FileSystemWriteVirtualization):
//
//   - nenhum socket em %LOCALAPPDATA% aceita conexao, nem o que o proprio
//     processo acabou de criar: lstat 1920, dial 10022;
//   - em %USERPROFILE%\.gobsidian-sondahost\run o socket proprio e o socket
//     criado por uma shell elevada conectam;
//   - um diretorio novo criado pelo processo na raiz de %LOCALAPPDATA% vai para
//     uma copia privada do pacote (Packages\...\LocalCache\Local), invisivel
//     para qualquer outro host.
//
// Desde 2026-08-24 nenhuma ponte aberta pelo Desktop alcancava o daemon. O
// diretorio INTEIRO mudou -- socket, travas, log, presenca e instalacao.lock --
// por decisao do dono no mesmo dia: mover so o socket deixaria as travas numa
// copia privada em toda maquina onde o Desktop cria a pasta primeiro.
//
// O perfil continua sendo o do usuario: a ACL dele ja restringe acesso ao
// proprio usuario e a administradores, sem nada que este pacote configure.
func runtimeDirDoSistema() (string, error) {
	perfil, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolvendo %%USERPROFILE%%: %w", err)
	}
	return filepath.Join(perfil, ".gobsidian", "run"), nil
}

// runtimeDirAntigoDoSistema e onde versoes ate 2026-09-14 guardavam socket,
// travas, log e presenca. Serve a transicao: saber se ha um daemon de versao
// anterior servindo um cofre, e varrer o lixo que ficou la.
func runtimeDirAntigoDoSistema() string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "gobsidian", "run")
}

// restrictPermission e no-op no Windows: o arquivo do socket herda a ACL do
// diretorio que o contem (decisao 4 da Task 91), e runtimeDirDoSistema aponta
// para dentro do perfil do usuario. Nao ha chmod nesta plataforma -- e por isso
// a verificacao real e um teste que tenta abrir o socket como outro usuario,
// nao uma chamada aqui.
func restrictPermission(string) error { return nil }
