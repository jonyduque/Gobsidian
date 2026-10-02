package lifecycle

import (
	"os"
	"os/signal"
)

// canalArmado devolve o canal armado sem consumi-lo -- so para teste.
func canalArmado() chan os.Signal {
	muArmados.Lock()
	defer muArmados.Unlock()
	return sinaisArmados
}

// desarmarSinaisParaTeste devolve o pacote ao estado de antes de ArmarSinais.
func desarmarSinaisParaTeste() {
	muArmados.Lock()
	defer muArmados.Unlock()
	if sinaisArmados != nil {
		signal.Stop(sinaisArmados)
		sinaisArmados = nil
	}
}
