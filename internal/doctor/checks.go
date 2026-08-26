package doctor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/vault"
)

// longPathThreshold espelha o limiar conservador de internal/vault (240, com
// folga abaixo do MAX_PATH de 260 do Windows para o nome temporario que a
// escrita atomica cria ao lado do alvo). Duplicado aqui de proposito: o
// original em internal/vault nao e exportado, e doctor mede um caminho
// diferente dele (o caminho absoluto tradicional, sem o prefixo \\?\, que e o
// que ferramentas externas ao gobsidian — Explorer, clientes de nuvem, git —
// realmente recebem).
const longPathThreshold = 240

// minFreeSpaceFailBytes e minFreeSpaceWarnBytes definem os tetos de espaco
// livre em disco. Abaixo do primeiro, escritas atomicas (que exigem espaco
// para o arquivo temporario ao lado do original) podem comecar a falhar;
// abaixo do segundo, ainda funcionam mas o usuario esta perto do limite.
const (
	minFreeSpaceFailBytes = 10 << 20  // 10 MB
	minFreeSpaceWarnBytes = 100 << 20 // 100 MB
)

// checkRootExists verifica que a raiz do cofre existe e e um diretorio. E a
// unica verificacao cuja falha interrompe as demais: sem raiz acessivel, todo
// resultado subsequente seria derivado e o relatorio viraria ruido.
func checkRootExists(ctx context.Context, cfg config.Config) Result {
	const name = "raiz do cofre existe"

	// os.Stat bloqueia em um share de rede parado ou um mount de nuvem sem
	// resposta. Nao da para interromper o syscall ja em voo, mas dá para nao
	// nem comecar quando o chamador ja desistiu.
	if err := ctx.Err(); err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("varredura interrompida: %v", err)}
	}

	info, err := os.Stat(vault.LongPath(cfg.VaultPath))
	if err != nil {
		// A dica de grafia mora AQUI, e nao numa verificacao propria, porque
		// esta e halting: ela aborta as seguintes, e uma checagem posterior
		// nunca rodaria justamente no caso para o qual existiria.
		//
		// Medido em 2026-08-26: o config de um host apontava para
		// ...\Obsidian\Jurisprudencia, sem acento, e no disco so existe
		// Jurisprudencia COM acento. As duas grafias produzem VaultKey
		// diferente -- socket, cache e daemon proprios -- e o servidor morria
		// na partida. "nao existe" e verdadeiro e inutil; a diferenca entre as
		// duas grafias e invisivel numa leitura apressada de JSON.
		detalhe := fmt.Sprintf("%q: %v", cfg.VaultPath, err)
		if vizinhos := vizinhosParecidos(cfg.VaultPath); len(vizinhos) > 0 {
			detalhe += fmt.Sprintf("\n     existe(m) ao lado, com grafia diferente: %s", strings.Join(vizinhos, ", "))
		}
		return Result{
			Name:   name,
			Status: StatusFail,
			Detail: detalhe,
		}
	}
	if !info.IsDir() {
		return Result{
			Name:   name,
			Status: StatusFail,
			Detail: fmt.Sprintf("%q existe mas nao e um diretorio", cfg.VaultPath),
		}
	}
	return Result{Name: name, Status: StatusOK}
}

// checkReadable verifica que o processo consegue listar a raiz do cofre.
func checkReadable(ctx context.Context, cfg config.Config) Result {
	const name = "permissao de leitura"

	if err := ctx.Err(); err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("varredura interrompida: %v", err)}
	}

	entries, err := os.ReadDir(vault.LongPath(cfg.VaultPath))
	if err != nil {
		return Result{
			Name:   name,
			Status: StatusFail,
			Detail: fmt.Sprintf("nao foi possivel listar %q: %v", cfg.VaultPath, err),
		}
	}
	return Result{Name: name, Status: StatusOK, Detail: fmt.Sprintf("%d entradas na raiz", len(entries))}
}

// checkWritable cria e apaga um arquivo temporario na raiz do cofre. A
// verificacao roda mesmo com --read-only ligado — e o unico jeito de saber se
// a falha aconteceria, o que muda apenas a gravidade do resultado: falha
// bloqueante quando o produto precisa escrever, aviso quando o usuario ja
// pediu para nao escrever e portanto a resposta nao importa para o exit code.
func checkWritable(ctx context.Context, cfg config.Config) Result {
	const name = "permissao de escrita"

	if err := ctx.Err(); err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("varredura interrompida: %v", err)}
	}

	root := vault.LongPath(cfg.VaultPath)
	f, err := os.CreateTemp(root, ".gobsidian-doctor-*.tmp")
	if err == nil {
		path := f.Name()
		closeErr := f.Close()
		removeErr := os.Remove(path)
		switch {
		case closeErr != nil:
			err = closeErr
		case removeErr != nil:
			err = removeErr
		}
	}

	if err != nil {
		detail := fmt.Sprintf("nao foi possivel escrever em %q: %v", cfg.VaultPath, err)
		if cfg.ReadOnly {
			return Result{Name: name, Status: StatusWarn, Detail: detail}
		}
		return Result{Name: name, Status: StatusFail, Detail: detail}
	}
	return Result{Name: name, Status: StatusOK}
}

// checkObsidianDir avisa quando a pasta .obsidian/ nao existe. Nunca falha: o
// produto funciona sobre qualquer pasta de Markdown, e exigir .obsidian/ seria
// arbitrario. Nao-existe e distinguido de outros erros (permissao, por
// exemplo): uma .obsidian/ que existe mas nao pode ser lida nao e a mesma
// situacao que uma .obsidian/ que nunca existiu, e reportar as duas como
// "ausente" esconderia um problema de permissao real.
func checkObsidianDir(ctx context.Context, cfg config.Config) Result {
	const name = ".obsidian presente"

	if err := ctx.Err(); err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("varredura interrompida: %v", err)}
	}

	path := filepath.Join(cfg.VaultPath, ".obsidian")
	info, err := os.Stat(vault.LongPath(path))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: "pasta .obsidian ausente: configuracoes, temas e plugins do Obsidian nao serao detectados",
		}
	case err != nil:
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: fmt.Sprintf("nao foi possivel verificar %q: %v", path, err),
		}
	case !info.IsDir():
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: fmt.Sprintf("%q existe mas nao e um diretorio", path),
		}
	default:
		return Result{Name: name, Status: StatusOK}
	}
}

// checkNoteCount conta as notas .md do cofre. Nunca falha; zero notas e
// aviso, nao erro — um cofre vazio e um fato, nao uma quebra. Le do resultado
// de scanVault em vez de varrer o cofre de novo.
func checkNoteCount(scan vaultScan) Result {
	const name = "contagem de notas"

	if res, failed := scanStatus(scan, name); failed {
		return res
	}
	if scan.noteCount == 0 {
		return Result{Name: name, Status: StatusWarn, Detail: "nenhuma nota .md encontrada"}
	}
	return Result{Name: name, Status: StatusOK, Detail: fmt.Sprintf("%d notas", scan.noteCount)}
}

// checkLongestPath mede o maior caminho absoluto entre as entradas do cofre,
// na forma tradicional (sem o prefixo \\?\ que internal/vault usa
// internamente nas suas proprias chamadas de sistema). E essa forma tradicional
// que o Explorer, clientes de sincronizacao de nuvem e outras ferramentas
// externas recebem, e por isso o limiar existe: o MAX_PATH de 260 delas. Le do
// resultado de scanVault em vez de varrer o cofre de novo.
func checkLongestPath(scan vaultScan) Result {
	const name = "comprimento de caminho"

	if res, failed := scanStatus(scan, name); failed {
		return res
	}
	if scan.longestPathLen > longPathThreshold {
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: fmt.Sprintf("%d caracteres, acima do limiar de %d: %s", scan.longestPathLen, longPathThreshold, scan.longestPath),
		}
	}
	return Result{Name: name, Status: StatusOK, Detail: fmt.Sprintf("maior caminho: %d caracteres", scan.longestPathLen)}
}

// checkCacheDir verifica que o diretorio de cache pode ser criado. Nunca
// falha: sem cache o produto reindexação do zero, mais lento, mas funcional.
func checkCacheDir(ctx context.Context, cfg config.Config) Result {
	const name = "diretorio de cache"

	if err := ctx.Err(); err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("varredura interrompida: %v", err)}
	}

	if strings.TrimSpace(cfg.CacheDir) == "" {
		// So acontece quando o chamador monta Config fora de config.Load
		// (por exemplo, um teste que usa config.Defaults() direto). Em uso
		// real config.Load sempre preenche um default fora do cofre.
		return Result{Name: name, Status: StatusOK, Detail: "nenhum diretorio de cache configurado"}
	}

	if err := os.MkdirAll(vault.LongPath(cfg.CacheDir), 0o755); err != nil {
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: fmt.Sprintf("nao foi possivel criar %q: %v", cfg.CacheDir, err),
		}
	}
	return Result{Name: name, Status: StatusOK, Detail: cfg.CacheDir}
}

// checkFreeSpace verifica o espaco livre no volume que contem a raiz do
// cofre. Falha (bloqueante) abaixo de 10 MB, porque escritas atomicas podem
// comecar a falhar; aviso abaixo de 100 MB.
func checkFreeSpace(ctx context.Context, cfg config.Config) Result {
	const name = "espaco em disco"

	if err := ctx.Err(); err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("varredura interrompida: %v", err)}
	}

	free, err := diskFreeBytes(cfg.VaultPath)
	if err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("nao foi possivel medir espaco livre: %v", err)}
	}

	detail := fmt.Sprintf("%d MB livres", free/(1<<20))
	switch {
	case free < minFreeSpaceFailBytes:
		return Result{Name: name, Status: StatusFail, Detail: detail}
	case free < minFreeSpaceWarnBytes:
		return Result{Name: name, Status: StatusWarn, Detail: detail}
	default:
		return Result{Name: name, Status: StatusOK, Detail: detail}
	}
}

// vaultScan e o resultado de uma unica varredura do cofre, usado por todo
// check que precisa de dados por arquivo (contagem de notas, caminho mais
// longo, arquivos somente-nuvem, colisoes de casing). Sem isso, cada um desses
// checks abriria o cofre e chamaria vault.Walk por conta propria — no cofre de
// 20 mil notas que o produto tem como alvo, cinco varreduras completas (quatro
// aqui mais o registro de caminhos longos do Windows) em vez de uma.
//
// err e nao-nulo quando a varredura foi interrompida (contexto cancelado,
// cofre que sumiu entre checkReadable e aqui, etc.) — os checks que consomem
// o scan reportam isso como aviso de "varredura interrompida", nunca como
// "zero encontrado", que seria uma resposta enganosa.
type vaultScan struct {
	noteCount        int
	longestPathLen   int
	longestPath      string
	cloudOnlyCount   int
	casingCollisions []string
	err              error
}

// scanStatus traduz o erro de uma varredura interrompida em Result, e e o
// unico ponto onde essa traducao acontece — os seis checks que consomem
// vaultScan chamam este helper em vez de montar o Result cada um.
//
// A distincao e o ponto todo. Cancelamento veio de quem chamou: o usuario deu
// Ctrl-C, ou o comando de cima desistiu. Nao e problema do ambiente e nao
// deve virar codigo de saida nao-zero. Qualquer outro erro veio do cofre —
// vault.Walk so o produz quando falha na propria raiz, que significa unidade
// desconectada, share caido ou pasta movida pelo cliente de nuvem. Reportar
// isso como aviso faz doctor sair 0 sobre um cofre que o servidor nao
// conseguira abrir, que e exatamente a confusao que este comando existe para
// desfazer.
//
// O bool devolvido e "houve erro": false significa que o chamador segue com
// os dados do scan.
func scanStatus(scan vaultScan, name string) (Result, bool) {
	if scan.err == nil {
		return Result{}, false
	}
	if errors.Is(scan.err, context.Canceled) || errors.Is(scan.err, context.DeadlineExceeded) {
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: fmt.Sprintf("varredura interrompida: %v", scan.err),
		}, true
	}
	return Result{
		Name:   name,
		Status: StatusFail,
		Detail: fmt.Sprintf("cofre inacessivel durante a varredura: %v", scan.err),
	}, true
}

// scanVault varre o cofre uma unica vez e coleta tudo que os checks acima
// precisam. So e chamada depois que os checks bloqueantes (raiz existe, raiz
// legivel) passam: varrer um cofre cuja raiz nao pode ser lida so produziria
// mais um erro derivado do mesmo problema ja sinalizado.
func scanVault(ctx context.Context, cfg config.Config) vaultScan {
	var scan vaultScan
	seen := make(map[string]string)

	scan.err = walkVault(ctx, cfg, func(e vault.Entry) {
		if e.IsNote {
			scan.noteCount++
		}

		abs := filepath.Join(cfg.VaultPath, filepath.FromSlash(string(e.Path)))
		if n := len(abs); n > scan.longestPathLen {
			scan.longestPathLen = n
			scan.longestPath = abs
		}

		if !e.IsNote {
			return
		}

		if e.CloudOnly {
			scan.cloudOnlyCount++
		}

		key := strings.ToLower(string(e.Path))
		if prev, ok := seen[key]; ok {
			if prev != string(e.Path) {
				scan.casingCollisions = append(scan.casingCollisions, fmt.Sprintf("%s <-> %s", prev, e.Path))
			}
			return
		}
		seen[key] = string(e.Path)
	})

	return scan
}

// walkVault varre o cofre chamando fn para cada entrada (nota ou anexo). E o
// unico ponto que abre um vault.Vault a partir de cfg — scanVault e a unica
// chamadora, e nao abre arquivos: vault.Walk ja evita tocar em placeholders
// de nuvem.
func walkVault(ctx context.Context, cfg config.Config, fn func(vault.Entry)) error {
	v, err := vault.New(cfg.VaultPath)
	if err != nil {
		return fmt.Errorf("abrindo cofre: %w", err)
	}
	return v.Walk(ctx, func(e vault.Entry) error {
		fn(e)
		return nil
	})
}
