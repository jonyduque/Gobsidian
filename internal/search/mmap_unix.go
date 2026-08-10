//go:build unix

package search

import (
	"fmt"
	"os"
	"syscall"
)

// mapearArquivo mapeia o arquivo inteiro em modo leitura via syscall.Mmap da
// biblioteca padrão — sem dependência nova (ver CLAUDE.md, "nunca rode go mod
// tidy": x/sys já é indireta, mas syscall cobre isto sem precisar promovê-la).
//
// f é fechado ANTES de devolver, com sucesso ou erro: no Unix, ao contrário do
// Windows, o mapeamento não depende do handle do arquivo continuar aberto —
// mmap duplica a referência ao inode internamente.
func mapearArquivo(f *os.File, tamanho int64) ([]byte, func() error, error) {
	if tamanho == 0 {
		_ = f.Close()
		return nil, nil, fmt.Errorf("arquivo vazio")
	}
	dados, err := syscall.Mmap(int(f.Fd()), 0, int(tamanho), syscall.PROT_READ, syscall.MAP_SHARED)
	_ = f.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("mmap: %w", err)
	}
	fechar := func() error {
		return syscall.Munmap(dados)
	}
	return dados, fechar, nil
}
