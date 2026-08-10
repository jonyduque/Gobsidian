package service

import (
	"context"
	"sync"
)

// CarregadorBusca traz o indice invertido de busca para o estado pronto —
// carrega o cache do disco, e completa por tokenizacao o que faltar. Quem
// monta o Service fornece a implementacao (ver cmd/gobsidian/serve.go); este
// pacote so decide QUANDO ela roda.
type CarregadorBusca func(ctx context.Context) error

// cargaUnica dispara uma funcao no maximo uma vez COM EXITO.
//
// sync.Once nao serve aqui: ele marca "ja disparei" mesmo quando a funcao
// falha, e a falha vira permanente — nenhuma busca depois consegue tentar de
// novo. cargaUnica so marca pronta depois de f() devolver nil, entao uma
// falha transitoria (cache corrompido, disco lento, contexto cancelado no
// meio da construcao) pode ser tentada de novo na proxima chamada.
//
// O mutex tambem resolve a concorrencia: goroutines que chegam enquanto f()
// esta rodando esperam no Lock em vez de disparar f() de novo, entao N
// buscas simultaneas na primeira chamada produzem UMA carga, nao N.
type cargaUnica struct {
	mu     sync.Mutex
	pronta bool
}

// fazer roda f() se ainda nao rodou com exito. Devolve o erro de f() sem
// marcar pronta, para que a proxima chamada tente de novo.
func (c *cargaUnica) fazer(f func() error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pronta {
		return nil
	}
	if err := f(); err != nil {
		return err
	}
	c.pronta = true
	return nil
}

// garanteIndiceDeBusca dispara o carregamento do indice invertido na
// primeira busca, e so nela. Buscas concorrentes durante o carregamento
// esperam a mesma carga terminar em vez de ver INDEX_BUILDING — a carga a
// partir do cache mede bem abaixo de 1 s no cofre de referencia, e esperar e
// mais simples do que fazer o cliente reconsultar.
//
// carregarBusca nil (modo --eager-search, ou um Service montado em teste
// com o indice ja pronto ou ja em construcao por outro caminho) faz desta
// funcao um no-op: nao ha o que carregar sob demanda.
func (s *Service) garanteIndiceDeBusca(ctx context.Context) error {
	if s.carregarBusca == nil {
		return nil
	}
	return s.cargaBusca.fazer(func() error {
		return s.carregarBusca(ctx)
	})
}
