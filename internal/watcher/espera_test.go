package watcher

import (
	"testing"
	"time"

	"github.com/jonyduque/Gobsidian/internal/vaulttest"
)

// EsperarAte consulta `cond` ate ela valer, e devolve se valeu dentro do prazo.
//
// Um sleep fixo e a pior das duas coisas: lento quando a condicao ja vale, e
// falso quando a maquina esta carregada. Este pacote pagou os dois lados —
// TestWatcher_Burst reprovou com 427 das 500 notas indexadas porque a janela
// media a carga da suite (ver o comentario em burst_test.go), e uma duzia de
// testes gastava 50 a 100 ms parada esperando um laco que ja tinha arrancado.
//
// # Por que devolve bool em vez de chamar t.Fatalf
//
// A versao que recebe `t` e reprova por conta propria escreve a MESMA mensagem
// em todo sitio ("condicao nao valeu em 5s"). Os testes deste pacote tem
// diagnostico caro e especifico: `diagnostico(w)` despeja os oito contadores e
// separa "o watch nunca viu o evento" de "o filtro comeu o evento" de "Apply
// nao indexou"; `bufferDeLog` guarda o log do watcher para o Cleanup despejar,
// e existe porque `io.Discard` ja tornou um estouro de prazo indiagnosticavel
// aqui. Centralizar a ESPERA sem centralizar a MENSAGEM e o que preserva isso.
//
// # Por que exportada, num arquivo _test.go
//
// Ela nao existe no binario de producao — `_test.go` so entra no binario de
// teste. E exportada porque parte dos testes deste diretorio esta em
// `package watcher_test` (apply_test.go, rename_test.go,
// rename_prefiltro_test.go), e essa parte so alcanca identificadores
// exportados de `package watcher`.
func EsperarAte(prazo time.Duration, cond func() bool) bool {
	fim := time.Now().Add(prazo)
	for {
		if cond() {
			return true
		}
		if !time.Now().Before(fim) {
			return false
		}
		// Intervalo de POLLING, nao sincronizacao: a condicao acabou de ser
		// consultada acima e sera de novo na volta seguinte. Nada aqui depende
		// de 10 ms serem suficientes para coisa alguma.
		time.Sleep(10 * time.Millisecond)
	}
}

// EsperarWatcherAtivo espera o laco de Run estar rodando, e reprova se ele nao
// arrancar.
//
// `w.active` e marcado na PRIMEIRA linha de Run. Isso basta como sinal de
// arranque porque os watches do fsnotify sao registrados em `New`, e nao em
// `Run`: quando `New` retorna, o sistema operacional ja esta reportando
// mudancas do cofre, e o que falta e apenas alguem lendo `fsWatcher.Events`.
// Um arquivo criado entre `New` e o primeiro `select` de `Run` nao se perde.
//
// Antes disto, seis testes escreviam `time.Sleep(50 * time.Millisecond)` ou
// `time.Sleep(100 * time.Millisecond)` com o comentario "Wait for watcher to
// start" — um numero que nao vinha de medicao nenhuma e que nao reprovava se o
// laco nunca arrancasse: o teste seguia e falhava depois, dizendo outra coisa.
func EsperarWatcherAtivo(t *testing.T, w *Watcher) {
	t.Helper()
	if !EsperarAte(vaulttest.Prazo, func() bool { return w.Stats().Active }) {
		t.Fatalf("o laco de Run nao ficou ativo em %v; nada do que vem depois "+
			"deste ponto seria observado\n%s", vaulttest.Prazo, diagnostico(w))
	}
}
