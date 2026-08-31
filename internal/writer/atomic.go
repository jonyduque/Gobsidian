package writer

import (
	"context"
	"fmt"
	"log/slog"
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
// WriteAtomic recebe ctx porque ESPERA DE VERDADE: o laco de rename abaixo
// dorme ate 100 ms tentando de novo, e a escrita em si pode bloquear num share
// de rede. A regra desta base e "ctx onde ha espera real" (achado M13).
//
// Cancelar NAO desfaz um rename ja aplicado — a escrita e atomica, nao
// transacional. O ctx e conferido nos pontos em que ainda nao houve efeito
// visivel, e entre as tentativas.
func WriteAtomic(ctx context.Context, targetPath string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir := filepath.Dir(targetPath)

	// Nenhuma varredura aqui: ver o comentario de CleanStaleTempFiles. O
	// temporario desta escrita e removido pelo defer abaixo em qualquer falha;
	// o de um processo morto e removido no boot por SweepStaleTempFiles.
	tmpFile, err := os.CreateTemp(dir, TempFilePrefix+"*")
	if err != nil {
		return fmt.Errorf("criando temporario em %q: %w", dir, err)
	}
	tmpName := tmpFile.Name()

	// os.CreateTemp cria com 0600. Sem restaurar o modo do ALVO, um rename por
	// cima de uma nota 0644 a deixa 0600 — a escrita "preserva o conteudo" e
	// muda a permissao pelas costas (achado M12). Em cofre compartilhado por
	// grupo, a nota some para os outros.
	//
	// Alvo inexistente e nota nova: 0644, o mesmo que o resto do projeto usa.
	// No Windows o modo e quase todo ignorado pelo runtime do Go — so o bit de
	// somente-leitura mapeia —, mas o cofre pode estar num share lido de Linux,
	// e a chamada e barata.
	modo := os.FileMode(0644)
	if info, err := os.Stat(targetPath); err == nil {
		modo = info.Mode().Perm()
	}
	if err := tmpFile.Chmod(modo); err != nil {
		// Nao e fatal. Ha sistema de arquivos que nao suporta Chmod, e recusar
		// a escrita inteira por causa da permissao perderia o conteudo, que e o
		// que o usuario pediu para gravar. Fica registrado.
		slog.Debug("nao foi possivel aplicar o modo do alvo ao temporario",
			"alvo", targetPath, "modo", modo, "err", err)
	}

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
		// Entre tentativas, e nao dentro: uma tentativa ja iniciada termina.
		if err := ctx.Err(); err != nil {
			return err
		}
		renameErr = os.Rename(tmpName, targetPath)
		if renameErr == nil {
			cleanup = false
			// O Sync do arquivo garante os DADOS; o rename e uma mudanca de
			// DIRETORIO, e sem sincronizar o diretorio uma queda de energia
			// logo depois pode deixar o alvo com o conteudo antigo — ou
			// nenhum — apesar de a escrita ter reportado sucesso (achado M12).
			//
			// Falha aqui nao desfaz o rename, que ja aconteceu: e reportada
			// como aviso, nao como erro da escrita.
			if err := sincronizarDiretorio(dir); err != nil {
				slog.Debug("nao foi possivel sincronizar o diretorio apos o rename",
					"dir", dir, "err", err)
			}
			return nil
		}
		// Dormir com select, e nao time.Sleep: um cancelamento durante a
		// espera nao pode ficar 100 ms sem resposta.
		select {
		case <-time.After(retryDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("falha ao renomear %q para %q apos %d tentativas: %w", tmpName, targetPath, maxRetries, renameErr)
}
