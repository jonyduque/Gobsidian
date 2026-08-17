---
title: Decisões fechadas
type: decision
status: active
description: Escolhas com trade-off registrado, que não devem ser re-litigadas sem dado novo.
source_paths:
  - internal/ipc/ipc.go
  - internal/config/config.go
  - internal/search/inverted.go
  - cmd/gobsidian/servico.go
source_commit: b2be492
tags: [adr, trade-offs]
language: pt-BR
updated_at: '2026-08-16'
---

# Decisões fechadas

As decisões normativas com identificador (D1–D13, AD-01–AD-09, D-M7-*) vivem em
`docs/PRD.md` e `docs/ARCHITECTURE.md`. Esta página registra as que mais afetam a
leitura do código e o motivo de cada uma.

## Transporte: AF_UNIX, não named pipe (D-M7-6)

**Medido**, ida e volta, 20.000 repetições:

| payload | AF_UNIX | named pipe |
|---|---|---|
| 256 B | 25,7 µs | 82,9 µs |
| 4 KB | 23,0 µs | 93,5 µs |
| 64 KB | 42,9 µs | 110,0 µs |

Está na biblioteca padrão e é o mesmo código nos três sistemas; build tag só para
o caminho do socket e a limpeza. **Não se re-litiga.**

## Carga preguiçosa do índice de busca

O modo padrão carrega o índice **na primeira `vault_search`**, não no boot.
`--eager-search` liga o comportamento antigo.

Motivo: o host desiste do handshake MCP em 30 s, e o carregamento do cache mede
1,83 s contra 1,3 s do índice de metadados — mais da metade do tempo até o
servidor responder, gasto numa estrutura que nenhuma tool precisa para ser
anunciada. Ver [Fluxo de boot](../flows/boot.md).

## `--debounce-ms=0` é recusado na configuração

Sem coalescência, cada evento vira um lote de um caminho só, e a correlação de
rename — que exige uma remoção **e** uma criação no mesmo lote — para de detectar
qualquer rename. Servidor sem debounce não é configuração que se possa pedir por
engano, então a recusa mora na config e não no watcher.

## Contadores de descarte publicados desdobrados por motivo

`vault_stats` expõe `events_dropped_by_reason`. Um contador único não permite
distinguir ruído esperado (chmod) de nota sumindo (fora do cofre, excluída).

## Persistir os dois caches

`docs/PRD.md` Q3 decidiu persistir **dois**. O do índice de metadados entrou na
Task 85, e o boot com cache válido caiu de 1.192–1.396 ms para 371–472 ms num
cofre real.

`byAlias`, `backlinks` e a resolução de cada link **não** vêm do arquivo: são
recalculados chamando as mesmas três funções que `Build` chama. Persistir isso
separadamente seria uma segunda forma de calcular o mesmo dado — e a lição
registrada é que a forma menos usada é a que diverge.

## `index.MoveNote` fica

Entrou fora do contrato e pagou as dívidas que contraiu — hoje respeita a
invariante do `*Note` imutável, atualiza `citantesPorNome` e reprocessa só os
citantes afetados.

## RNF-30 mudou de redação, não de intenção

Era "nenhum socket"; é **"nenhum socket que saia da máquina"** (Task 90,
autorizado pelo dono em 2026-08-05). Ver
[Regras não negociáveis](../risks/regras-nao-negociaveis.md).

## `GOGC` testado duas vezes e rejeitado nas duas

`GOGC=off` deu `~` (p=0,093, n=6). `GOGC=400` deu −28,51% no benchmark mas **não
significativo no boot real** (12 partidas por braço, U de Mann-Whitney 88 contra
região crítica 37/107) e com RSS pior.

O que pagou foi `debug.FreeOSMemory()` depois de o índice ficar pronto: −195 MB.

> Re-aferido parcialmente em 2026-08-15 sob o toolchain 1.26: `GOGC=400` melhora
> `IndexBuild` em 10,18% (p=0,002) e não move a busca; `GOGC=off` **piora** a
> busca em 39,55% (p=0,002). Isso reproduz só a metade que nunca esteve em
> disputa — o boot real e o RSS continuam sem re-aferição. **A rejeição segue de
> pé.** Ver [Medições de 2026-08-15](../notes/medicoes-2026-08-15.md).

## Task 82 revertida

`benchstat` deu `~`. **Mudança sem ganho significativo é dívida pura.**

## Ver também

- `docs/PRD.md` (D1–D13) · `docs/ARCHITECTURE.md` (AD-01–AD-09)
