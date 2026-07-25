package lifecycle

import (
	"context"
	"errors"
	"io"
)

// watchStdin encerra o processo quando o host MCP fecha o stdin do filho.
// E o mecanismo primario e o mais confiavel no Windows: o sistema operacional
// fecha os handles de um processo morto, inclusive apos taskkill /F.
func (l *Lifecycle) watchStdin(ctx context.Context, r io.Reader) {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()

		buf := make([]byte, 4096)
		for {
			_, err := r.Read(buf)
			if err == nil {
				continue
			}
			if errors.Is(err, io.EOF) {
				l.trigger("stdin-eof")
				return
			}
			// Qualquer outro erro de leitura em stdin tambem significa que
			// o canal com o host morreu. Tratar diferente seria fingir que
			// ha um estado de recuperacao que nao existe.
			l.trigger("stdin-error")
			return
		}
	}()
}
