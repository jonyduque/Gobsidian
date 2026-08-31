// daemon.go traz as verificações do runtime do daemon para dentro do `doctor`.
//
// Existe por um diagnóstico real. Em 2026-08-26, três sessões MCP da máquina do
// dono caíram para o modo em processo em TODA partida, ao longo de dias,
// pagando dez segundos cada uma — e descobrir por quê custou quatro camadas de
// investigação manual: processos e parentesco, o diretório de runtime, os logs
// do host, e uma reprodução com stderr capturado. As quatro respostas estavam
// disponíveis para um programa; nenhuma estava disponível para quem rodava o
// comando de diagnóstico.
//
// `doctor` é o comando que alguém roda quando JÁ está confuso. Sobre a metade
// do sistema que quebrou, ele não dizia uma palavra.

package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/daemon"
	"github.com/jonyd/gobsidian/internal/ipc"
)

// prazoDeSondaDoDaemon limita o dial de diagnóstico. Curto: D-M7-6 mediu 25,7 us
// de ida e volta num socket Unix local que responde, então qualquer coisa acima
// disto já é "não está servindo".
const prazoDeSondaDoDaemon = 500 * time.Millisecond

// classeDoCaminhoDoSocket descreve o que existe no caminho do socket.
//
// A distinção não é acadêmica, e não se descobre pela mensagem de erro. Medido
// em 2026-08-26 com net.Dial("unix", ...) no Windows:
//
//	10061  ECONNREFUSED  arquivo comum, socket órfão de dono morto à força,
//	                     E caminho inexistente — os três produzem o MESMO erro
//	10022  EINVAL        diretório no lugar do socket
//
// Ou seja: errnos diferentes descrevem o mesmo estado, e o mesmo errno descreve
// estados diferentes. Foi o 10022 que apareceu nos logs de campo do dono, e
// nenhum dos reprodutores conhecidos o explica — esta checagem existe para que
// a próxima ocorrência se explique sozinha, dizendo o que de fato está lá.
//
// Usa Lstat, e nunca abre o caminho: abrir um socket é justamente a operação
// cujo resultado se está tentando diagnosticar.
func classeDoCaminhoDoSocket(path string) string {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "ausente"
		}
		return fmt.Sprintf("inacessivel (%v)", err)
	}
	modo := fi.Mode()
	switch {
	case modo&os.ModeSocket != 0:
		return "socket"
	case modo.IsDir():
		return "DIRETORIO (nenhum daemon consegue usar este caminho)"
	case modo&os.ModeSymlink != 0:
		return "symlink"
	case modo.IsRegular():
		return fmt.Sprintf("arquivo comum de %d bytes (residuo; nenhum daemon escuta aqui)", fi.Size())
	default:
		return fmt.Sprintf("outro (modo=%v)", modo)
	}
}

// checkSocketPath diz onde o socket deveria estar e o que existe lá.
//
// Não recebe ctx: Lstat não é espera real, e ctx que nenhum corpo verifica
// ensina revisor a ignorar ctx.
func checkSocketPath(_ context.Context, cfg config.Config) Result {
	const name = "caminho do socket do daemon"

	path, err := ipc.SocketPath(cfg.VaultPath)
	if err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("nao foi possivel derivar: %v", err)}
	}

	classe := classeDoCaminhoDoSocket(path)
	detalhe := fmt.Sprintf("%s -- %s", path, classe)

	// Ausente e socket sao os dois estados saudaveis: sem daemon ainda, ou com
	// um daemon que abriu o socket. Qualquer outra coisa e residuo que impede o
	// daemon de subir, e e informacao acionavel.
	if classe == "ausente" || classe == "socket" {
		return Result{Name: name, Status: StatusOK, Detail: detalhe}
	}
	return Result{Name: name, Status: StatusWarn, Detail: detalhe}
}

// checkDaemonVivo tenta o handshake, que é o ÚNICO critério de "há daemon
// servindo aqui".
//
// Conectar não basta: um daemon cujo laço de Accept morreu continua com o
// socket bound, e o SO aceita a conexão no backlog sem que ninguém a atenda.
// Por isso a sonda vai até o handshake, e por isso o resultado não é derivado
// do errno — ver o comentário de classeDoCaminhoDoSocket.
//
// Recebe ctx porque há espera real: um dial pode bloquear até o prazo.
func checkDaemonVivo(ctx context.Context, cfg config.Config) Result {
	const name = "daemon respondendo"

	conn, err := ipc.DialAndHandshake(ctx, cfg.VaultPath, cfg.ReadOnly, cfg.MaxResults, prazoDeSondaDoDaemon)
	if err == nil {
		_ = conn.Close()
		return Result{Name: name, Status: StatusOK, Detail: "handshake completo"}
	}

	path, perr := ipc.SocketPath(cfg.VaultPath)
	if perr != nil {
		path = "(caminho indisponivel)"
	}
	// Sem daemon nao e falha: o modo em processo e um caminho suportado, e a
	// ponte cai nele de proposito quando nao ha daemon.
	if classeDoCaminhoDoSocket(path) == "ausente" {
		return Result{
			Name:   name,
			Status: StatusOK,
			Detail: "nenhum daemon rodando (a ponte servira em processo)",
		}
	}
	return Result{
		Name:   name,
		Status: StatusWarn,
		Detail: fmt.Sprintf("arquivo existe mas o handshake falhou: %v", err),
	}
}

// checkDaemonLog mostra o fim do log do daemon.
//
// É onde a causa de morte passou a ser escrita. Antes disso, dois daemons
// morreram deixando a linha "daemon iniciado" e mais nada — e um log com uma
// linha só continua sendo o sintoma que essa checagem torna visível.
func checkDaemonLog(_ context.Context, cfg config.Config) Result {
	const name = "log do daemon"

	sock, err := ipc.SocketPath(cfg.VaultPath)
	if err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("nao foi possivel derivar: %v", err)}
	}
	path := sock + ".log"

	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{Name: name, Status: StatusOK, Detail: "ainda nao existe (nenhum daemon rodou para este cofre)"}
		}
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("%s: %v", path, err)}
	}

	linhas, _ := daemon.UltimasLinhasDoLog(cfg.VaultPath, 3)

	idade := time.Since(fi.ModTime()).Round(time.Minute)
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%d bytes, ultima escrita ha %s)", path, fi.Size(), idade)
	for _, l := range linhas {
		b.WriteString("\n      | ")
		b.WriteString(l)
	}
	return Result{Name: name, Status: StatusOK, Detail: b.String()}
}

// checkLocksOrfaos acusa lock de inicialização cujo dono já morreu.
//
// Medido em 2026-08-26 na máquina do dono: cinco deles, de 15 a 19 de agosto,
// todos com PID morto. A liveness vem de daemon.PIDVivo — a mesma conta que o
// próprio lock usa, nunca uma segunda.
func checkLocksOrfaos(_ context.Context, cfg config.Config) Result {
	const name = "locks de daemon orfaos"

	sock, err := ipc.SocketPath(cfg.VaultPath)
	if err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("nao foi possivel derivar: %v", err)}
	}
	dir := filepath.Dir(sock)

	entradas, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{Name: name, Status: StatusOK, Detail: "diretorio de runtime ainda nao existe"}
		}
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("%s: %v", dir, err)}
	}

	var orfaos []string
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sock.lock") {
			continue
		}
		dados, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(dados)))
		if err != nil {
			orfaos = append(orfaos, fmt.Sprintf("%s (conteudo ilegivel)", e.Name()))
			continue
		}
		if !daemon.PIDVivo(pid) {
			orfaos = append(orfaos, fmt.Sprintf("%s (PID %d morto)", e.Name(), pid))
		}
	}

	if len(orfaos) == 0 {
		return Result{Name: name, Status: StatusOK, Detail: "nenhum"}
	}
	return Result{
		Name:   name,
		Status: StatusWarn,
		Detail: fmt.Sprintf("%d em %s: %s", len(orfaos), dir, strings.Join(orfaos, ", ")),
	}
}

// vizinhosParecidos lista irmãos cujo nome bate ignorando caixa e acento.
//
// É o que transforma "não existe" em "você quis dizer isto": a diferença entre
// `Jurisprudencia` e `Jurisprudência` é invisível numa leitura apressada de
// JSON, e foi exatamente o que custou dois dias.
func vizinhosParecidos(alvo string) []string {
	pai := filepath.Dir(alvo)
	base := semAcentoNemCaixa(filepath.Base(alvo))

	entradas, err := os.ReadDir(pai)
	if err != nil {
		return nil
	}
	var achados []string
	for _, e := range entradas {
		if !e.IsDir() {
			continue
		}
		if semAcentoNemCaixa(e.Name()) == base {
			achados = append(achados, e.Name())
		}
	}
	return achados
}

// semAcentoNemCaixa reduz um nome para comparação frouxa. Cobre a família
// latina que aparece em português, que é onde o defeito ocorreu; não pretende
// ser transliteração geral.
func semAcentoNemCaixa(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'á', 'à', 'â', 'ã', 'ä':
			b.WriteRune('a')
		case 'é', 'è', 'ê', 'ë':
			b.WriteRune('e')
		case 'í', 'ì', 'î', 'ï':
			b.WriteRune('i')
		case 'ó', 'ò', 'ô', 'õ', 'ö':
			b.WriteRune('o')
		case 'ú', 'ù', 'û', 'ü':
			b.WriteRune('u')
		case 'ç':
			b.WriteRune('c')
		case 'ñ':
			b.WriteRune('n')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
