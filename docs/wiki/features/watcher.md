---
title: Watcher
type: feature
status: active
description: Fachada sobre fsnotify — filtro, debounce, correlação de rename e reconciliação.
source_paths:
  - internal/watcher/watcher.go
  - internal/watcher/filter.go
  - internal/watcher/debounce.go
  - internal/watcher/apply.go
  - internal/watcher/rename.go
  - internal/watcher/overflow.go
  - internal/watcher/counters.go
source_commit: b2be492
tags: [watcher, fsnotify, incremental]
language: pt-BR
updated_at: '2026-08-16'
---

# Watcher

Mantém o índice atualizado sem reindexar tudo. **Nenhum tipo do fsnotify cruza
para fora deste pacote.**

São três camadas de filtragem antes de qualquer trabalho real, mais uma rede de
segurança.

## 1. Relevância

`filter` decide se o evento interessa e, quando descarta, **diz por quê** —
`DropChmod`, `DropOutsideVault`, `DropExcluded`, `DropUnknownOp`. Os quatro
motivos viram contadores desdobrados em `vault_stats`: um contador único de
descarte não permite distinguir ruído esperado de nota sumindo.

A regra de relevância é `vault.Classify`, a **mesma** que a varredura consulta.
Duas cópias da regra divergiriam, e a que fica na cópia menos usada é a que
ninguém percebe.

## 2. Debounce

Tique único com conjunto sujo: eventos que chegam dentro da janela
(`--debounce-ms`, padrão 250) são coalescidos num lote de caminhos distintos. Por
isso `--debounce-ms=0` é **recusado na configuração** — sem coalescência, cada
evento vira um lote de um caminho só, e a correlação de rename (que exige uma
remoção **e** uma criação no mesmo lote) para de detectar qualquer rename.

## 3. Mudança real

`Apply` compara `mtime` e tamanho contra o que o índice já tem, e pula se
bateram. Isso não é prova de que o conteúdo não mudou — uma reescrita dentro do
mesmo tique preservando o tamanho passa despercebida. É aceitável porque há dois
anteparos: a reconciliação por overflow e a reindexação no boot.

## Correlação de rename

`CorrelateRenames` casa remoção + criação no mesmo lote por **xxhash do
conteúdo**. Casando, chama `index.MoveNote` em vez de remover e reinserir — o que
preserva backlinks sem reler o disco.

Ela respeita as mesmas duas regras que `index.Replace` respeita: anexo não é
aberto, placeholder de nuvem não é aberto. **Quem roda antes do guarda precisa do
mesmo guarda** — essa lição foi paga aqui.

## Reconciliação por overflow

Se a fila do sistema operacional estourar, o fsnotify emite `ErrEventOverflow` e
eventos se perdem. `handleFSError` agenda **uma** reconciliação (canal com buffer
1, `default` descarta a segunda), e `Reconcile` faz uma varredura completa
comparando disco contra índice.

Ela repara **os dois índices**. Reparar só o de metadados deixava o de busca
obsoleto, e como `service.Search` descarta posting cujo caminho não está nos
metadados, uma nota movida durante o overflow devolvia zero resultados para
sempre.

## Diretório novo

Uma pasta que chega ao cofre **já com arquivos dentro** entrega *um* evento — a
criação do diretório — e nenhum arquivo, porque eles existiam antes do watch
existir. Medido: 3 notas, 1 evento, 0 indexadas.

Por isso todo `Add` de watch é seguido de `varreDiretorioNovo`, que emite eventos
sintéticos pelo **mesmo** filtro e os mesmos contadores. É também o que fazia
`note_move` perder a nota.

## Ordem de partida

`watcher.New` roda **antes** da construção do índice de busca; `w.Run` roda
depois. New registra os watches, então os eventos do período de construção ficam
enfileirados no fsnotify em vez de se perderem.

## Ver também

- [Fluxo de boot](../flows/boot.md)
- [Armadilhas já pagas](../risks/armadilhas-pagas.md)
