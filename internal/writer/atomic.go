package writer

import (
	"context"

	"github.com/jonyd/gobsidian/internal/vault"
)

// A escrita atomica mora em internal/vault desde a Task 171: o temporario, o
// rename com retry e a varredura de temporarios orfaos sao I/O de arquivo do
// cofre, e vault e a unica camada que toca o sistema de arquivos. O que sobra
// aqui sao encaminhadores, para que a migracao dos chamadores seja uma PR
// separada da mudanca de estrutura.
//
// Os quatro somem na Task 172, quando o ultimo chamador tiver migrado.
//
// Nenhum deles leva o marcador `// Deprecated:`, e isso e decisao registrada, nao
// esquecimento: `staticcheck` esta ligado em `.golangci.yml`, entao o marcador
// faria SA1019 reprovar o gate em todo chamador que esta tarefa deliberadamente
// NAO migra — e um gate vermelho por desenho e um gate que alguem contorna. O
// que marca a transitoriedade e a prosa abaixo, mais o ledger da Task 172.

// TempFilePrefix e o prefixo dos temporarios de escrita atomica.
//
// Encaminhador transitorio para vault.TempFilePrefix; a Task 172 migra os
// chamadores e o remove.
const TempFilePrefix = vault.TempFilePrefix

// SweepResult e o que a varredura de temporarios encontrou.
//
// Encaminhador transitorio para vault.SweepResult; a Task 172 migra os
// chamadores e o remove.
type SweepResult = vault.SweepResult

// WriteAtomic escreve os dados no caminho de destino de forma atomica.
//
// Encaminhador transitorio para vault.WriteAtomic; a Task 172 migra os
// chamadores e o remove.
func WriteAtomic(ctx context.Context, targetPath string, data []byte) error {
	return vault.WriteAtomic(ctx, targetPath, data)
}

// SweepStaleTempFiles remove do cofre os temporarios que escritas interrompidas
// deixaram.
//
// Encaminhador transitorio para vault.SweepStaleTempFiles; a Task 172 migra os
// chamadores e o remove.
func SweepStaleTempFiles(ctx context.Context, root string) (SweepResult, error) {
	return vault.SweepStaleTempFiles(ctx, root)
}
