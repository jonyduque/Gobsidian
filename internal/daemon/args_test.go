package daemon_test

import (
	"slices"
	"testing"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/daemon"
)

// TestArgsDoDaemonEncaminhaAsFlags cobre a classe de defeito que esta função
// carrega inteira: uma flag que a ponte recebeu e não encaminhou ao daemon.
//
// `--eager-search` já tinha sido esquecida assim, e `--max-results` estava
// esquecida até 2026-08-28 (achado M9) — a ponte aceitava a flag, o daemon subia
// com o padrão, e o cliente era atendido com um teto que ninguém pediu. É a
// mesma classe de `ReadOnlySet`/`DebounceMSSet`: parâmetro aceito que não faz
// efeito, sem nada dizendo isso.
func TestArgsDoDaemonEncaminhaAsFlags(t *testing.T) {
	cfg := config.Config{
		VaultPath:   `C:\cofre`,
		CacheDir:    `C:\cache`,
		DebounceMS:  250,
		ReadOnly:    true,
		EagerSearch: true,
		MaxResults:  137,
	}
	args := daemon.ArgsDoDaemon(cfg, 900, "debug")

	temPar := func(flag, valor string) bool {
		i := slices.Index(args, flag)
		return i >= 0 && i+1 < len(args) && args[i+1] == valor
	}

	casos := []struct {
		nome string
		ok   bool
	}{
		{"--vault", temPar("--vault", `C:\cofre`)},
		{"--cache-dir", temPar("--cache-dir", `C:\cache`)},
		{"--debounce-ms 250", temPar("--debounce-ms", "250")},
		{"--idle-seconds 900", temPar("--idle-seconds", "900")},
		{"--log-level debug", temPar("--log-level", "debug")},
		{"--max-results 137", temPar("--max-results", "137")},
		{"--read-only", slices.Contains(args, "--read-only")},
		{"--eager-search", slices.Contains(args, "--eager-search")},
	}
	for _, c := range casos {
		if !c.ok {
			t.Errorf("faltou %s\nargs = %v", c.nome, args)
		}
	}
}

// TestArgsDoDaemonOmiteOQueNaoFoiPedido é o contrapeso.
//
// Sem ele, "corrigir" o encaminhamento passando TODAS as flags sempre passaria
// no teste acima — e `--max-results 0` diria ao daemon um teto de zero, que não
// é o mesmo que "use o padrão".
func TestArgsDoDaemonOmiteOQueNaoFoiPedido(t *testing.T) {
	args := daemon.ArgsDoDaemon(config.Config{VaultPath: "/c"}, 60, "")

	for _, naoDeveria := range []string{"--read-only", "--eager-search", "--max-results", "--log-level"} {
		if slices.Contains(args, naoDeveria) {
			t.Errorf("%s foi passado sem ter sido pedido\nargs = %v", naoDeveria, args)
		}
	}
}
