package lifecycle

import (
	"context"
	"errors"
	"io"
)

// watchStdin encerra o processo quando o host MCP fecha o stdin do filho.
// E o mecanismo primario e o mais confiavel no Windows: o sistema operacional
// fecha os handles de um processo morto, inclusive apos taskkill /F.
//
// Esta goroutine e deliberadamente NAO registrada no WaitGroup. Read bloqueia
// e nao ha forma portavel de interromper uma leitura bloqueada num io.Reader
// arbitrario via cancelamento de context — ctx aqui so existe para consistencia
// de assinatura, nao e observado dentro do loop. Se ela estivesse no
// WaitGroup, o dia em que sinais (Task 4) ou vigilia do pai (Task 5)
// disparassem o desligamento primeiro, Wait() bloquearia para sempre esperando
// uma leitura que nunca retorna enquanto stdin permanecer aberto. A saida
// documentada desta goroutine e o encerramento do processo, nao um retorno
// de funcao aguardado por alguem.
func (l *Lifecycle) watchStdin(ctx context.Context, r io.Reader) {
	go func() {
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
