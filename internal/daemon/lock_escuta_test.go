package daemon_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jonyd/gobsidian/internal/daemon"
)

// TestLockDeEscutaSerializaSondaEBind fecha o item 4 do brief da Task 126 — a
// metade que a prova de órfão NÃO fecha.
//
// `ipc.Listen` prova que o socket está órfão antes de desvinculá-lo, o que
// impede roubar o socket de um daemon vivo. Mas a sonda e o bind **não são
// atômicos entre si**: dois daemons lançados no mesmo instante podem ambos
// sondar "ninguém escuta" antes de qualquer um dos dois bindar, e aí os dois
// desvinculam e bindam — duas instâncias servindo o mesmo cofre, gravando
// concorrentemente no mesmo cache de busca.
//
// O teste exercita a região crítica diretamente: se ela não for exclusiva, mais
// de um entra ao mesmo tempo e o contador de simultâneos passa de um.
func TestLockDeEscutaSerializaSondaEBind(t *testing.T) {
	vault := t.TempDir()

	const tentativas = 10
	var dentroAgora atomic.Int32
	var maxSimultaneos atomic.Int32
	var entraram atomic.Int32
	var recusados atomic.Int32

	var wg sync.WaitGroup
	inicio := make(chan struct{})
	for range tentativas {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-inicio
			err := daemon.ComLockDeEscuta(vault, func() error {
				entraram.Add(1)
				n := dentroAgora.Add(1)
				for {
					pico := maxSimultaneos.Load()
					if n <= pico || maxSimultaneos.CompareAndSwap(pico, n) {
						break
					}
				}
				// Segura a região crítica tempo suficiente para que os
				// concorrentes tenham chance real de entrar junto.
				for i := 0; i < 200000; i++ {
					_ = i
				}
				dentroAgora.Add(-1)
				return nil
			})
			if err != nil {
				recusados.Add(1)
			}
		}()
	}
	close(inicio)
	wg.Wait()

	if got := maxSimultaneos.Load(); got > 1 {
		t.Errorf("%d daemons dentro da regiao critica ao mesmo tempo\n"+
			"a sonda de orfao e o bind precisam ser exclusivos entre processos: "+
			"dois que sondem juntos ambos veem \"ninguem escuta\" e ambos bindam", got)
	}
	// Sem isto o teste passaria se NINGUÉM entrasse — um lock que recusa todo
	// mundo é exclusivo e inútil.
	if entraram.Load() == 0 {
		t.Fatal("ninguem entrou na regiao critica: o cenario nao exercita nada")
	}
	if entraram.Load()+recusados.Load() != tentativas {
		t.Errorf("entraram=%d recusados=%d, soma != %d: alguem sumiu sem resposta",
			entraram.Load(), recusados.Load(), tentativas)
	}
	t.Logf("entraram=%d recusados=%d simultaneos_max=%d",
		entraram.Load(), recusados.Load(), maxSimultaneos.Load())
}

// TestLockDeEscutaLiberaDepois fixa que o lock não sobrevive à chamada.
//
// Um lock de escuta que vaza trava TODA partida de daemon daquele cofre até
// alguém apagar o arquivo à mão — e o sintoma seria o mesmo defeito de campo de
// 2026-08-26, com as sessões caindo no fallback em processo para sempre.
func TestLockDeEscutaLiberaDepois(t *testing.T) {
	vault := t.TempDir()

	for i := range 3 {
		entrou := false
		if err := daemon.ComLockDeEscuta(vault, func() error {
			entrou = true
			return nil
		}); err != nil {
			t.Fatalf("chamada %d recusada: %v\no lock da chamada anterior nao foi liberado", i, err)
		}
		if !entrou {
			t.Fatalf("chamada %d nao entrou na regiao critica", i)
		}
	}
}
