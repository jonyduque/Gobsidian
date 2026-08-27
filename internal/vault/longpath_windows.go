//go:build windows

package vault

import (
	"path/filepath"
	"strings"
)

// longPathThreshold e conservador de proposito: o limite e 260, e a folga
// cobre o nome do arquivo temporario que a escrita atomica cria ao lado do
// alvo.
const longPathThreshold = 240

// LongPath prefixa com \\?\ quando o caminho se aproxima de MAX_PATH.
//
// Restricoes do prefixo, que o chamador precisa ter respeitado antes: exige
// caminho absoluto, exige separador "\", e nao aceita "." nem "..".
//
// Isto e para SYSCALL DIRETA — windows.GetFileAttributes e afins. O pacote os
// do Go ja aplica o prefixo sozinho (fixLongPath), entao caminho que passa por
// os.Open, os.Stat ou filepath.WalkDir NAO precisa desta funcao. Sondado em
// 2026-08-27: MkdirAll, WriteFile e WalkDir alcancaram 318 caracteres sem
// prefixo nenhum. Uma versao anterior desta tarefa acrescentou aqui um
// LongPathSempre para a raiz de varredura e a prova de mutacao o reprovou como
// guarda morta — ver o comentario em writer.SweepStaleTempFiles.
func LongPath(abs string) string {
	if len(abs) < longPathThreshold {
		return abs
	}
	if strings.HasPrefix(abs, `\\?\`) {
		return abs
	}
	clean := filepath.Clean(abs)
	if strings.HasPrefix(clean, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(clean, `\\`)
	}
	return `\\?\` + clean
}
