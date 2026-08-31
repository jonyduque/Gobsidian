//go:build !windows

package writer

import "os"

// sincronizarDiretorio forca a entrada de diretorio ao disco.
//
// O Sync do arquivo garante os DADOS gravados; o rename e uma mudanca de
// metadado do DIRETORIO, e em ext4/xfs ela pode ficar em cache. Uma queda de
// energia logo apos um WriteAtomic bem-sucedido pode entao deixar o alvo com o
// conteudo antigo, ou com nenhum, apesar de a escrita ter reportado sucesso.
//
// Abrir o diretorio so para leitura e chamar Sync e o idioma POSIX para isso.
func sincronizarDiretorio(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return d.Sync()
}
