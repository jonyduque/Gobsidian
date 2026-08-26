package ipc

import (
	"os"
	"strings"
	"testing"
)

// TestListenRecusaSocketDeDaemonVivo cobre o A5.
//
// `ipc.Listen` chamava `cleanupSocketFile` INCONDICIONALMENTE antes do
// `net.Listen`. Um segundo daemon removia o arquivo do socket do primeiro e
// bindava no mesmo nome: o daemon vivo ficava inalcançável, com memória e
// watcher rodando, e **duas instâncias passavam a gravar concorrentemente no
// MESMO cache de busca** — corrupção por escrita intercalada.
//
// O critério de "há daemon vivo" NÃO pode ser o errno. Medido em 2026-08-26 com
// `net.Dial("unix", ...)` no Windows:
//
//	10061 ECONNREFUSED  arquivo comum, socket órfão E caminho inexistente
//	10022 EINVAL        diretório no lugar do socket
//
// Errnos diferentes descrevem o mesmo estado, e o mesmo errno descreve estados
// diferentes. O critério é comportamental: alguém aceita conexão naquele
// socket?
func TestListenRecusaSocketDeDaemonVivo(t *testing.T) {
	cofre := t.TempDir()

	primeiro, path, err := Listen(cofre)
	if err != nil {
		t.Skipf("socket unix indisponivel nesta maquina: %v", err)
	}
	defer func() { _ = primeiro.Close() }()

	// O primeiro daemon precisa estar ACEITANDO, senão o cenário não é
	// "daemon vivo" — é "socket bound sem ouvinte", que é outro caso.
	go func() {
		for {
			c, err := primeiro.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()

	segundo, _, err := Listen(cofre)
	if err == nil {
		_ = segundo.Close()
		t.Fatal("o segundo Listen ROUBOU o socket de um daemon vivo: " +
			"o primeiro ficou inalcancavel, e as duas instancias gravariam " +
			"no mesmo cache de busca")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "ativo") {
		t.Errorf("o erro nao diz que ja ha daemon ativo: %v", err)
	}

	// E o arquivo do primeiro NAO pode ter sido tocado.
	if _, errStat := os.Lstat(path); errStat != nil {
		t.Errorf("o socket do daemon vivo foi REMOVIDO apesar da recusa: %v", errStat)
	}
}

// TestListenLimpaSocketOrfao é o contrapeso: sem ele, a correção poderia ser
// "nunca limpar", e um socket deixado por um daemon morto travaria toda partida
// seguinte para sempre — que é exatamente o defeito de campo de 2026-08-26,
// onde três sessões pagavam 10 s por partida durante dias.
func TestListenLimpaSocketOrfao(t *testing.T) {
	cofre := t.TempDir()

	primeiro, path, err := Listen(cofre)
	if err != nil {
		t.Skipf("socket unix indisponivel nesta maquina: %v", err)
	}
	// Fecha SEM remover o arquivo: e o estado que um daemon morto a forca
	// deixa. O Close do Go desvincula, entao recriamos o arquivo para montar
	// o cenario.
	_ = primeiro.Close()
	if _, errStat := os.Lstat(path); os.IsNotExist(errStat) {
		f, errCreate := os.Create(path)
		if errCreate != nil {
			t.Fatalf("recriando arquivo orfao: %v", errCreate)
		}
		_ = f.Close()
	}

	segundo, _, err := Listen(cofre)
	if err != nil {
		t.Fatalf("Listen recusou um socket ORFAO; toda partida seguinte ficaria "+
			"travada: %v", err)
	}
	_ = segundo.Close()
}
