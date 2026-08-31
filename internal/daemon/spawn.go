// spawn.go lanca o processo do daemon de verdade -- a implementacao real de
// "iniciar" que EnsureStarted (lock.go) chama quando esta ponte vence a
// corrida. Onde e COMO destacar o processo do pai varia por plataforma (ver
// spawn_windows.go / spawn_unix.go); o resto -- que argumentos passar -- e
// comum aos tres sistemas, por isso mora num arquivo sem build tag.

package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/jonyd/gobsidian/internal/config"
)

// DefaultIdleSeconds e o padrao de producao: 15 minutos sem nenhuma ponte
// conectada (decisao 3 da Task 92). O gate de orfaos usa um valor bem
// menor via --idle-seconds explicito, para nao levar 15 minutos por
// cenario (ver scripts/test_orphans.ps1) -- mas o padrao aqui tem de
// continuar sendo este numero: um padrao de teste vazado para producao
// mataria o daemon no meio do uso real.
const DefaultIdleSeconds = 15 * 60

// SpawnDetached lanca `gobsidian daemon` para cfg, destacado deste
// processo -- ele tem de sobreviver a ponte que o iniciou, que sai logo em
// seguida (cmd/gobsidian/ponte.go). logLevel vazio omite a flag e deixa o
// subcomando `daemon` resolver o proprio default.
func SpawnDetached(cfg config.Config, idleSeconds int, logLevel string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolvendo caminho do proprio executavel: %w", err)
	}

	args := ArgsDoDaemon(cfg, idleSeconds, logLevel)

	cmd := exec.Command(exe, args...) // #nosec G204 -- exe e o proprio binario, args sao construidos aqui, nao vem de entrada externa
	applyDetachAttrs(cmd)
	// Stdin/Stdout/Stderr ficam nil de proposito: o daemon nao herda nada
	// do stdio desta ponte, que carrega o JSON-RPC da sessao MCP em curso.
	// os/exec conecta um Reader/Writer nil ao dispositivo nulo.

	return cmd.Start()
}

// ArgsDoDaemon monta a linha de comando do processo do daemon.
//
// Extraida de SpawnDetached para ser TESTAVEL: o defeito que ela carrega e
// sempre uma flag que a ponte recebeu e nao encaminhou — --eager-search ja
// tinha sido esquecida assim, e --max-results estava esquecida ate 2026-08-28
// (achado M9). Um teste que so observa o processo lancado nao consegue dizer
// QUAL argumento faltou.
func ArgsDoDaemon(cfg config.Config, idleSeconds int, logLevel string) []string {
	args := []string{
		"daemon",
		"--vault", cfg.VaultPath,
		"--cache-dir", cfg.CacheDir,
		"--debounce-ms", strconv.Itoa(cfg.DebounceMS),
		"--idle-seconds", strconv.Itoa(idleSeconds),
	}
	if cfg.ReadOnly {
		args = append(args, "--read-only")
	}
	if cfg.EagerSearch {
		// Sem isto, uma ponte iniciada com --eager-search que acaba
		// lancando o daemon perderia a flag em silencio: o daemon subiria
		// no modo padrao (carga preguicosa), divergindo do que quem
		// configurou a ponte pediu -- a mesma classe de defeito que
		// CLAUDE.md registra para ReadOnlySet/DebounceMSSet.
		args = append(args, "--eager-search")
	}
	if cfg.MaxResults > 0 {
		// Sem isto a flag --max-results da ponte era no-op silencioso no modo
		// daemon: o daemon subia com o padrao e a ponte que a pediu era
		// atendida com outro teto (achado M9). Mesma classe de
		// ReadOnlySet/DebounceMSSet, e mesma correcao que --eager-search ja
		// tinha recebido acima.
		args = append(args, "--max-results", strconv.Itoa(cfg.MaxResults))
	}
	if logLevel != "" {
		args = append(args, "--log-level", logLevel)
	}

	return args
}
