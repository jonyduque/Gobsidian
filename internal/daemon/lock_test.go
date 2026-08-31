package daemon_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/daemon"
)

// TestDezPontesIniciamUmDaemonSo e o teste nomeado na prova de mutacao do
// brief da Task 92: se "if !adquiriu {" virar "if false {", as dez chamadas
// tratam a corrida como vencida por todas elas, e o contador abaixo sobe
// para 10 em vez de ficar em 1.
//
// A largada usa uma barreira em vez de simplesmente lancar dez goroutines e
// esperar: sem ela, o VENCEDOR poderia terminar (liberar o lock) antes de
// uma goroutine atrasada por agendamento do SO sequer TENTAR adquirirLock --
// essa atrasada veria o lock ja livre e venceria uma corrida NOVA, inflando
// o contador para 2 sem nenhum defeito na exclusao mutua em si (isso
// aconteceu de verdade rodando a suite inteira sob carga: "go test ./..."
// contra "go test ./internal/daemon/..." isolado). A barreira faz a
// PRIMEIRA chamada travar DENTRO de iniciar -- ou seja, com o lock em maos,
// confirmado pelo canal abaixo -- antes das outras nove serem lancadas, e so
// solta a primeira depois de uma folga para as nove tentarem adquirir o
// lock (retido) e desistirem sem nunca chamar iniciar.
//
// iniciar so incrementa um contador -- nenhum daemon de verdade sobe, entao
// EnsureStarted sempre acaba desistindo em esperarSocket para as dez
// chamadas. O erro final de cada uma e ignorado de proposito: o que este
// teste prova e QUANTAS VEZES iniciar foi chamado, nao se a espera por um
// socket que nunca vai existir teve sucesso.
func TestDezPontesIniciamUmDaemonSo(t *testing.T) {
	vault := t.TempDir()
	cfg := config.Config{VaultPath: vault}

	var chamadas atomic.Int32
	vencedorEntrou := make(chan struct{})
	vencedorPodeSair := make(chan struct{})

	iniciar := func() error {
		n := chamadas.Add(1)
		if n == 1 {
			close(vencedorEntrou)
			<-vencedorPodeSair
		}
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = daemon.EnsureStarted(ctx, cfg, 500*time.Millisecond, iniciar)
	}()

	select {
	case <-vencedorEntrou:
	case <-time.After(2 * time.Second):
		t.Fatal("nenhuma chamada entrou em iniciar a tempo -- adquirirLock nunca venceu a corrida")
	}

	var perdedores sync.WaitGroup
	for range 9 {
		perdedores.Add(1)
		go func() {
			defer perdedores.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = daemon.EnsureStarted(ctx, cfg, 300*time.Millisecond, iniciar)
		}()
	}

	// Espera as nove RETORNAREM, e nao um sono de duracao arbitraria.
	//
	// Ate 2026-08-31 aqui havia `time.Sleep(200ms)` com o comentario de que
	// precisava ser "o suficiente para sobreviver a uma maquina sob carga".
	// Nao era: o gate reprovou com `iniciar foi chamado 2 vez(es)` durante uma
	// rodada de `go test -race ./...`, que e exatamente a carga que o
	// comentario previa. Um teste que passa por folga de relogio reprova por
	// falta dela, e um gate que reprova ao acaso ensina a re-rodar ate ficar
	// verde -- que e como uma falha real passa despercebida.
	//
	// A espera e deterministica porque uma goroutine que JA RETORNOU de
	// EnsureStarted por definicao ja tentou adquirirLock: ela nao pode mais
	// vencer uma corrida nova. E nenhuma delas depende do vencedor soltar o
	// lock para retornar -- todas desistem sozinhas em esperarSocket, porque
	// socket nenhum vai existir.
	perdedores.Wait()
	close(vencedorPodeSair)

	wg.Wait()

	if got := chamadas.Load(); got != 1 {
		t.Fatalf("iniciar foi chamado %d vez(es), esperado exatamente 1 -- dez pontes concorrentes tem de resultar em UM daemon", got)
	}
}

// TestEnsureStartedPerdedorNuncaChamaIniciarSozinho e um caso menor do
// mesmo mecanismo, sequencial em vez de concorrente: a segunda chamada, com
// o lock da primeira ainda ativo, nunca deve invocar iniciar.
func TestEnsureStartedPerdedorNuncaChamaIniciar(t *testing.T) {
	vault := t.TempDir()
	cfg := config.Config{VaultPath: vault}

	primeiraLiberou := make(chan struct{})
	primeiraPodeSair := make(chan struct{})
	iniciarPrimeira := func() error {
		close(primeiraLiberou)
		<-primeiraPodeSair
		return nil
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = daemon.EnsureStarted(ctx, cfg, 200*time.Millisecond, iniciarPrimeira)
	}()

	<-primeiraLiberou // a primeira ja adquiriu o lock e esta "iniciando"

	var chamadasSegunda atomic.Int32
	iniciarSegunda := func() error {
		chamadasSegunda.Add(1)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = daemon.EnsureStarted(ctx, cfg, 200*time.Millisecond, iniciarSegunda)

	close(primeiraPodeSair)

	if got := chamadasSegunda.Load(); got != 0 {
		t.Fatalf("iniciar (segunda chamada) foi invocado %d vez(es) com o lock da primeira ainda ativo, esperado 0", got)
	}
}
