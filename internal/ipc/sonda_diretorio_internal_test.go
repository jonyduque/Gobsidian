package ipc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSondarDiretorioDeSocketsAceitaDiretorioUtilizavel(t *testing.T) {
	// os.MkdirTemp, e nao t.TempDir(): o t.TempDir leva o nome do teste no
	// caminho e passa do limite de um caminho AF_UNIX no Windows -- a sonda
	// reprovaria pelo comprimento, nao pelo diretorio.
	dir, err := os.MkdirTemp("", "gs-")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	if err := SondarDiretorioDeSockets(dir); err != nil {
		t.Fatalf("SondarDiretorioDeSockets(diretorio temporario) = %v, esperado nil", err)
	}
}

// Um arquivo comum no lugar do diretorio: nenhum socket cabe ali. E o caso real
// que da para montar sem o host, e o erro tem de nomear o diretorio.
func TestSondarDiretorioDeSocketsRecusaOndeNaoCabeSocket(t *testing.T) {
	arquivo := filepath.Join(t.TempDir(), "arquivo")
	if err := os.WriteFile(arquivo, nil, 0o600); err != nil {
		t.Fatalf("criando arquivo: %v", err)
	}
	dir := filepath.Join(arquivo, "sub")

	err := SondarDiretorioDeSockets(dir)
	if !errors.Is(err, ErrDiretorioSemSocket) {
		t.Fatalf("SondarDiretorioDeSockets(%s) = %v, esperado ErrDiretorioSemSocket", dir, err)
	}
	if !strings.Contains(err.Error(), dir) {
		t.Fatalf("erro %q nao nomeia o diretorio %s", err, dir)
	}
}

// TestListenNaoLimpaSocketOndeOProprioSocketNaoConecta e a guarda de G3.
//
// O caminho do socket tem um arquivo comum -- AlguemEscuta responde "ninguem" e,
// sem a guarda, Listen chamaria cleanupSocketFile. No contexto do Claude Desktop
// medido em 2026-09-14 essa resposta vale tambem para um daemon VIVO, e so nao
// virou o roubo de socket de 2026-08-26 porque o remove falhava ali. A sonda
// simulada reprova como la; a limpeza nao pode ser chamada.
func TestListenNaoLimpaSocketOndeOProprioSocketNaoConecta(t *testing.T) {
	vault := t.TempDir()
	path, err := SocketPath(vault)
	if err != nil {
		t.Fatalf("SocketPath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("criando diretorio do socket: %v", err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("criando arquivo no caminho do socket: %v", err)
	}

	originalSonda, originalRemover, originalRenomear := sondaDeDiretorio, removerArquivo, renomearArquivo
	t.Cleanup(func() {
		sondaDeDiretorio, removerArquivo, renomearArquivo = originalSonda, originalRemover, originalRenomear
	})
	sondaDeDiretorio = func(dir string) error {
		return fmt.Errorf("%w: %s: simulado como no processo do Claude Desktop", ErrDiretorioSemSocket, dir)
	}
	var tocados []string
	removerArquivo = func(p string) error {
		tocados = append(tocados, "remove "+p)
		return nil
	}
	renomearArquivo = func(de, _ string) error {
		tocados = append(tocados, "rename "+de)
		return nil
	}

	ln, _, err := Listen(vault)
	if ln != nil {
		_ = ln.Close()
	}
	if !errors.Is(err, ErrDiretorioSemSocket) {
		t.Fatalf("Listen() = %v, esperado ErrDiretorioSemSocket", err)
	}
	if len(tocados) > 0 {
		t.Fatalf("Listen limpou o caminho do socket num diretorio onde nem o proprio socket conecta: %v", tocados)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("o arquivo no caminho do socket sumiu: %v", err)
	}
}
