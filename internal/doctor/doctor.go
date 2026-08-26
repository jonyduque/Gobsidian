// Package doctor verifica o ambiente antes que o resto do produto tente
// funcionar nele. E o primeiro comando a rodar quando algo nao funciona.
package doctor

import (
	"context"

	"github.com/jonyd/gobsidian/internal/config"
)

// Status e a gravidade de uma verificacao. A ordem importa: so StatusFail
// altera o codigo de saida, e so ele interrompe as verificacoes seguintes
// quando marcado como halting.
type Status int

// Os tres status que uma verificacao pode produzir. StatusWarn existe para o
// caso comum de um ambiente que funciona de forma incompleta — cofre sem
// .obsidian, arquivo ainda na nuvem — e que nao deve reprovar um setup.
const (
	StatusOK Status = iota
	StatusWarn
	StatusFail
)

// Marker devolve um marcador DISTINTO por status. Falha e aviso precisam ser
// distinguiveis a olho: o relatorio impresso e a unica coisa que a pessoa le,
// e ela roda este comando justamente porque ja esta confusa. Devolver o mesmo
// marcador para os dois apaga a informacao que o comando existe para dar.
func (s Status) Marker() string {
	switch s {
	case StatusOK:
		return "[OK]"
	case StatusWarn:
		return "[*]"
	default:
		return "[!]"
	}
}

// Result e uma linha do relatorio. Detail e opcional e carrega o numero ou o
// caminho que torna o resultado acionavel; sem ele, "permissao de escrita
// [!]" nao diz a ninguem o que fazer em seguida.
type Result struct {
	Name   string
	Status Status
	Detail string
}

type check func(context.Context, config.Config) Result

// Run executa todas as verificacoes na ordem em que importam: as que
// invalidam as seguintes vem primeiro.
func Run(ctx context.Context, cfg config.Config) []Result {
	// halting marca as verificacoes cuja falha invalida tudo o que vem depois.
	// Sem raiz acessivel ou legivel, as seguintes reportariam falhas derivadas
	// e o relatorio viraria ruido — que e o oposto do que se quer de um
	// diagnostico. Comparar pelo nome de uma unica verificacao nao basta: uma
	// raiz que existe mas nao pode ser lida produz exatamente a mesma cascata.
	type entry struct {
		fn      check
		halting bool
	}

	checks := []entry{
		{checkRootExists, true},
		{checkReadable, true},
	}

	out := make([]Result, 0, 16)
	for _, c := range checks {
		res := c.fn(ctx, cfg)
		out = append(out, res)
		if res.Status == StatusFail && c.halting {
			return out
		}
	}

	// A esta altura a raiz existe e pode ser listada: seguro varrer o cofre.
	// A varredura roda uma unica vez aqui, e os checks que precisam de dados
	// por arquivo leem do resultado em vez de abrir o cofre de novo cada um —
	// sem isso, contagem de notas, comprimento de caminho, arquivos
	// somente-nuvem e colisoes de casing fariam quatro (cinco, com o registro
	// de caminhos longos do Windows) varreduras completas do mesmo cofre.
	scan := scanVault(ctx, cfg)

	out = append(out,
		checkWritable(ctx, cfg),
		checkObsidianDir(ctx, cfg),
		checkNoteCount(scan),
		checkLongestPath(scan),
		checkCacheDir(ctx, cfg),
		checkFreeSpace(ctx, cfg),
		// Runtime do daemon (daemon.go). Vem depois das checagens de cofre
		// porque so fazem sentido com a raiz ja validada, e antes das de
		// plataforma porque sao as que alguem consulta quando o servidor
		// "sumiu do host" -- o sintoma que trouxe estas linhas para ca.
		checkSocketPath(ctx, cfg),
		checkDaemonVivo(ctx, cfg),
		checkDaemonLog(ctx, cfg),
		checkLocksOrfaos(ctx, cfg),
	)
	out = append(out, platformChecks(scan)...)

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
