package service_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/service"
	"github.com/jonyd/gobsidian/internal/vault"
)

// servicoComCargaPreguicosa monta um Service minimo, com o indice invertido
// marcado como em construcao, e injeta `carregar` como a funcao que
// Service.garanteIndiceDeBusca dispara na primeira busca. Nao precisa de
// cofre nem de indice de metadados de verdade: o que este teste cobre e o
// contrato entre Search e o carregamento sob demanda, nao a construcao do
// indice em si — essa ja tem cobertura em search_building_test.go e nos
// testes de cmd/gobsidian.
//
// O wrapper chama inv.MarkReady() quando `carregar` tem exito, espelhando o
// que prepararIndiceDeBusca faz de verdade: sem isso, uma segunda busca bem
// sucedida ainda veria Building() == true e devolveria INDEX_BUILDING, o que
// não é o que se está testando aqui.
func servicoComCargaPreguicosa(t *testing.T, carregar func() error) *service.Service {
	t.Helper()
	root := t.TempDir()

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	inv := search.NewInverted()
	inv.MarkBuilding()

	return service.New(v, idx, inv, nil, service.Options{
		CarregarBusca: func(context.Context) error {
			if err := carregar(); err != nil {
				return err
			}
			inv.MarkReady()
			return nil
		},
	})
}

// TestBuscaPreguicosaCarregaUmaVezESoUmaVez guarda os dois defeitos que o
// adiamento introduz: carregar N vezes sob concorrencia, e nunca mais tentar
// depois de uma falha.
func TestBuscaPreguicosaCarregaUmaVezESoUmaVez(t *testing.T) {
	var cargas atomic.Int32
	svc := servicoComCargaPreguicosa(t, func() error {
		cargas.Add(1)
		return nil
	})

	// Vinte buscas concorrentes: uma carga, nao vinte.
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.Search(context.Background(), service.SearchOptions{Query: "x"})
		}()
	}
	wg.Wait()
	if got := cargas.Load(); got != 1 {
		t.Errorf("carregou %d vezes sob concorrencia, quer 1", got)
	}

	// Falha na carga nao pode ser definitiva.
	var tentativas atomic.Int32
	svc2 := servicoComCargaPreguicosa(t, func() error {
		if tentativas.Add(1) == 1 {
			return errors.New("falha transitoria")
		}
		return nil
	})
	if _, err := svc2.Search(context.Background(), service.SearchOptions{Query: "x"}); err == nil {
		t.Fatal("primeira busca deveria propagar a falha da carga")
	}
	if _, err := svc2.Search(context.Background(), service.SearchOptions{Query: "x"}); err != nil {
		t.Errorf("segunda busca falhou (%v): o Once travou o erro para sempre", err)
	}
}
