package ipc

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// ErrDiretorioSemSocket diz que, NESTE processo, um socket criado no diretorio
// de sockets nao aceita conexao -- nem o que o proprio processo acabou de
// criar.
//
// # O estado que isto nomeia
//
// Medido em 2026-09-14 (plano 2026-09-11, Parte G): os processos que o Claude
// Desktop cria nao conseguem usar socket AF_UNIX em lugar nenhum de
// %LOCALAPPDATA%. `listen` funciona, mas `lstat` do arquivo da 1920 e `dial`
// da 10022, inclusive no socket que o mesmo processo criou um instante antes.
// Fora do host, na shell do usuario, o mesmo diretorio funciona. Desde
// 2026-08-24 nenhuma ponte aberta pelo Desktop alcancou o daemon: todas cairam
// para o modo em processo, e cada uma subiu um daemon que morreu logando
// `daemon nao pode abrir o socket`.
var ErrDiretorioSemSocket = errors.New("diretorio de sockets nao aceita conexao neste processo")

// sondaDeDiretorio e SondarDiretorioDeSockets numa variavel, para o teste de
// Listen simular o contexto do host sem precisar do host. Producao nunca a
// troca.
var sondaDeDiretorio = SondarDiretorioDeSockets

// SondarDiretorioDeSockets cria um socket descartavel em dir e tenta conectar
// nele. Devolve nil se conectou, ou um erro que embrulha ErrDiretorioSemSocket.
//
// O criterio e COMPORTAMENTAL, como o de AlguemEscuta, e pelo mesmo motivo:
// errnos diferentes descrevem o mesmo estado e o mesmo errno descreve estados
// diferentes. A pergunta que o protocolo responde e uma so -- um socket criado
// aqui aceita conexao deste processo?
//
// E ela que da sentido a AlguemEscuta. "Ninguem escuta" num diretorio onde nem
// o proprio socket conecta nao diz nada sobre o daemon que talvez esteja la.
func SondarDiretorioDeSockets(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("%w: %s: criando o diretorio: %w", ErrDiretorioSemSocket, dir, err)
	}
	// O nome tem o MESMO comprimento do socket de um cofre (16 caracteres +
	// ".sock"). O caminho de um AF_UNIX tem pouco mais de 100 bytes, e uma sonda
	// de nome mais longo reprovaria um diretorio onde o socket real cabe: a
	// primeira redacao, "sonda-<pid>-<nanos>.sock", deu `bind: invalid argument`
	// no proprio teste deste arquivo.
	caminho := filepath.Join(dir, fmt.Sprintf("s%015x.sock", (uint64(time.Now().UnixNano())^uint64(os.Getpid())<<40)&(1<<60-1)))
	ln, err := net.Listen("unix", caminho)
	if err != nil {
		return fmt.Errorf("%w: %s: listen: %w", ErrDiretorioSemSocket, dir, err)
	}
	defer func() {
		_ = ln.Close()
		_ = os.Remove(caminho)
	}()
	go func() {
		if c, err := ln.Accept(); err == nil {
			_ = c.Close()
		}
	}()

	c, err := net.Dial("unix", caminho)
	if err != nil {
		return fmt.Errorf("%w: %s: o socket recem-criado nao aceita conexao: %w", ErrDiretorioSemSocket, dir, err)
	}
	_ = c.Close()
	return nil
}

// SondarSocketDoCofre sonda o diretorio onde o socket do cofre mora. E a conta
// que a ponte e o doctor usam: os dois perguntam sobre o cofre, e o diretorio
// sai de SocketPath, a unica conta do caminho do socket.
func SondarSocketDoCofre(vaultPath string) error {
	path, err := SocketPath(vaultPath)
	if err != nil {
		return err
	}
	return SondarDiretorioDeSockets(filepath.Dir(path))
}
