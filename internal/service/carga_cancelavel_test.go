package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestCargaConcorrenteRespeitaOPrazoDoChamador cobre a metade do A6 que atinge
// quem CHEGA durante a carga.
//
// `cargaUnica.fazer` segurava o mutex durante toda a carga, e os concorrentes
// esperavam em `mu.Lock()` puro — sem `select` em `ctx.Done()`. O prazo que o
// host mandou era ignorado: com cache frio ou corrompido, a tokenização do
// cofre inteiro roda por minutos sem resposta e sem erro.
func TestCargaConcorrenteRespeitaOPrazoDoChamador(t *testing.T) {
	var c cargaUnica

	comecou := make(chan struct{})
	liberar := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = c.fazer(context.Background(), func(context.Context) error {
			close(comecou)
			<-liberar
			return nil
		})
	}()
	<-comecou

	// Concorrente com prazo curto: tem de voltar com o erro do context, e nao
	// esperar a carga inteira.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	inicio := time.Now()
	err := c.fazer(ctx, func(context.Context) error {
		t.Error("o concorrente NAO devia disparar uma segunda carga")
		return nil
	})
	decorrido := time.Since(inicio)

	if err == nil {
		t.Error("o concorrente esperou a carga inteira em vez de respeitar o prazo")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("erro = %v, queria context.DeadlineExceeded", err)
	}
	if decorrido > 2*time.Second {
		t.Errorf("o concorrente demorou %s; o prazo era de 100ms", decorrido)
	}

	close(liberar)
	wg.Wait()
}

// TestCargaSegueEmSegundoPlanoAposOPrazo fixa a outra metade: o chamador
// desiste, a CARGA não.
//
// Sem isto, devolver INDEX_BUILDING no prazo cancelaria o trabalho junto, e a
// próxima busca recomeçaria do zero — trocando uma espera longa por uma espera
// longa repetida.
func TestCargaSegueEmSegundoPlanoAposOPrazo(t *testing.T) {
	var c cargaUnica

	comecou := make(chan struct{})
	liberar := make(chan struct{})
	terminou := make(chan struct{})

	go func() {
		_ = c.fazer(context.Background(), func(context.Context) error {
			close(comecou)
			<-liberar
			close(terminou)
			return nil
		})
	}()
	<-comecou

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_ = c.fazer(ctx, func(context.Context) error { return nil })

	// A carga original nao pode ter sido interrompida pela desistencia do
	// concorrente.
	select {
	case <-terminou:
		t.Error("a carga terminou cedo demais: o cenario nao exercitou nada")
	default:
	}

	close(liberar)
	select {
	case <-terminou:
	case <-time.After(2 * time.Second):
		t.Error("a carga nao concluiu depois de liberada")
	}

	// E depois de concluida, uma busca nova nao recarrega.
	chamou := false
	if err := c.fazer(context.Background(), func(context.Context) error {
		chamou = true
		return nil
	}); err != nil {
		t.Errorf("fazer apos carga concluida: %v", err)
	}
	if chamou {
		t.Error("recarregou apesar de a carga ja ter concluido com exito")
	}
}

// TestCargaPermiteNovaTentativaAposFalha preserva a semântica que já existia e
// já era testada: falha NÃO marca pronta.
//
// Está aqui de novo porque o redesenho troca o mutex por uma porta, e uma porta
// fechada cedo demais tornaria a falha permanente — exatamente o que `sync.Once`
// faria, e o motivo de `cargaUnica` existir.
func TestCargaPermiteNovaTentativaAposFalha(t *testing.T) {
	var c cargaUnica
	falha := errors.New("cache corrompido")

	if err := c.fazer(context.Background(), func(context.Context) error {
		return falha
	}); !errors.Is(err, falha) {
		t.Fatalf("erro = %v, queria %v", err, falha)
	}

	segunda := false
	if err := c.fazer(context.Background(), func(context.Context) error {
		segunda = true
		return nil
	}); err != nil {
		t.Fatalf("segunda tentativa: %v", err)
	}
	if !segunda {
		t.Error("a segunda tentativa nao rodou: a falha virou permanente")
	}
}
