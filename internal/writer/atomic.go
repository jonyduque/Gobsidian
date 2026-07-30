package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TempFilePrefix e o prefixo usado para todos os arquivos temporarios de escrita atomica.
const TempFilePrefix = ".gobsidian-tmp-"

// CleanStaleTempFiles remove arquivos temporarios de escritas interrompidas no diretorio.
func CleanStaleTempFiles(dir string) {
	matches, err := filepath.Glob(filepath.Join(dir, TempFilePrefix+"*"))
	if err == nil {
		for _, m := range matches {
			_ = os.Remove(m)
		}
	}
}

// WriteAtomic escreve os dados fornecidos no caminho de destino de forma atomica:
// 1. Cria um arquivo temporario no mesmo diretorio do destino.
// 2. Escreve os dados.
// 3. Executa Sync() (fsync) para garantir a gravacao fisica no disco contra quedas de energia.
// 4. Fecha o temporario.
// 5. Executa rename atomico sobre o arquivo de destino, com retry em caso de bloqueio temporario (Windows).
func WriteAtomic(targetPath string, data []byte) error {
	dir := filepath.Dir(targetPath)

	CleanStaleTempFiles(dir)

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
