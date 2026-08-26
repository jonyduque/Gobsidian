# Task 134 — A6: a primeira busca passa a respeitar o prazo de quem espera

**Status:** DONE
**Commit:** `7337e78` — `fix(service): make the first search wait cancellable, closing A6`

---

## O defeito

`internal/service/search_lazy.go` era um `sync.Mutex` segurado durante **toda** a
carga do índice invertido. Quem chegava durante ela esperava em `mu.Lock()`
**puro** — sem `select` em `ctx.Done()`. O prazo que o host mandou era ignorado.

O comentário que justificava o desenho dizia:

> a carga a partir do cache mede bem abaixo de 1 s

Isso é verdade **com cache válido** — que é exatamente o caso em que ninguém
espera. Com cache frio, ausente ou corrompido, o caminho cai em
`buildInvertedIndex`, e num cofre real de 109 MB a tokenização mediu **219 s**.
Durante esses minutos, toda `vault_search` concorrente ficava parada, **sem
resposta e sem erro** — o host não tinha como saber se o servidor estava vivo.

A justificativa media o caso barato e o desenho pagava o caro.

## O desenho

O mutex vira uma **porta**. Quem chega durante a carga faz `select` entre a porta
e o próprio `ctx`, e desiste com o erro do context — que `search.go` converte em
`INDEX_BUILDING`, um código que o host sabe reconsultar.

```go
if c.porta != nil {
	porta := c.porta
	c.mu.Unlock()
	select {
	case <-porta:
		c.mu.Lock(); err := c.erro; c.mu.Unlock(); return err
	case <-ctx.Done():
		return ctx.Err()   // a carga segue em segundo plano
	}
}
```

### Duas decisões que não são detalhe

**A carga NÃO é cancelada quando um espectador desiste.** Ela roda com o `ctx` de
quem a disparou. Amarrar o trabalho ao primeiro chamador faria a busca seguinte
recomeçar do zero — trocaria uma espera longa por **várias** esperas longas. É a
decisão (b) que o dono fechou: orçamento no primeiro chamador, `INDEX_BUILDING`
com `Retryable` para os demais.

**Não é `sync.Once`.** `Once` marca "já disparei" mesmo quando a função falha, e
a falha vira **permanente**: nenhuma busca depois consegue tentar de novo. Um
cache corrompido derrubaria a busca até o próximo reinício. `cargaUnica` só marca
`pronta` depois de `f()` devolver `nil`.

## Provas

Três testes em `internal/service/carga_cancelavel_test.go`, um por metade do
contrato:

- `TestCargaConcorrenteRespeitaOPrazoDoChamador` — o concorrente volta com
  `context.DeadlineExceeded` e **não** dispara uma segunda carga.
- `TestCargaSegueEmSegundoPlanoAposOPrazo` — a carga original não morre junto com
  a desistência; e depois de concluída, uma busca nova não recarrega.
- `TestCargaPermiteNovaTentativaAposFalha` — falha não marca `pronta`. Já existia
  a semântica; o teste está aqui **de novo** porque o redesenho troca o mutex por
  uma porta, e uma porta fechada cedo demais tornaria a falha permanente.

### A mutação que fez o teste TRAVAR em vez de reprovar

A primeira mutação substituiu a porta por `<-make(chan struct{})`, que bloqueia
para sempre. O teste **não reprovou: pendurou** — e um teste pendurado não nomeia
falha nenhuma, ele só estoura o timeout do pacote inteiro.

É exatamente a armadilha que `docs/ARMADILHAS.md` registra: mutação que troca o
modo de falha não prova a regra. Substituída por uma que preserva o modo:

```
-Anchor 'case <-ctx.Done():'  -Replacement 'case <-porta:'
    -> EXIT=0
```

Com ela o concorrente volta a esperar a carga inteira, e
`TestCargaConcorrenteRespeitaOPrazoDoChamador` reprova nomeando o defeito
original: *"o concorrente esperou a carga inteira em vez de respeitar o prazo"*.

## Documentação

O parágrafo de `cmd/gobsidian/servico.go` que explica o boot descrevia **só o
modo eager** — a auditoria acusou essa lacuna como parte do próprio A6. No modo
preguiçoso (o padrão) o índice também chega vazio, mas a carga só dispara na
primeira `vault_search`. O comentário agora cobre os dois modos e nomeia a porta.

## Verificações

1. `go test -race ./internal/service/`: verde.
2. `pwsh -File scripts/verify.ps1`: **14 de 14 [OK]**.

## O que ficou de fora

- **O número de `INDEX_BUILDING` em produção não foi medido.** Sei que o caminho
  existe e que o teste o exercita; quantas vezes um host real o recebe num boot
  frio de cofre grande, **não medido**.
