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

// cargaUnica dispara uma funcao no maximo uma vez COM EXITO, e deixa quem
// espera desistir.
//
// # Por que nao sync.Once
//
// sync.Once marca "ja disparei" mesmo quando a funcao falha, e a falha vira
// permanente — nenhuma busca depois consegue tentar de novo. cargaUnica so
// marca pronta depois de f() devolver nil, entao uma falha transitoria (cache
// corrompido, disco lento) pode ser tentada de novo na proxima chamada.
//
// # Por que nao o mutex simples que estava aqui
//
// Ate 2026-08-26 esta struct era um sync.Mutex segurado durante toda a carga.
// Quem chegava durante ela esperava em mu.Lock() PURO, sem select em
// ctx.Done(): o prazo que o host mandou era ignorado. Com cache frio ou
// corrompido, a tokenizacao do cofre inteiro roda por minutos sem resposta e
// sem erro — e o comentario que justificava o mutex dizia "a carga a partir do
// cache mede bem abaixo de 1 s", o que e verdade so COM cache valido, que e
// exatamente o caso em que ninguem espera.
//
// Agora a espera e uma PORTA. Quem chega durante a carga faz select entre a
// porta e o proprio ctx, e desiste com o erro do context — que o chamador
// converte em INDEX_BUILDING, um codigo que o host sabe reconsultar.
//
// A carga NAO e cancelada quando um espectador desiste: ela roda com o ctx de
// quem a disparou. Amarrar o trabalho ao primeiro chamador faria a proxima
// busca recomecar do zero, trocando uma espera longa por varias.
type cargaUnica struct {
	mu     sync.Mutex
	pronta bool
	// porta e fechada quando a carga em andamento termina, com exito ou nao.
	// nil quando nao ha carga em andamento.
	porta chan struct{}
	// erro guarda o resultado da ultima carga, para quem esperou na porta
	// receber a mesma resposta de quem a disparou.
	erro error
}

// fazer roda f() se ainda nao rodou com exito.
//
// Devolve o erro de f() sem marcar pronta, para que a proxima chamada tente de
// novo. Quem chega durante uma carga em andamento espera a porta ou o ctx, o
// que vier primeiro.
func (c *cargaUnica) fazer(ctx context.Context, f func(context.Context) error) error {
	c.mu.Lock()
	if c.pronta {
		c.mu.Unlock()
		return nil
	}
	if c.porta != nil {
		// Ja ha carga em andamento: espera, sem disparar uma segunda.
		porta := c.porta
		c.mu.Unlock()
		select {
		case <-porta:
			c.mu.Lock()
			err := c.erro
			c.mu.Unlock()
			return err
		case <-ctx.Done():
			// A carga segue em segundo plano, de proposito.
			return ctx.Err()
		}
	}

	// Esta chamada e a dona da carga.
	porta := make(chan struct{})
	c.porta = porta
	c.mu.Unlock()

	err := f(ctx)

	c.mu.Lock()
	c.erro = err
	if err == nil {
		c.pronta = true
	}
	c.porta = nil
	c.mu.Unlock()
	close(porta)

	return err
}

// garanteIndiceDeBusca dispara o carregamento do indice invertido na
// primeira busca, e so nela.
//
// Buscas concorrentes durante o carregamento esperam a MESMA carga, mas
// respeitando o proprio prazo: quem desiste recebe o erro do context, que
// search.go converte em INDEX_BUILDING. A carga continua.
//
// carregarBusca nil (modo --eager-search, ou um Service montado em teste
// com o indice ja pronto ou ja em construcao por outro caminho) faz desta
// funcao um no-op: nao ha o que carregar sob demanda.
func (s *Service) garanteIndiceDeBusca(ctx context.Context) error {
	if s.carregarBusca == nil {
		return nil
	}
	return s.cargaBusca.fazer(ctx, s.carregarBusca)
}
