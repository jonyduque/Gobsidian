package ipc_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/ipc"
)

// boundedWait e o prazo usado para esperar por algo neste pacote. Um defeito
// real (por exemplo, o handshake travando em vez de devolver erro) nao pode
// travar "go test -race ./..." ate o timeout padrao de 10 minutos.
const boundedWait = 3 * time.Second

func TestSocketPathDeterministicoEMesmaChaveDoCache(t *testing.T) {
	p1, err := ipc.SocketPath("C:/cofre/um")
	if err != nil {
		t.Fatalf("SocketPath() error = %v", err)
	}
	p2, err := ipc.SocketPath("C:/cofre/um")
	if err != nil {
		t.Fatalf("SocketPath() error = %v", err)
	}
	if p1 != p2 {
		t.Fatalf("SocketPath nao e deterministico: %q != %q", p1, p2)
	}

	p3, err := ipc.SocketPath("C:/cofre/dois")
	if err != nil {
		t.Fatalf("SocketPath() error = %v", err)
	}
	if p1 == p3 {
		t.Fatalf("SocketPath colidiu para cofres diferentes: %q", p1)
	}

	// SocketPath tem de usar a MESMA chave que defaultCacheDir usa para o
	// diretorio de cache (config.VaultKey) -- e nao um hash calculado de
	// novo aqui, que divergiria do outro do jeito que byAlias divergiu em
	// alias.go/resolve.go (ver CLAUDE.md). O nome do arquivo do socket
	// carrega essa chave, entao ela tem que aparecer literalmente nele.
	chave := config.VaultKey("C:/cofre/um")
	if !strings.Contains(filepath.Base(p1), chave) {
		t.Fatalf("SocketPath(%q) = %q, esperado conter a chave do cofre %q (config.VaultKey)", "C:/cofre/um", p1, chave)
	}
}

func TestDialAndHandshakeSocketAusente(t *testing.T) {
	vault := t.TempDir()

	start := time.Now()
	conn, err := ipc.DialAndHandshake(context.Background(), vault, 200*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		_ = conn.Close()
		t.Fatal("DialAndHandshake() error = nil, esperado erro (nenhum daemon escutando)")
	}
	if elapsed > boundedWait {
		t.Fatalf("DialAndHandshake demorou %s para desistir de um socket ausente, esperado abaixo de %s", elapsed, boundedWait)
	}
}

func TestDialAndHandshakeRoundTrip(t *testing.T) {
	vault := t.TempDir()

	ln, path, err := ipc.Listen(vault)
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	t.Logf("socket em %s", path)

	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		if err := ipc.Greet(c); err != nil {
			t.Errorf("Greet() error = %v", err)
			return
		}
		accepted <- c
	}()

	ctx, cancel := context.WithTimeout(context.Background(), boundedWait)
	defer cancel()

	conn, err := ipc.DialAndHandshake(ctx, vault, boundedWait)
	if err != nil {
		t.Fatalf("DialAndHandshake() error = %v", err)
	}
	defer func() { _ = conn.Close() }()

	var serverSide net.Conn
	select {
	case serverSide = <-accepted:
	case <-time.After(boundedWait):
		t.Fatal("servidor nao aceitou a conexao a tempo")
	}
	defer func() { _ = serverSide.Close() }()

	// A ponte e burra: depois do handshake, o que passa pelo socket nao e
	// interpretado, so copiado. Provar isso aqui e escrever um byte cru de
	// cada lado e conferir que o outro lado recebe exatamente aquilo, sem
	// nada do protocolo de saudacao sobrando no meio -- e a prova de que
	// readGreeting nao "roubou" bytes que pertenciam ao proxy (a armadilha
	// que um bufio.Reader introduziria).
	if _, err := fmt.Fprint(conn, "ola daemon"); err != nil {
		t.Fatalf("escrevendo no cliente: %v", err)
	}
	buf := make([]byte, len("ola daemon"))
	if _, err := io.ReadFull(serverSide, buf); err != nil {
		t.Fatalf("lendo no servidor: %v", err)
	}
	if string(buf) != "ola daemon" {
		t.Fatalf("servidor recebeu %q, esperado %q (saudacao vazou para o proxy?)", buf, "ola daemon")
	}

	if _, err := fmt.Fprint(serverSide, "ola ponte"); err != nil {
		t.Fatalf("escrevendo no servidor: %v", err)
	}
	buf2 := make([]byte, len("ola ponte"))
	if _, err := io.ReadFull(conn, buf2); err != nil {
		t.Fatalf("lendo no cliente: %v", err)
	}
	if string(buf2) != "ola ponte" {
		t.Fatalf("cliente recebeu %q, esperado %q", buf2, "ola ponte")
	}
}

func TestDialAndHandshakeVersaoDiferente(t *testing.T) {
	vault := t.TempDir()

	ln, _, err := ipc.Listen(vault)
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		// Uma saudacao de uma versao que a ponte desta arvore nao fala --
		// simula um daemon de outra versao, sem depender de a Task 92
		// existir.
		_, _ = fmt.Fprint(c, "GOBSIDIAN-IPC 999\n")
		// Mantem a conexao aberta o suficiente para o cliente ler a linha
		// antes de qualquer FIN chegar primeiro.
		time.Sleep(100 * time.Millisecond)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), boundedWait)
	defer cancel()

	conn, err := ipc.DialAndHandshake(ctx, vault, boundedWait)
	if err == nil {
		_ = conn.Close()
		t.Fatal("DialAndHandshake() error = nil, esperado ErrVersionMismatch")
	}
	if !errors.Is(err, ipc.ErrVersionMismatch) {
		t.Fatalf("DialAndHandshake() error = %v, esperado embrulhar ipc.ErrVersionMismatch", err)
	}
}

func TestDialAndHandshakeRespeitaContext(t *testing.T) {
	vault := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // ja cancelado antes de discar

	start := time.Now()
	conn, err := ipc.DialAndHandshake(ctx, vault, boundedWait)
	elapsed := time.Since(start)

	if err == nil {
		_ = conn.Close()
		t.Fatal("DialAndHandshake() error = nil, esperado erro de context ja cancelado")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("DialAndHandshake() error = %v, esperado embrulhar context.Canceled", err)
	}
	if elapsed > boundedWait {
		t.Fatalf("DialAndHandshake com context ja cancelado demorou %s, esperado retorno imediato", elapsed)
	}
}

// TestListenRestringePermissaoUnix prova a garantia 4 da Task 91 em Unix:
// 0600, so o dono le e escreve. Em Windows a garantia vem da ACL herdada do
// diretorio (ipc_windows.go), nao de chmod -- ver
// TestListenSocketEmDiretorioDoPerfilNoWindows para o que da para provar la.
func TestListenRestringePermissaoUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permissao unix (0600) nao se aplica no Windows -- ver TestListenSocketEmDiretorioDoPerfilNoWindows")
	}

	vault := t.TempDir()
	ln, path, err := ipc.Listen(vault)
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("socket %s tem permissao %04o, esperado 0600", path, mode)
	}
}

// TestListenSocketEmDiretorioDoPerfilNoWindows verifica o que da para provar
// sem privilegio administrativo neste ambiente: o socket fica dentro do
// perfil do usuario corrente (%LocalAppData%), cuja ACL padrao do Windows ja
// nega acesso a outros usuarios locais. Isto NAO E o mesmo que abrir uma
// segunda conta de usuario e tentar conectar -- essa prova exigiria criar
// uma conta local, o que requer privilegio administrativo que este ambiente
// de teste nao concede. Ver o relatorio da Task 91 para o registro explicito
// dessa lacuna.
func TestListenSocketEmDiretorioDoPerfilNoWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("especifico de Windows -- ver TestListenRestringePermissaoUnix para a garantia equivalente em Unix")
	}

	vault := t.TempDir()
	ln, path, err := ipc.Listen(vault)
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	base, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("UserCacheDir() error = %v", err)
	}
	if !strings.HasPrefix(path, base) {
		t.Fatalf("socket %s nao esta dentro do perfil do usuario %s", path, base)
	}
}
