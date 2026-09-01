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
	"net"
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
	if conn, err := ipc.DialAndHandshake(ctx, cfg.VaultPath, cfg.ReadOnly, cfg.MaxResults, dialProbeTimeout); err == nil {
		return conn.Close()
	}

	if err := iniciar(); err != nil {
		return fmt.Errorf("iniciando processo do daemon: %w", err)
	}
	return esperarSocket(ctx, cfg, prazo)
}

// adquirirLock tenta se tornar quem inicia o daemon deste cofre.
//
// O caminho deriva de ipc.SocketPath -- a MESMA chave que nomeia o socket,
// nunca uma segunda conta do mesmo hash (ver config.VaultKey sobre a licao do
// byAlias: duas contas do mesmo valor concordam por coincidencia ate que uma
// delas mude sozinha).
//
// Nao ha volta de repeticao nem recuperacao de lock obsoleto: com a trava do
// kernel, o lock de um processo morto ja esta livre quando o proximo pergunta.
func adquirirLock(vaultPath string) (adquiriu bool, liberar func(), err error) {
	path, err := lockPath(vaultPath)
	if err != nil {
		return false, nil, err
	}
	trava, tomou, err := tentarTravar(path)
	if err != nil {
		return false, nil, err
	}
	if !tomou {
		return false, nil, nil
	}
	return true, trava.Liberar, nil
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
		conn, err := ipc.DialAndHandshake(ctx, cfg.VaultPath, cfg.ReadOnly, cfg.MaxResults, dialProbeTimeout)
		if err == nil {
			return conn.Close()
		}
		if errors.Is(err, ipc.ErrVersionMismatch) || errors.Is(err, ipc.ErrConfigMismatch) {
			return err
		}
		ultimoErr = err

		if time.Now().After(limite) {
			// A pista do log entra no erro porque a mensagem sozinha culpa o
			// socket -- o unico lugar onde a resposta NAO esta. Medido em
			// 2026-08-26: tres sessoes MCP pagaram este prazo em toda partida
			// durante dias, e quem lia "socket nao respondeu" nao tinha como
			// saber se o daemon morreu na montagem (log com uma linha) ou nem
			// chegou a nascer (log ausente). Sao dois defeitos diferentes com
			// o mesmo sintoma aqui.
			return fmt.Errorf("socket do daemon nao respondeu em %s: %w (%s)",
				prazo, ultimoErr, pistaDoLog(cfg.VaultPath))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// ComLockDeEscuta serializa a sequencia "ninguem escuta? entao escuto eu" de
// DOIS processos de daemon do mesmo cofre.
//
// # Por que um segundo lock, e nao o de EnsureStarted
//
// O lock de EnsureStarted responde "quem LANCA o daemon". Este responde "quem
// ABRE o socket". Sao perguntas diferentes, e usar o mesmo arquivo para as duas
// trava uma na outra: EnsureStarted segura o lock ate esperarSocket devolver, e
// esperarSocket espera o socket que o daemon recem-lancado so pode abrir depois
// de adquirir o lock. Deadlock, com o prazo inteiro do EnsureStarted de sintoma.
//
// # O que ele fecha
//
// ipc.Listen prova que o socket esta orfao antes de desvincula-lo (Task 133), o
// que impede uma ponte de roubar o socket de um daemon vivo. Mas a sonda e o
// Listen NAO sao atomicos entre si: dois daemons lancados no mesmo instante
// podem ambos sondar "ninguem escuta" antes de qualquer um dos dois bindar.
// Segurar este lock durante o par fecha essa janela — o segundo entra depois do
// primeiro ter bindado, a sonda dele encontra o ouvinte, e ele recusa subir.
//
// Quem nao consegue o lock NAO espera: outro daemon do mesmo cofre esta subindo
// agora, e o certo e sair, nao competir. O erro devolvido diz isso.
func ComLockDeEscuta(vaultPath string, fn func() error) error {
	sock, err := ipc.SocketPath(vaultPath)
	if err != nil {
		return err
	}
	// Deriva do MESMO caminho do socket, nunca de uma segunda conta do hash do
	// cofre — a licao do byAlias que config.VaultKey registra.
	path := sock + ".listen.lock"

	trava, tomou, err := tentarTravar(path)
	if err != nil {
		return err
	}
	if !tomou {
		return fmt.Errorf("outro daemon deste cofre esta abrindo o socket agora")
	}
	defer trava.Liberar()

	return fn()
}

// EscutarComLock e ipc.Listen sob o lock de escuta.
//
// Existe para que cmd/gobsidian nao precise nomear net.Listener: RNF-30 diz que
// nenhum pacote sob cmd/ ou internal/ importa net fora do IPC local, e um tipo
// de retorno arrasta o import junto. O daemon ja depende deste pacote, e este
// pacote ja fala IPC.
func EscutarComLock(vaultPath string) (net.Listener, string, error) {
	var ln net.Listener
	var sock string
	err := ComLockDeEscuta(vaultPath, func() error {
		var e error
		ln, sock, e = ipc.Listen(vaultPath)
		return e
	})
	return ln, sock, err
}
