---
title: Formato do cache de busca
type: entity
status: active
description: Codec binário próprio, arrays achatados e arena mapeada em memória.
source_paths:
  - internal/search/persist_codec.go
  - internal/search/persist.go
  - internal/search/soa.go
  - internal/search/mmap.go
source_commit: f7de8e81
tags: [cache, formato, mmap]
language: pt-BR
updated_at: '2026-08-31'
---

# Formato do cache de busca

**Formato 6, e não é `gob`.** A extensão do arquivo continua `.gob` por
compatibilidade de caminho; o conteúdo é um codec binário próprio.

## Por que saiu do `gob`

Medido num cofre real de 3.148 notas e 109 MB, cujo cache dava 471,6 MB:

```
carregamento     5,03 s/op   3,94 GB/op   12.797.583 allocs/op
  reflexão do gob  ~3,6 s     72%
  materialização    1,40 s    28%
```

Os 471,6 MB eram duas coisas: **286,3 MB de caminhos repetidos** (2,96 milhões de
pares termo-documento × ~101 chars) e **272,4 MB de posições** (17,8 milhões × 16
bytes fixos).

O formato ataca as duas: o caminho vira índice numa tabela escrita uma vez, e a
posição vira varint sobre o **delta** da anterior. Nada passa por reflexão.

Medido na virada do formato 4 para o 5: carregamento **5,59 s → 659 ms**, arquivo
**482 → 67 MB**, boot quente **~7 s → 842 ms**.

## Layout

```
magic      4 bytes  "GBS6"
versões    3 varints  (formato, parser, analisador)
vaultPath  string
noteCount, totPost, totPos   varints
nCaminhos  varint + nCaminhos strings          ← ordenados
nDocs      varint + pares (pathID, tamanho)
nTermos    varint                              ← ordenados
  por termo:   string, nPostings
    por posting: pathID, nPos                  ← pathID crescente
      por posição: delta(Start), (End-Start)
[seção fixa de posições, alinhada em 8]
rodapé     24 bytes: "GBSARENA" + offset + count   ← little-endian, sem varint
```

**A ordem não é decorativa.** `termos` e `caminhos` ordenados são o que permite
busca binária no lugar de mapa, e `postPath` ordenado dentro de cada termo é o que
faz `Postings` devolver resultado já ordenado por caminho.

O leitor confere as três ordens (`validaOrdem`) porque um arquivo fora de ordem
**não falharia**: `sort.SearchStrings` devolveria "termo não existe" para termos
que existem, e a busca pararia de achar notas — sem erro, sem log, com cara de
cofre que simplesmente não tem a palavra.

## Tetos contra arquivo corrompido

`limiteCaminhos`, `limiteTermos`, `limitePostings`, `limitePosicoes`,
`limiteString`. Sem eles, um comprimento adulterado — ou um byte trocado por
corrupção de disco — vira `make([]T, 4_000_000_000)` e o processo morre por OOM em
vez de devolver "cache corrompido" e reconstruir.

## A arena mapeada (formato 6)

O array de posições é ~291 MB dos ~382 MB em repouso de uma instância. Mapeado do
arquivo em modo leitura, o cache de páginas do sistema operacional **compartilha
as mesmas páginas físicas** entre processos que abrem o mesmo arquivo — várias
instâncias pagam a arena uma vez.

O rodapé é fixo e fica no fim justamente para quem só quer mapear a arena achar o
offset sem decodificar o corpo inteiro em varint.

`tentaAbrirArena` recusa mapear quando não é seguro, e um dos motivos é
`dentroDoCofre`: um `--cache-dir` apontado para dentro do cofre por engano nunca
deve resultar em mapear um arquivo que o OneDrive pode reescrever por baixo do
processo.

`promoverArenaSePresente` copia `pos` para o heap e desfaz o mapeamento antes de
um `os.Rename` de regravação — renomear por cima de um arquivo mapeado falha no
Windows com `ERROR_SHARING_VIOLATION`.

## O outro cache, e por que a versão dele é um alias

São **dois** caches independentes com **duas** versões: o índice de busca está no
formato **6** (`inverted_cache.gob`, magic `GBS6`) e o de metadados no **5**
(`index_cache.gob`, magic `GIC1`). Subir um não sobe o outro, e não deve.

Dentro do de metadados, porém, a versão é **um alias, não uma cópia**:

```go
indexCacheCodecVers = IndexCacheFormatVersion
```

Até 2026-08-26 eram duas constantes independentes guardando o **mesmo** portão —
`persist.go` conferia a sua em `LoadIndexCache`, o decodificador conferia a sua no
cabeçalho — com o mesmo valor por coincidência. Subir uma sem a outra não quebra
build nem teste: faz o leitor **recusar todo save que o próprio processo acabou de
gravar**, com reconstrução completa a cada boot e nenhum log dizendo por quê
(achado B11). O alias existe para que o bump seja impossível de fazer pela metade.

## Custo de trocar de formato

**Toda troca reconstrói o cache de todo cofre no boot seguinte**, em segundo
plano, com as outras onze tools respondendo desde o primeiro segundo.

Vale também para quem atualizar de uma v1.0.x, que grava `gob` e invalida o cache
desta versão a cada alternância.

## Ver também

- [Os dois índices](../concepts/os-dois-indices.md) · [Busca](../features/busca.md)
