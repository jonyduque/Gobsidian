---
title: Os dois índices
type: concept
status: active
description: Metadados e busca são estruturas separadas, com custos e ciclos de vida diferentes.
source_paths:
  - internal/index/index.go
  - internal/search/inverted.go
  - internal/search/soa.go
source_commit: b2be492
tags: [indice, busca, memoria]
language: pt-BR
updated_at: '2026-08-16'
---

# Os dois índices

O servidor mantém **duas** estruturas em memória, e confundi-las é a origem de
metade das perguntas sobre este código.

| | Índice de metadados | Índice de busca (invertido) |
|---|---|---|
| Pacote | `internal/index` | `internal/search` |
| Guarda | título, tags, aliases, headings, blocos, links, offsets | termo → (nota, posições) |
| Custo | proporcional ao **número de notas** | proporcional aos **bytes** do cofre |
| Tempo de montagem | ~1 s num cofre real | minutos, num cofre de 109 MB |
| Quando fica pronto | antes de anunciar as tools | em segundo plano, ou na 1ª busca |
| Cache | `index_cache.gob` | `inverted_cache.gob` |

A diferença de custo é o motivo de existirem separados: o host desiste do
handshake MCP em 30 s, e esperar a tokenização do cofre significava morrer antes
de anunciar qualquer coisa. Hoje o índice de metadados sozinho já sustenta 11
das 12 tools; só `vault_search` precisa do outro.

## O índice de metadados

`index.Index` são sete mapas sob um `sync.RWMutex`: `notes`, `assets`,
`lowerPath`, `byName`, `byAlias`, `backlinks`, `tags`, mais `citantesPorNome`
(índice reverso que permite reindexar um arquivo sem varrer o cofre).

**Invariante central: um `*Note` publicado em `ix.notes` é imutável.** Quem
precisa mudar uma nota troca a entrada do mapa por uma cópia — é o que
`mutarNotaLocked` faz. O mutex protege o *mapa*; o `*Note` escapa por `Get` e por
`List`, e o chamador lê os campos depois de soltar o `RLock`. Mutar no lugar é
escrever no que outra goroutine está lendo, com trava ou sem.

## O índice de busca: base + delta

`search.Inverted` tem **duas camadas**:

- **base** — arrays achatados, imutáveis, vindos do cache (`soa.go`).
- **delta** — mapas pequenos, com o que mudou desde a partida.

Toda leitura consulta as duas e o delta ganha. Toda escrita vai só para o delta e
marca o caminho como substituído no base (`sombra`). Quando o índice é construído
do zero, `base` é `nil` e o delta é o índice inteiro.

A representação anterior era `map[string]map[string][]TokenPosition`: no cofre de
referência, 126.342 mapas internos e 3 milhões de entradas. Achatar em arrays
tirou 35% do tempo de carga e a maior parte das alocações.

`sombra` garante que um caminho editado **nunca** devolva posting do base — se
devolvesse, uma nota editada apareceria na busca com o conteúdo que tinha na
partida.

## Os dois estados que não podem se confundir

`Inverted.building` (um `atomic.Bool`) diz que o índice **não cobre o cofre
inteiro ainda**. Uma busca nesse estado devolve `INDEX_BUILDING`, não zero
resultados: *cofre sem a palavra* e *índice ainda sem a palavra* não podem
produzir a mesma resposta.

O mesmo raciocínio governa `HasDoc` × `DocLength`: uma nota vazia tem
`DocLength == 0` e **está** indexada. Confundir os dois fazia o boot reler as
notas vazias a cada retomada, para sempre.

## Ver também

- [Formato do cache](../entities/formato-do-cache.md)
- [Busca](../features/busca.md)
- [Fluxo de boot](../flows/boot.md)
