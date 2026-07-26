// Package doctor verifica o ambiente antes que o resto do produto tente
// funcionar nele. E o primeiro comando a rodar quando algo nao funciona.
package doctor

import (
	"context"

	"github.com/jonyd/gobsidian/internal/config"
)

type Status int

const (
	StatusOK Status = iota
	StatusWarn
	StatusFail
)

func (s Status) Marker() string {
	switch s {
	case StatusOK:
		return "[OK]"
	case StatusWarn:
		return "[!]"
	default:
		return "[!]"
	}
}

type Result struct {
	Name   string
	Status Status
	Detail string
}

type check func(context.Context, config.Config) Result

// Run executa todas as verificacoes na ordem em que importam: as que
// invalidam as seguintes vem primeiro.
func Run(ctx context.Context, cfg config.Config) []Result {
	checks := []check{
		checkRootExists,
		checkReadable,
		checkWritable,
		checkObsidianDir,
		checkNoteCount,
		checkLongestPath,
		checkCacheDir,
		checkFreeSpace,
	}
	checks = append(checks, platformChecks()...)

	out := make([]Result, 0, len(checks))
	for _, c := range checks {
		res := c(ctx, cfg)
		out = append(out, res)

		// Sem raiz acessivel, todas as verificacoes seguintes reportariam
		// falhas derivadas e o relatorio viraria ruido.
		if res.Status == StatusFail && res.Name == "raiz do cofre existe" {
			break
		}
	}
	return out
}

// ExitCode e zero quando nao ha falha bloqueante. Avisos nao alteram o codigo:
// um cofre com arquivos somente-nuvem funciona, apenas de forma incompleta.
func ExitCode(results []Result) int {
	for _, r := range results {
		if r.Status == StatusFail {
			return 1
		}
	}
	return 0
}
