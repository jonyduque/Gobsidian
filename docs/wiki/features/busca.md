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
source_commit: f7de8e81
tags: [busca, bm25, ranking]
language: pt-BR
updated_at: '2026-08-31'
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
corpo 1.0**. O campo é decidido em `pesoDeCampo`; `dentroDeHeading` acha o
heading que contém a posição do token por `sort.Search`, não por varredura.

**O peso de título é por termo inteiro, não por substring** (achado P2). O teste
era `strings.Contains`, e por ele uma nota chamada "Recursos" recebia o bônus de
título ao buscar "curso". A conferência agora varre `TitleNorm` — que já está
pré-computado — checando fronteira de palavra dos dois lados; a primeira tentativa
de correção chamava `Analyze` por nota candidata por termo e custou **+38%**, e
foi trocada por esta, que não custa nada mensurável.

**A pontuação roda em espaço de IDs densos** (Oportunidade 1, 2026-08-28). Cada
consulta mapeia os documentos que ela toca para `0..n-1` e acumula em slices, em
vez de manter mapas por termo. Entrega **−31% a −45%** no tempo de busca e
**−42% a −47%** em alocação, com **paridade de ranking verificada**: ordem
idêntica em seis consultas, maior diferença de score 3,55e-15. De brinde, o
`avgdl` deixou de ser O(N) por consulta — `SomaDocLen()` é memoizado contra um
contador de geração do índice.

Consulta entre aspas duplas é **frase exata**: os tokens têm de aparecer em
sequência na nota, verificado por `matchPhraseInNote` sobre as posições.

## Recorte de trecho

`GenerateSnippet` lê **só a janela** em volta da ocorrência escolhida, via
`vault.ReadRange`.

**A ocorrência escolhida é a mais densa, não a primeira** (achado M16).
`melhorJanela` passa duas pontas sobre as ocorrências ordenadas e ancora onde
mais termos distintos da consulta cabem na largura da janela. Ancorar na primeira
devolvia trecho que não mostrava a consulta — o usuário lia um parágrafo com uma
palavra e concluía que a busca errou.

Três cuidados no recorte:

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
