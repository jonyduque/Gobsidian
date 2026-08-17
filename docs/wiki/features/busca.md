---
title: Busca
type: feature
status: active
description: Tokenização, ranking BM25, recorte de trecho e o cache de duas camadas.
source_paths:
  - internal/search/analyzer.go
  - internal/search/bm25.go
  - internal/search/inverted.go
  - internal/search/snippet.go
  - internal/search/snippet_cache.go
  - internal/service/search.go
source_commit: b2be492
tags: [busca, bm25, ranking]
language: pt-BR
updated_at: '2026-08-16'
---

# Busca

`vault_search` é a única tool que precisa do índice invertido, e a única que toca
o disco depois de rankear.

## O caminho de uma consulta

```
service.Search
 ├─ garanteIndiceDeBusca      carrega o índice na 1ª busca (modo padrão)
 ├─ recusa se inverted.Building()   → INDEX_BUILDING, nunca "zero resultados"
 ├─ search.Analyze(consulta)  → tokens
 ├─ search.CalculateBM25      → []Result ordenado por score
 ├─ filtros (pasta, tags, frontmatter, data, frase exata)
 ├─ paginação (offset/limit, teto de 200)
 └─ pool de até 8 goroutines recortando trechos do disco
```

## Tokenização

`search.Analyze` extrai sequências alfanuméricas com **offsets exatos em bytes** e
produz, para cada token, duas formas:

- **`Raw`** — normalizada: sem acento, minúscula.
- **`Reduced`** — redução conservadora para português (8 regras: `-coes → -cao`,
  `-oes → -ao`, `-ais → -al`, plural simples etc.). Vazia quando igual à crua.

As duas entram no índice. No ranking, a forma crua pesa mais que a reduzida
(`WeightRaw` 1.5 contra `WeightReduced` 1.0), para que busca por termo exato
receba peso superior ao termo derivado.

## Ranking

BM25 com `k1 = 1.2`, `b = 0.75`, mais pesos por campo: **título 3.0, heading 2.0,
corpo 1.0**. O campo é decidido em `getFieldWeight`, que compara a posição do
token contra a linha de cada heading da nota.

Consulta entre aspas duplas é **frase exata**: os tokens têm de aparecer em
sequência na nota, verificado por `matchPhraseInNote` sobre as posições.

## Recorte de trecho

`GenerateSnippet` lê **só a janela** em volta da ocorrência escolhida, via
`vault.ReadRange`. Três cuidados nele:

- **Arquivo somente-nuvem nunca é aberto** — o placeholder do OneDrive sai antes,
  com trecho vazio. Abrir dispararia download síncrono.
- **Offset do BOM**: as posições do índice são relativas ao corpo sem BOM; se a
  nota tem BOM no disco, os offsets deslocam +3 bytes.
- **UTF-8 parcial nas bordas** é aparado, e o destaque é recalculado relativo ao
  trecho.

O `SnippetCache` é um LRU de 1.024 entradas, chaveado por
`(caminho, hash da nota, ocorrência, maxChars)`. **Não há `Invalidate()`, e isso é
decisão**: nota editada muda de hash, logo muda de chave, logo a entrada velha
morre por LRU. Nota fora do índice de metadados não tem hash, logo não é cacheada.

## O pool de 8 trabalhadores

`maxSnippetWorkers = 8` é **medido, não escolhido por gosto**. Uma varredura de
1/4/8/16 trabalhadores em três regimes mostrou que 16 é *pior* que 8 em duas
escalas — 12 núcleos não absorvem 16 leitores. Re-aferido em 2026-08-12 com o
regime já mudado, e continuou 8: mudança sem ganho significativo é dívida pura.

## Duas garantias que o código defende

**Página parcial não sai como sucesso.** Se o contexto for cancelado no meio do
recorte, `Search` devolve erro. Antes disso, um `ctx` cancelado devolvia entre 24
e 99 dos 200 resultados com `err == nil` e `Total` intacto.

**Índice em construção não devolve lista curta.** Ver
[Os dois índices](../concepts/os-dois-indices.md).

## Ver também

- [Formato do cache](../entities/formato-do-cache.md)
- [Parser](parser.md) — de onde vêm os headings que o peso de campo consulta.
- [As 12 tools MCP](tools-mcp.md) — o schema de `vault_search`.
- [Achados em aberto](../notes/achados-abertos.md) — desempenho e precisão do
  recorte têm itens abertos.
