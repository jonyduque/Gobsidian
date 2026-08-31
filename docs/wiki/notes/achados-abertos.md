---
title: Achados da revisão de 2026-08-15 e da auditoria de 2026-08-25
type: note
status: active
description: As duas listas estão fechadas. Esta página registra o que fechou, o que foi rejeitado com fundamento e o pouco que segue aberto.
source_paths:
- docs/REVISAO-2026-08-15.md
- docs/SUGESTOES.md
- docs/ESTADO.md
- docs/OPERACAO.md
source_commit: f7de8e81
tags:
- revisao
- auditoria
language: pt-BR
updated_at: '2026-08-31'
---

# Achados da revisão de 2026-08-15 e da auditoria de 2026-08-25

Esta página se chamava "Achados em aberto" e listava, um a um, defeitos que hoje
não existem mais. **As duas listas estão fechadas.** O que ela guarda agora é o
que uma lista fechada ainda ensina: quais achados foram recusados, por quê, e
onde a recusa está provada.

Quem precisa do detalhe de um achado específico vai a
[`docs/REVISAO-2026-08-15.md`](../../REVISAO-2026-08-15.md) e a
[`docs/SUGESTOES.md`](../../SUGESTOES.md). O placar vive em
[`docs/ESTADO.md`](../../ESTADO.md) — **um fato, um lugar**; esta página não o
recopia.

## O placar, em uma linha

A auditoria de 2026-08-25 levantou **61 achados**: 58 corrigidos, **3 rejeitados
depois de verificados**, nenhum aberto.

A revisão de 2026-08-15 gerou as Tasks 104–123 e **não fechou inteira**. O ledger
registra 104, 105, 106 e 116; outras (110, 113, 118, 119, 120, 121, 122, 123)
foram entregues sob a numeração da auditoria, e o código as confirma. **Quatro
não foram entregues** — conferidas no código em 2026-08-31, não no ledger:

| Task | O que falta | Conferência |
|---|---|---|
| **107** | `vault_search` não devolve onde o casamento está no arquivo | `match_offset` não existe em lugar nenhum |
| **109** | `note_read.paths` é `[]string`, não objeto por item — não há como pedir offset diferente por nota do lote | `Paths []string` em `tools_read.go` |
| **112** | `note_outline` não existe | nenhuma referência no código |
| **114** | chaves do índice aplicam só caixa, sem NFC/NFD | `aliasKey` e `nomeChave` fazem `strings.ToLower` e mais nada |

As três primeiras são a superfície que causou o incidente de campo. A quarta
aparece em cofre sincronizado com macOS: `Capítulo` em NFD e em NFC são strings
diferentes, e `text.Normalize` e `parser.Slug` já normalizam enquanto as chaves
do índice não.

## Os três rejeitados — e por que a recusa é o registro que importa

Rejeitar achado é a decisão que mais precisa de prova, porque ela some no
silêncio: ninguém audita o que não foi feito.

**M1 estava prescrito ao contrário.** Ele pedia que o `note_delete` reescrevesse
as notas citantes antes de apagar. Fazer isso **esconde** exatamente os
wikilinks que quebram por causa do delete — que é a informação que o usuário
precisa ver.

**B15 pede guarda para um caso que o sistema de arquivos torna
indistinguível.** Um rename no disco **é** delete+create com bytes idênticos: no
nível de evento não há como separá-lo de uma cópia seguida de remoção. A guarda
de cardinalidade cobre o caso comum, e nenhum caminho reescreve conteúdo com base
nessa inferência.

**P11 foi rejeitado por uma sondagem mascarada — e a rejeição foi corrigida em
2026-08-31.** A sondagem original concluiu que o prefixo `\\?\` explícito era
guarda morta porque o pacote `os` do Go o aplica sozinho. Mas a máquina tem
`LongPathsEnabled = 1`, e um caminho **relativo** de 327 caracteres também passa
— o que o `fixLongPath` do Go **não** explica, porque ele só atua em caminho
absoluto. A prova de mutação é inconclusiva pelo mesmo motivo: mediu o mesmo
ambiente. A metade do P11 que era defeito de verdade está corrigida em qualquer
máquina — `SweepStaleTempFiles` descartava erro de subárvore em silêncio e agora
devolve `SweepResult{Removidos, NaoRemovidos, Inacessiveis}`, os três logados no
boot. **Segue sem verificação** se o prefixo explícito é necessário com a chave
desligada. Detalhe e sondagens em
[`docs/OPERACAO.md`](../../OPERACAO.md), seção "Correção da rejeição do P11".

> A lição não é sobre MAX_PATH. É que **uma sondagem que roda num único ambiente
> mede o ambiente junto com o código**, e a prova de mutação herda o vício.

## O que a parte de desempenho virou

A revisão listou desempenho como **hipótese, não medição** — e estava certa em
dizer isso. Depois de medidas, as hipóteses não sobreviveram inteiras: o perfil
mostrou que o BM25 valia 16% da CPU da busca, não a maior parte. A
**Oportunidade 1** (pontuar em espaço de IDs densos) subsumiu P1–P3 e entregou
**−31% a −45%** no tempo de busca com paridade de ranking verificada. Os números,
com condições e protocolo, estão em [`docs/OPERACAO.md`](../../OPERACAO.md).

## O que segue aberto

As quatro tasks da tabela acima, mais o que está em
[`docs/ESTADO.md`](../../ESTADO.md) § Dívidas abertas: o `measure.ps1` fora de
gate, e a folga do RNF-07, cujo caminho medido é o índice de metadados (67% do
heap vivo, com `Link.Raw` repetindo `Link.Target` em 100,0% dos links).

## Ver também

- [Medições de 2026-08-15](medicoes-2026-08-15.md) — a única parte medida da
  revisão, e o que continuava sem medição.
- [Incidente de campo de 2026-08-15](incidente-de-campo-2026-08-15.md) — a falha
  observada em produção e o que a causou.
- [Armadilhas já pagas](../risks/armadilhas-pagas.md) — os defeitos escritos para
  não voltarem.
