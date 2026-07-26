package doctor

import (
	"context"
	"fmt"
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
func checkRootExists(_ context.Context, cfg config.Config) Result {
	const name = "raiz do cofre existe"

	info, err := os.Stat(vault.LongPath(cfg.VaultPath))
	if err != nil {
		return Result{
			Name:   name,
			Status: StatusFail,
			Detail: fmt.Sprintf("%q: %v", cfg.VaultPath, err),
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
func checkReadable(_ context.Context, cfg config.Config) Result {
	const name = "permissao de leitura"

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
func checkWritable(_ context.Context, cfg config.Config) Result {
	const name = "permissao de escrita"

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
// arbitrario.
func checkObsidianDir(_ context.Context, cfg config.Config) Result {
	const name = ".obsidian presente"

	info, err := os.Stat(vault.LongPath(filepath.Join(cfg.VaultPath, ".obsidian")))
	if err != nil || !info.IsDir() {
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: "pasta .obsidian ausente: configuracoes, temas e plugins do Obsidian nao serao detectados",
		}
	}
	return Result{Name: name, Status: StatusOK}
}

// checkNoteCount conta as notas .md do cofre. Nunca falha; zero notas e
// aviso, nao erro — um cofre vazio e um fato, nao uma quebra.
func checkNoteCount(ctx context.Context, cfg config.Config) Result {
	const name = "contagem de notas"

	var count int
	err := walkVault(ctx, cfg, func(e vault.Entry) {
		if e.IsNote {
			count++
		}
	})
	if err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("varredura interrompida: %v", err)}
	}
	if count == 0 {
		return Result{Name: name, Status: StatusWarn, Detail: "nenhuma nota .md encontrada"}
	}
	return Result{Name: name, Status: StatusOK, Detail: fmt.Sprintf("%d notas", count)}
}

// checkLongestPath mede o maior caminho absoluto entre as entradas do cofre,
// na forma tradicional (sem o prefixo \\?\ que internal/vault usa
// internamente nas suas proprias chamadas de sistema). E essa forma tradicional
// que o Explorer, clientes de sincronizacao de nuvem e outras ferramentas
// externas recebem, e por isso o limiar existe: o MAX_PATH de 260 delas.
func checkLongestPath(ctx context.Context, cfg config.Config) Result {
	const name = "comprimento de caminho"

	length, longest, err := longestVaultPath(ctx, cfg)
	if err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("varredura interrompida: %v", err)}
	}
	if length > longPathThreshold {
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: fmt.Sprintf("%d caracteres, acima do limiar de %d: %s", length, longPathThreshold, longest),
		}
	}
	return Result{Name: name, Status: StatusOK, Detail: fmt.Sprintf("maior caminho: %d caracteres", length)}
}

// checkCacheDir verifica que o diretorio de cache pode ser criado. Nunca
// falha: sem cache o produto reindexação do zero, mais lento, mas funcional.
func checkCacheDir(_ context.Context, cfg config.Config) Result {
	const name = "diretorio de cache"

	if strings.TrimSpace(cfg.CacheDir) == "" {
		// So acontece quando o chamador monta Config fora de config.Load
		// (por exemplo, um teste que usa config.Defaults() direto). Em uso
		// real config.Load sempre preenche um default fora do cofre.
		return Result{Name: name, Status: StatusOK, Detail: "nenhum diretorio de cache configurado"}
	}

	if err := os.MkdirAll(cfg.CacheDir, 0o755); err != nil {
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
func checkFreeSpace(_ context.Context, cfg config.Config) Result {
	const name = "espaco em disco"

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

// walkVault varre o cofre chamando fn para cada entrada (nota ou anexo). E o
// unico ponto que abre um vault.Vault a partir de cfg — cada verificacao que
// precisa varrer chama esta funcao, e nao abre arquivos: vault.Walk ja evita
// tocar em placeholders de nuvem.
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

// longestVaultPath devolve o comprimento e o valor do maior caminho absoluto
// tradicional (sem prefixo \\?\) entre as entradas do cofre. Compartilhada
// entre checkLongestPath e a verificacao de caminhos longos do Windows, que
// precisa do mesmo dado para decidir se o registro importa.
func longestVaultPath(ctx context.Context, cfg config.Config) (length int, longest string, err error) {
	walkErr := walkVault(ctx, cfg, func(e vault.Entry) {
		abs := filepath.Join(cfg.VaultPath, filepath.FromSlash(string(e.Path)))
		if n := len(abs); n > length {
			length = n
			longest = abs
		}
	})
	if walkErr != nil {
		return length, longest, walkErr
	}
	return length, longest, nil
}
