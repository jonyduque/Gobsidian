//go:build windows

package writer

// sincronizarDiretorio nao existe no Windows, e este arquivo diz isso em vez de
// fingir.
//
// O Windows nao expoe handle de diretorio sincronizavel pela API que o pacote
// os usa: CreateFile num diretorio exige FILE_FLAG_BACKUP_SEMANTICS e o
// FlushFileBuffers resultante nao tem a semantica do fsync de diretorio POSIX.
// A durabilidade do rename no NTFS e coberta pelo journal do proprio sistema de
// arquivos, que registra a operacao de metadado antes de aplica-la.
//
// Devolve nil, e nao um erro, porque nao ha falha: nao ha o que fazer. Um erro
// aqui viraria um aviso em toda escrita, e aviso que sempre aparece deixa de ser
// lido — que e como um aviso de verdade se perde.
func sincronizarDiretorio(string) error { return nil }
