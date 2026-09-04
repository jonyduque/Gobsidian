package ipc

import (
	"context"
	"errors"
	"io"
	"os"
)

// EhDesconexaoLimpa diz se um erro devolvido por um loop de transporte
// significa "o outro lado foi embora", e nao falha.
//
// context.Canceled vem do proprio lifecycle; io.EOF e io.ErrClosedPipe sao
// como o SDK reporta o fim do stdin; os.ErrClosed e como o fechamento de um
// pipe ou conn aparece do lado que ainda estava copiando. Tres lugares
// tinham essa lista, e um deles (shutdownExitCode) nao tinha os.ErrClosed —
// um serve que encerrava por essa via saia com codigo 1.
func EhDesconexaoLimpa(err error) bool {
	return err == nil ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, os.ErrClosed)
}
