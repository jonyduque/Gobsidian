package writer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TempFilePrefix e o prefixo usado para todos os arquivos temporarios de escrita atomica.
const TempFilePrefix = ".gobsidian-tmp-"

// CleanStaleTempFiles remove os temporarios de escritas interrompidas em UM
// diretorio. Recuperacao de crash, e so isso: uma escrita que falha por
// qualquer motivo normal ja remove o proprio temporario no defer de
// WriteAtomic. O unico caso que sobra e o processo morto, que nao roda defer.
//
// NAO CHAME ISTO NO INICIO DE UMA ESCRITA. Ate 2026-07-30 WriteAtomic a chamava
// ali, e o glob apaga TODOS os temporarios do diretorio — inclusive um que
// outra escrita em voo esta usando. A trava do writer e por CAMINHO, de
// proposito: duas notas na mesma pasta escrevem em paralelo. O recurso
// compartilhado aqui e o DIRETORIO, e a trava por caminho nao o cobre.
//
// No Windows a corrida ficava mascarada: os.Remove sobre arquivo com handle
// aberto falha com sharing violation, e o erro era engolido. Em Linux e macOS o
// unlink sucede por semantica POSIX — a outra escrita segue gravando num inode
// desvinculado, Sync e Close passam, e o rename falha com ENOENT. Escrita
// perdida, com erro cuja causa nao tem relacao com o que o chamador pediu.
//
// O lugar certo e o boot, onde nao ha escrita em voo: SweepStaleTempFiles.
func CleanStaleTempFiles(dir string) {
	matches, err := filepath.Glob(filepath.Join(dir, TempFilePrefix+"*"))
	if err == nil {
		for _, m := range matches {
			_ = os.Remove(m)
		}
	}
}

// SweepResult e o que a varredura de temporarios encontrou.
//
// Era um int so — a contagem de removidos — e todo o resto era descartado
// (achado P11). Uma varredura que nao conseguiu entrar em nenhum diretorio
// devolvia "0 removidos", indistinguivel de um cofre limpo. Quem chama loga
// os tres numeros.
type SweepResult struct {
	Removidos int
	// NaoRemovidos conta temporarios encontrados que os.Remove recusou —
	// tipicamente arquivo travado por outro processo.
	NaoRemovidos int
	// Inacessiveis conta subarvores em que a varredura nao conseguiu entrar.
	Inacessiveis int
}

// SweepStaleTempFiles remove, do cofre inteiro, os temporarios que escritas
// interrompidas deixaram, e devolve quantos removeu. Roda no boot, quando
// nenhuma escrita esta em voo — e por isso nao tem a corrida que CleanStaleTempFiles
// tem quando chamada durante uma escrita.
//
// Existe tambem porque a varredura preguicosa nao bastava: um temporario orfao
// numa pasta que nunca mais fosse escrita ficava no cofre do usuario para
// sempre. O filtro de ruido do vault o esconde do indice, entao o usuario o via
// so no Explorer, sem saber de onde veio.
//
// Recebe ctx porque percorrer um cofre grande bloqueia. Erro em subdiretorio
// nao aborta a varredura: um temporario que nao pudemos remover e lixo, nao
// motivo para o servidor nao subir.
func SweepStaleTempFiles(ctx context.Context, root string) (SweepResult, error) {
	var res SweepResult

	// A raiz vai CRUA, e isso esta medido.
	//
	// O achado P11 dizia que a varredura pulava diretorios alem de MAX_PATH em
	// silencio, e que a raiz precisava do prefixo \?\ para os filhos herdarem.
	// Sondado em 2026-08-27: falso. O pacote os do Go aplica o prefixo sozinho
	// (fixLongPath), e MkdirAll, WriteFile e WalkDir alcancaram 318 caracteres
	// sem prefixo nenhum. A prova de mutacao confirmou: trocar por vault.LongPath
	// — que para raiz curta e identidade, ou seja, o comportamento antigo —
	// deixou o teste PASSANDO.
	//
	// Prefixar aqui seria guarda que nao muda resultado: parece protecao e nao e.
	// TestSweepAlcancaCaminhoAlemDeMaxPath fixa o alcance real.
	err := filepath.WalkDir(root, func(caminho string, d os.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			// d == nil e falha na propria raiz — cofre desmontado, share caido.
			// Devolver nil ali faria a varredura reportar sucesso com zero
			// removidos, e cofre inacessivel nao pode dar a mesma resposta que
			// cofre limpo.
			if d == nil {
				return fmt.Errorf("varrendo temporarios em %q: %w", root, err)
			}
			// Erro em subarvore continua nao abortando — um temporario que nao
			// pudemos remover e lixo, nao motivo para o servidor nao subir. Mas
			// agora ele é CONTADO: "varri e nao achei nada" e "varri e nao
			// consegui entrar em 30 diretorios" nao podem produzir a mesma
			// resposta.
			res.Inacessiveis++
			return nil
		}
		if d.IsDir() || !strings.HasPrefix(d.Name(), TempFilePrefix) {
			return nil
		}
		if os.Remove(caminho) == nil {
			res.Removidos++
		} else {
			res.NaoRemovidos++
		}
		return nil
	})
	return res, err
}

// WriteAtomic escreve os dados fornecidos no caminho de destino de forma atomica:
// 1. Cria um arquivo temporario no mesmo diretorio do destino.
// 2. Escreve os dados.
// 3. Executa Sync() (fsync) para garantir a gravacao fisica no disco contra quedas de energia.
// 4. Fecha o temporario.
// 5. Executa rename atomico sobre o arquivo de destino, com retry em caso de bloqueio temporario (Windows).
func WriteAtomic(targetPath string, data []byte) error {
	dir := filepath.Dir(targetPath)

	// Nenhuma varredura aqui: ver o comentario de CleanStaleTempFiles. O
	// temporario desta escrita e removido pelo defer abaixo em qualquer falha;
	// o de um processo morto e removido no boot por SweepStaleTempFiles.
	tmpFile, err := os.CreateTemp(dir, TempFilePrefix+"*")
	if err != nil {
		return fmt.Errorf("criando temporario em %q: %w", dir, err)
	}
	tmpName := tmpFile.Name()

	cleanup := true
	defer func() {
		if cleanup {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("escrevendo dados no temporario %q: %w", tmpName, err)
	}

	// Sync e obrigatorio para durabilidade e integridade fisica contra queda de energia.
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync do temporario %q: %w", tmpName, err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("fechando temporario %q: %w", tmpName, err)
	}

	maxRetries := 10
	retryDelay := 10 * time.Millisecond
	var renameErr error

	for i := 0; i < maxRetries; i++ {
		renameErr = os.Rename(tmpName, targetPath)
		if renameErr == nil {
			cleanup = false
			return nil
		}
		time.Sleep(retryDelay)
	}

	return fmt.Errorf("falha ao renomear %q para %q apos %d tentativas: %w", tmpName, targetPath, maxRetries, renameErr)
}
