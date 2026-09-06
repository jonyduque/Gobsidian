package boot

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/jonyd/gobsidian/internal/lifecycle"
)

// Vigia e o que o processo precisa para saber quando o host foi embora: o
// Lifecycle que explica por que o encerramento foi pedido, e o stdin
// espelhado que o servidor le enquanto o lifecycle vigia o EOF. O context
// que cancela NAO mora aqui — VigiarHost o devolve a parte, como
// lifecycle.New faz, porque context em campo de struct e o que a
// documentacao do proprio pacote desaconselha e o que contextcheck acusa:
// quem recebe `ctx` de um campo nao consegue provar que ele descende do
// context do chamador.
//
// O andaime que ele guarda — pipe, mirrorReader, lifecycle.New — estava
// escrito duas vezes, em serveEmProcesso e em servePonteRemota, e a ORDEM
// entre as tres pecas e o que faz o EOF do host chegar ao monitor de stdin:
// o mirrorReader tem de existir antes de lifecycle.New receber a ponta de
// leitura do pipe. Duas copias de uma ordem que importa e uma copia que vai
// ficar para tras.
type Vigia struct {
	LC    *lifecycle.Lifecycle
	Stdin io.Reader
	pw    *io.PipeWriter
}

// VigiarHost liga a vigilia e devolve o context que EOF em stdin ou morte do
// PID pai cancelam, junto com o Vigia que o explica e alimenta.
//
// O monitor de stdin do lifecycle CONSOME bytes, e o stdin aqui pertence ao
// JSON-RPC (ou, na ponte, ao daemon do outro lado do socket). A saida e
// espelhar: quem serve le de Stdin, e o lifecycle observa so a copia.
func VigiarHost(parent context.Context, stdin io.Reader, log *slog.Logger) (context.Context, *Vigia) {
	pr, pw := io.Pipe()
	ctx, lc := lifecycle.New(parent, lifecycle.Options{
		Stdin:     pr,
		ParentPID: lifecycle.ParentPID(),
		Logger:    log,
	})

	return ctx, &Vigia{
		LC:    lc,
		Stdin: &mirrorReader{src: stdin, dst: pw},
		pw:    pw,
	}
}

// PassoFecharEspelho fecha o pipe para que o leitor de stdin do servidor
// receba EOF; 500 ms de orcamento.
//
// O nome "close-pipe" e lido pelo gate de orfaos e pelos logs: nao muda.
func (v *Vigia) PassoFecharEspelho() lifecycle.Step {
	return lifecycle.Step{Name: "close-pipe", Budget: 500 * time.Millisecond, Fn: func(context.Context) error {
		return v.pw.Close()
	}}
}
