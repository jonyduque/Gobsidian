// lock.go resolve a corrida de inicializacao (decisao 2 da Task 92): dez
// pontes conectando ao mesmo cofre ao mesmo tempo nao podem resultar em dez
// daemons. EnsureStarted e chamado por toda ponte antes de tentar o
// handshake de novo -- so uma delas ganha o direito de lancar o processo;
// as outras esperam e conectam.

package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/ipc"
)

// pollInterval e o intervalo entre tentativas de handshake enquanto se
// espera o socket aparecer -- tanto para quem perdeu a corrida quanto para
// quem venceu e acabou de pedir o lancamento do processo.
const pollInterval = 25 * time.Millisecond

// dialProbeTimeout limita cada tentativa individual de handshake dentro de
// esperarSocket. Curto de proposito, do mesmo jeito que ipcDialTimeout em
// cmd/gobsidian/ponte.go: o que estica a espera total e o prazo do
// chamador, nao esta tentativa isolada.
const dialProbeTimeout = 200 * time.Millisecond

// EnsureStarted garante que ha um daemon respondendo no socket de
// cfg.VaultPath, iniciando um se esta chamada ganhar a corrida de
// inicializacao. iniciar e chamado no MAXIMO uma vez por chamada desta
// funcao -- e nunca mais de uma vez entre chamadas concorrentes para o
// MESMO cofre, o que TestDezPontesIniciamUmDaemonSo prova diretamente.
//
// iniciar so precisa PEDIR o lancamento do processo (nao esperar ele ficar
// pronto) -- quem confere prontidao de verdade e esperarSocket, chamada
// depois por QUALQUER chamador, vencedor ou nao, porque o daemon leva um
// instante para abrir o socket seja quem for que o lancou.
func EnsureStarted(ctx context.Context, cfg config.Config, prazo time.Duration, iniciar func() error) error {
	adquiriu, liberar, err := adquirirLock(cfg.VaultPath)
	if err != nil {
		return fmt.Errorf("adquirindo lock de inicializacao do daemon: %w", err)
	}

	if !adquiriu {
		// Perdeu a corrida: outra ponte ja esta iniciando o daemon deste
		// cofre. So espera o socket responder -- nunca inicia um segundo.
		//
		// adquiriu e referenciada de novo aqui dentro, alem da condicao
		// acima, pelo mesmo motivo do comentario em
		// tools/netcheck/netcheck.go: uma mutacao que apague so o teste da
		// condicao (trocando por um "if false" fixo) nao pode deixar a
		// variavel sem uso e quebrar a compilacao -- isso sairia
		// INCONCLUSIVO no mutate.ps1, nao prova nenhuma regra.
		_ = adquiriu
		return esperarSocket(ctx, cfg, prazo)
	}
	defer liberar()

	// Confere de novo, AGORA que o lock foi adquirido, se um daemon ja esta
	// respondendo -- antes de pedir um novo.
	//
	// O lock so serializa quem tenta adquiri-lo NO MESMO INSTANTE; ele nao
	// impede que uma chamada MUITO atrasada pelo agendamento do SO vença
	// uma corrida NOVA depois que um vencedor anterior ja terminou de subir
	// o proprio daemon e liberado o lock (medido: dez pontes lancadas
	// juntas sob carga pesada da maquina produziram dois daemons vivos ao
	// mesmo tempo para o MESMO cofre, com quase um minuto de intervalo
	// entre os dois "daemon iniciado" -- a segunda ponte nunca viu o
	// primeiro daemon porque so chegou aqui muito depois do dial inicial
	// de servePonte). Esta segunda checagem fecha essa janela: se alguem
	// mais ja subiu o daemon enquanto esta chamada esperava a vez, usa
	// esse, e nunca chama iniciar.
	if conn, err := ipc.DialAndHandshake(ctx, cfg.VaultPath, cfg.ReadOnly, dialProbeTimeout); err == nil {
		return conn.Close()
	}

	if err := iniciar(); err != nil {
		return fmt.Errorf("iniciando processo do daemon: %w", err)
	}
	return esperarSocket(ctx, cfg, prazo)
}

// adquirirLock tenta se tornar quem inicia o daemon deste cofre, via
// criacao exclusiva (O_EXCL) de um arquivo. O caminho deriva de
// ipc.SocketPath -- a MESMA chave que nomeia o socket, nunca uma segunda
// conta do mesmo hash (ver o comentario de config.VaultKey sobre a licao
// do byAlias em CLAUDE.md: duas contas do mesmo valor concordam por
// coincidencia ate que uma delas mude sozinha).
func adquirirLock(vaultPath string) (adquiriu bool, liberar func(), err error) {
	path, err := lockPath(vaultPath)
	if err != nil {
		return false, nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, nil, fmt.Errorf("criando diretorio do lock: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("criando lock %s: %w", path, err)
	}
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
	_ = f.Close()

	// liberar remove o arquivo SEMPRE que a tentativa termina, tenha ela
	// dado certo ou nao: um lock que sobrevive a uma tentativa fracassada
	// de iniciar o daemon travaria toda ponte seguinte atras de um arquivo
	// que ninguem mais vai liberar.
	return true, func() { _ = os.Remove(path) }, nil
}

// lockPath deriva do MESMO caminho que ipc.SocketPath calcula, trocando so
// a extensao -- um cofre, um socket, um lock, todos no mesmo diretorio de
// runtime.
func lockPath(vaultPath string) (string, error) {
	sock, err := ipc.SocketPath(vaultPath)
	if err != nil {
		return "", err
	}
	return sock + ".lock", nil
}

// esperarSocket tenta o handshake completo -- versao E configuracao, ver
// internal/ipc.HandshakeConfig -- ate o daemon responder ou o prazo
// esgotar. Um erro de versao ou de configuracao divergente e TERMINAL: o
// socket respondeu, so nao aceita esta ponte, e insistir no mesmo prazo
// nao mudaria isso.
func esperarSocket(ctx context.Context, cfg config.Config, prazo time.Duration) error {
	limite := time.Now().Add(prazo)

	var ultimoErr error
	for {
		conn, err := ipc.DialAndHandshake(ctx, cfg.VaultPath, cfg.ReadOnly, dialProbeTimeout)
		if err == nil {
			return conn.Close()
		}
		if errors.Is(err, ipc.ErrVersionMismatch) || errors.Is(err, ipc.ErrConfigMismatch) {
			return err
		}
		ultimoErr = err

		if time.Now().After(limite) {
			return fmt.Errorf("socket do daemon nao respondeu em %s: %w", prazo, ultimoErr)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}
