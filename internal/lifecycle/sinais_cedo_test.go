package lifecycle

import (
	"context"
	"io"
	"os"
	"testing"
	"time"
)

// TestSinalAntesDoLifecycleNaoSePerde: o sinal que chega entre ArmarSinais e
// New fica no canal armado, e o New o adota -- o encerramento sai com
// reason=signal em vez de o processo morrer pela acao padrao.
//
// Medido no CI: o gate de orfaos reprovou em 4 de 15 execucoes desde
// 2026-09-16, cada uma com "1 de 100 ciclos encerraram sem registrar
// reason=". O tratador so existia dentro de lifecycle.New, e antes dele o
// processo ainda monta o cobra, carrega a config e, em servePonte, disca e as
// vezes sobe o daemon. Um CTRL_BREAK nessa janela nao tinha tratador.
//
// Este teste injeta o sinal no canal em vez de mandar um de verdade: no Windows
// um CTRL_BREAK para o proprio processo derrubaria o go test inteiro. O sinal
// real esta em TestSinalRealAntesDoLifecycle (signals_test.go), fora do
// Windows.
func TestSinalAntesDoLifecycleNaoSePerde(t *testing.T) {
	ArmarSinais()
	t.Cleanup(desarmarSinaisParaTeste)

	armado := canalArmado()
	if armado == nil {
		t.Fatal("ArmarSinais nao armou canal nenhum")
	}
	armado <- os.Interrupt // chegou antes de o lifecycle existir

	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })
	ctx, lc := New(context.Background(), Options{Stdin: pr})

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("o sinal recebido antes de New se perdeu: ctx nao foi cancelado")
	}
	if got := lc.Reason(); got != "signal" {
		t.Errorf("Reason() = %q, quer \"signal\"", got)
	}
	lc.Wait()

	if canalArmado() != nil {
		t.Error("o canal armado continuou disponivel depois de adotado: um segundo New o disputaria")
	}
}
