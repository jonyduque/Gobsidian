// Package ipc implementa o transporte local entre a ponte (cmd/gobsidian) e
// um daemon de cofre (internal/daemon, Task 92). D-M7-6 fechou o transporte
// em AF_UNIX nos tres sistemas, sem compilacao condicional para o socket em
// si — medido em 2026-08-05, ida e volta de 25,7 us contra 82,9 us de named
// pipe em 256 B. So o CAMINHO do socket e a limpeza dele mudam por
// plataforma (ver ipc_unix.go e ipc_windows.go).
//
// A garantia de rede que este pacote respeita e o RNF-30 reformulado pela
// Task 90: nenhum socket que saia da maquina. tools/netcheck reprova
// qualquer net.Dial/net.Listen cuja rede nao seja a constante literal
// "unix" — e por isso as duas chamadas abaixo escrevem "unix" direto, nunca
// atras de uma variavel ou constante intermediaria.
package ipc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
)

// ProtocolVersion identifica o formato do handshake entre ponte e daemon.
// Task 92, decisao 4: ponte e daemon de versoes diferentes nao se falam —
// a ponte cai para o modo em processo em vez de arriscar um protocolo que
// os dois lados entendem diferente.
const ProtocolVersion = "1"

// greetingPrefix comeca a unica linha que este pacote interpreta. Depois
// dela a conexao vira bytes crus: quem fala JSON-RPC de verdade e o lado
// que a recebe, nunca a ponte.
const greetingPrefix = "GOBSIDIAN-IPC "

// greetingMaxLen limita quanto a leitura da saudacao espera antes de desistir.
// Uma saudacao real tem menos de 20 bytes; o teto so existe para um daemon
// hostil ou corrompido nao prender a leitura para sempre num socket que
// nunca manda '\n'.
const greetingMaxLen = 128

// ErrVersionMismatch e a causa especifica de handshake recusado por versao.
// Exportado para quem quiser distinguir esse caso (por exemplo, para
// decidir se vale a pena reportar de outro jeito) via errors.Is.
var ErrVersionMismatch = errors.New("versao do protocolo ipc incompativel")

// ErrConfigMismatch e a causa especifica de handshake recusado por
// configuracao divergente -- versao bate, mas ReadOnly ou VaultKey nao
// (ver HandshakeConfig). Duas pontes do mesmo cofre com configuracao
// diferente conectadas ao mesmo daemon e bug de seguranca, nao detalhe: uma
// ponte iniciada com --read-only receberia uma sessao que escreve no cofre,
// e quem a configurou nao teria como saber (Task 92, perigo 1). A resposta
// certa nao e negociar -- e recusar o daemon e cair para o modo em
// processo, que so pode servir a configuracao de quem o esta rodando.
var ErrConfigMismatch = errors.New("configuracao do daemon diverge da ponte")

// Conn e o subconjunto de *net.UnixConn que a ponte precisa: ler, escrever,
// fechar dos dois lados, e meio-fechar so a metade de escrita quando o
// stdin do host chega ao fim mas o daemon ainda pode ter algo para
// responder. Expor esta interface em vez de net.Conn mantem o pacote "net"
// fora de cmd/gobsidian, do mesmo jeito que internal/mcpsrv e o unico lugar
// que fala tipos do SDK MCP.
type Conn interface {
	io.Reader
	io.Writer
	io.Closer
	CloseWrite() error
}

// SocketPath deriva o caminho do socket Unix do cofre a partir da MESMA
// chave que config.defaultCacheDir usa para o diretorio de cache —
// config.VaultKey e a unica funcao que calcula essa chave, e as duas
// concordam de proposito. Um cofre, um socket; cofres diferentes nunca
// colidem.
func SocketPath(vaultPath string) (string, error) {
	dir, err := runtimeDir()
	if err != nil {
		return "", fmt.Errorf("diretorio de runtime do usuario: %w", err)
	}
	return filepath.Join(dir, config.VaultKey(vaultPath)+".sock"), nil
}

// Listen abre o socket Unix do cofre. Cria o diretorio de runtime se
// preciso, limpa um arquivo orfao de uma partida anterior que morreu sem
// fechar o listener, e restringe a permissao do arquivo (decisao 4 da Task
// 91: 0600 em Unix via restrictPermission; no Windows a garantia vem da ACL
// herdada do diretorio, e restrictPermission ali e no-op documentado).
//
// Quem chama esta funcao hoje sao os testes deste pacote — o daemon de
// verdade e a Task 92 — mas o contrato de permissao vale desde ja, porque e
// ele que a Task 91 tem de provar.
func Listen(vaultPath string) (net.Listener, string, error) {
	path, err := SocketPath(vaultPath)
	if err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, "", fmt.Errorf("criando diretorio do socket: %w", err)
	}
	if err := cleanupSocketFile(path); err != nil {
		return nil, "", fmt.Errorf("limpando socket anterior: %w", err)
	}

	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, "", fmt.Errorf("abrindo socket %s: %w", path, err)
	}
	if err := restrictPermission(path); err != nil {
		_ = ln.Close()
		_ = cleanupSocketFile(path)
		return nil, "", fmt.Errorf("restringindo permissao do socket: %w", err)
	}
	return ln, path, nil
}

// HandshakeConfig e o que o handshake confere ALEM da versao: os campos que
// mudam comportamento observavel entre pontes diferentes conectadas ao
// mesmo daemon. Hoje sao dois -- ReadOnly (ver ErrConfigMismatch) e
// VaultKey, a mesma chave que nomeia o socket (config.VaultKey), conferida
// de novo aqui como segunda camada: se algum dia as duas contas dessa chave
// divergirem (a licao do byAlias em CLAUDE.md), o handshake pega o que o
// nome do arquivo do socket sozinho nao pegaria.
type HandshakeConfig struct {
	ReadOnly bool
	VaultKey string
}

// Greet escreve a saudacao de versao e configuracao que readGreeting espera
// do outro lado. Quem aceita a conexao (o daemon, Task 92; os testes deste
// pacote hoje) chama isto uma vez, antes de entregar a conexao ao proxy —
// e a UNICA linha que atravessa o socket que este pacote interpreta; tudo
// depois dela e byte cru.
func Greet(w io.Writer, cfg HandshakeConfig) error {
	ro := 0
	if cfg.ReadOnly {
		ro = 1
	}
	_, err := fmt.Fprintf(w, "%s%s ro=%d vault=%s\n", greetingPrefix, ProtocolVersion, ro, cfg.VaultKey)
	return err
}

// DialAndHandshake conecta ao socket do cofre e confere a versao do
// protocolo antes de devolver a conexao. Qualquer falha — socket ausente,
// conexao recusada, prazo excedido, ou versao incompativel — devolve um
// erro nao-nil; quem chama decide o fallback (ver servePonte em
// cmd/gobsidian/ponte.go, que cai para o modo em processo nos tres casos).
func DialAndHandshake(ctx context.Context, vaultPath string, readOnly bool, timeout time.Duration) (Conn, error) {
	path, err := SocketPath(vaultPath)
	if err != nil {
		return nil, err
	}

	dctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	raw, err := dialUnix(dctx, path)
	if err != nil {
		return nil, fmt.Errorf("conectando ao socket %s: %w", path, err)
	}

	if err := raw.SetDeadline(time.Now().Add(timeout)); err != nil {
		_ = raw.Close()
		return nil, fmt.Errorf("configurando prazo do handshake: %w", err)
	}
	want := HandshakeConfig{ReadOnly: readOnly, VaultKey: config.VaultKey(vaultPath)}
	if err := readGreeting(raw, want); err != nil {
		_ = raw.Close()
		return nil, err
	}
	if err := raw.SetDeadline(time.Time{}); err != nil {
		_ = raw.Close()
		return nil, fmt.Errorf("limpando prazo apos handshake: %w", err)
	}

	conn, ok := raw.(Conn)
	if !ok {
		_ = raw.Close()
		return nil, fmt.Errorf("conexao unix sem suporte a meio-fechamento (CloseWrite)")
	}
	return conn, nil
}

// dialUnix isola a unica chamada de net.Dial deste pacote, com a rede como
// constante literal "unix" — e o que tools/netcheck exige (ver o comentario
// do pacote). Roda em goroutine separada e devolve pelo context porque
// net.Dial nao aceita um; se o context vencer primeiro, uma segunda
// goroutine fecha a conexao que a primeira eventualmente trouxer, para nao
// vazar um file descriptor que ninguem mais vai usar.
func dialUnix(ctx context.Context, path string) (net.Conn, error) {
	type result struct {
		conn net.Conn
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		conn, err := net.Dial("unix", path)
		ch <- result{conn, err}
	}()

	select {
	case r := <-ch:
		return r.conn, r.err
	case <-ctx.Done():
		go func() {
			if r := <-ch; r.conn != nil {
				_ = r.conn.Close()
			}
		}()
		return nil, ctx.Err()
	}
}

// readGreeting le e valida a linha de saudacao do outro lado: primeiro a
// versao (recusa imediata e independente do resto se ela nao bater — um
// daemon de outra versao pode nem falar o resto deste formato), depois a
// configuracao (ErrConfigMismatch; ver o comentario de HandshakeConfig).
func readGreeting(r io.Reader, want HandshakeConfig) error {
	line, err := readLine(r, greetingMaxLen)
	if err != nil {
		return fmt.Errorf("lendo saudacao do daemon: %w", err)
	}
	if !strings.HasPrefix(line, greetingPrefix) {
		return fmt.Errorf("saudacao do daemon invalida: %q", line)
	}

	fields := strings.Fields(strings.TrimPrefix(line, greetingPrefix))
	if len(fields) == 0 {
		return fmt.Errorf("saudacao do daemon vazia")
	}

	version := fields[0]
	if version != ProtocolVersion {
		return fmt.Errorf("%w: ponte=%s daemon=%s", ErrVersionMismatch, ProtocolVersion, version)
	}

	got, err := parseHandshakeFields(fields[1:])
	if err != nil {
		return fmt.Errorf("saudacao do daemon malformada: %w", err)
	}
	if got != want {
		return fmt.Errorf("%w: ponte quer %+v, daemon oferece %+v", ErrConfigMismatch, want, got)
	}
	return nil
}

// parseHandshakeFields le os campos "chave=valor" depois da versao. Nao
// exige ordem nem presenca de todos os campos -- um campo ausente fica no
// zero-valor de HandshakeConfig, o que so importa se `want` esperar algo
// diferente, e ai o comparador em readGreeting recusa do jeito certo.
func parseHandshakeFields(fields []string) (HandshakeConfig, error) {
	var cfg HandshakeConfig
	for _, f := range fields {
		key, value, ok := strings.Cut(f, "=")
		if !ok {
			return HandshakeConfig{}, fmt.Errorf("campo sem '=': %q", f)
		}
		switch key {
		case "ro":
			switch value {
			case "1":
				cfg.ReadOnly = true
			case "0":
				cfg.ReadOnly = false
			default:
				return HandshakeConfig{}, fmt.Errorf("valor invalido para ro: %q", value)
			}
		case "vault":
			cfg.VaultKey = value
		default:
			return HandshakeConfig{}, fmt.Errorf("campo desconhecido: %q", key)
		}
	}
	return cfg, nil
}

// readLine le um byte de cada vez ate '\n' ou maxLen, sem usar bufio.Reader.
// Um bufio.Reader dá um Read em blocos maiores que a linha, e os bytes que
// sobrarem no buffer interno dele ficariam presos ali — perdidos para quem
// le direto de r depois, que e exatamente o que a ponte faz assim que o
// handshake termina. A ponte e burra (nao interpreta o que copia); nao pode
// perder um byte so por causa de como leu os primeiros vinte.
func readLine(r io.Reader, maxLen int) (string, error) {
	var b strings.Builder
	buf := make([]byte, 1)
	for range maxLen {
		n, err := r.Read(buf)
		if n == 1 {
			if buf[0] == '\n' {
				return b.String(), nil
			}
			b.WriteByte(buf[0])
		}
		if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("linha de saudacao excede %d bytes", maxLen)
}

// cleanupSocketFile remove um arquivo de socket orfao de uma partida
// anterior que morreu sem fechar o listener. Ausente nao e erro. E
// independente de plataforma — os.Remove se comporta igual nas tres — por
// isso mora aqui e nao em ipc_unix.go/ipc_windows.go, que so tem o que
// varia de verdade (runtimeDir, restrictPermission).
func cleanupSocketFile(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
