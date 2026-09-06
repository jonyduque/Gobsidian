package vault

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
// nenhuma escrita esta em voo — por isso pode varrer o diretorio inteiro
// sem risco: ate 2026-07-30 WriteAtomic varria assim no INICIO de cada
// escrita, e o glob apagava TODOS os temporarios do diretorio, inclusive o
// de outra escrita em voo na mesma pasta (a trava do writer e por CAMINHO,
// nao por diretorio). No Windows a corrida ficava mascarada — os.Remove
// sobre arquivo com handle aberto falha com sharing violation, e o erro era
// engolido; em Linux e macOS o unlink sucede por semantica POSIX, a outra
// escrita segue gravando num inode desvinculado, e o rename final falhava
// com ENOENT.
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

// ReplaceFile substitui o arquivo em targetPath de forma atomica:
// 1. Cria um arquivo temporario no mesmo diretorio do destino.
// 2. Chama escrever(tmp) — quem produz os bytes e o chamador.
// 3. Executa Sync() (fsync) para garantir a gravacao fisica no disco contra quedas de energia.
// 4. Fecha o temporario.
// 5. Executa rename atomico sobre o arquivo de destino, com retry em caso de bloqueio temporario (Windows).
//
// O conteudo chega por callback, e nao como []byte, porque os dois caches
// codificam em STREAMING para o temporario — index/persist.go e
// search/persist.go escrevem num io.Writer. Exigir []byte obrigaria a
// materializar o cache inteiro em memoria so para poder grava-lo; o invertido
// mede 24,95 MiB por gravacao (SaveInvertedCacheReal, B/op da Baseline).
// WriteAtomic e o caso particular em que os bytes ja existem.
//
// ReplaceFile recebe ctx porque ESPERA DE VERDADE: o laco de rename abaixo
// dorme ate 100 ms tentando de novo, e a escrita em si pode bloquear num share
// de rede. A regra desta base e "ctx onde ha espera real" (achado M13).
//
// Cancelar NAO desfaz um rename ja aplicado — a escrita e atomica, nao
// transacional. O ctx e conferido nos pontos em que ainda nao houve efeito
// visivel, e entre as tentativas.
//
// Falha do callback nao toca o alvo: o defer abaixo fecha e remove o
// temporario, e o alvo continua com o conteudo anterior.
func ReplaceFile(ctx context.Context, targetPath string, escrever func(*os.File) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir := filepath.Dir(targetPath)

	// Nenhuma varredura de diretorio aqui -- so no boot, por
	// SweepStaleTempFiles, e pelo motivo no comentario dela. O temporario
	// desta escrita e removido pelo defer abaixo em qualquer falha; o de um
	// processo morto (que nao roda defer) fica ate o proximo boot.
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

	if err := escrever(tmpFile); err != nil {
		return fmt.Errorf("escrevendo no temporario %q: %w", tmpName, err)
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

// WriteAtomic escreve os dados fornecidos no caminho de destino de forma
// atomica. E o caso particular de ReplaceFile em que os bytes ja estao na mao —
// uma nota inteira, que a escrita ja precisa ter lido para transformar.
func WriteAtomic(ctx context.Context, targetPath string, data []byte) error {
	return ReplaceFile(ctx, targetPath, func(f *os.File) error {
		_, err := f.Write(data)
		return err
	})
}
