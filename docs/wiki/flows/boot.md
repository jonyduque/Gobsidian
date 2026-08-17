---
title: Fluxo de boot
type: flow
status: active
description: O que roda entre o host lançar o processo e as tools serem anunciadas.
source_paths:
  - cmd/gobsidian/serve.go
  - cmd/gobsidian/servico.go
  - cmd/gobsidian/ponte.go
source_commit: b2be492
tags: [boot, inicializacao]
language: pt-BR
updated_at: '2026-08-16'
---

# Fluxo de boot

```
runServe
 └─ servePonte                          decide ponte ou processo
     ├─ ipc.DialAndHandshake            daemon vivo? → copia bytes e acabou
     ├─ daemon.EnsureStarted            ganhou a corrida? lança um
     └─ serveEmProcesso                 fallback obrigatório
         ├─ lifecycle.New               os três mecanismos de encerramento
         ├─ construirServico            ⟵ o miolo
         └─ mcpsrv.New + srv.Serve
```

`construirServico` (`cmd/gobsidian/servico.go`) é compartilhada entre o modo em
processo e o daemon **de propósito**: as duas precisam da mesma sequência, e
sequência de boot construída em dois lugares tem o mesmo risco de divergência que
chave de mapa calculada em dois lugares.

## A sequência

**1. Índice de metadados.** Tenta o cache do disco. Só aceita quando
`VerifyFreshness` confirma que ele bate com o disco — mesma contagem de arquivos,
mesmo tamanho e mtime por arquivo. Qualquer divergência cai para `idx.Build`, que
varre e parseia. **Não há reparo parcial aqui**: reparar só o que mudou é
trabalho do watcher, não do boot.

**2. Varredura de temporários.** `SweepStaleTempFiles` remove o que escritas
interrompidas deixaram. É o único momento sem escrita em voo.

**3. Índice de busca, vazio e marcado em construção.** Nem a construção nem o
carregamento do cache bloqueiam o anúncio das tools.

**4. `watcher.New`** — registra os watches. Antes da construção do índice de
busca, para que os eventos do período fiquem enfileirados no fsnotify.

**5. `service.New`** e o log `servidor pronto`.

**6. Em segundo plano:** o índice de busca (no modo `--eager-search`) e depois
`w.Run`.

## Por que o índice de busca saiu do caminho de boot

O custo da tokenização é proporcional aos **bytes**, não à contagem de notas. Num
cofre real de 109 MB ela levou 219 s, contra 1,3 s do índice de metadados.

**O host desiste do handshake MCP em 30 s** e mata o processo — antes de
`SaveInvertedCache` rodar, então a tentativa seguinte recomeçava do zero. Toda
tentativa falhava pelo mesmo motivo, para sempre, e nada no log dizia isso:
"servidor pronto" só aparecia depois.

O carregamento do cache saiu pelo mesmo raciocínio numa escala menor: 1,83 s
contra 1,3 s do índice de metadados — mais da metade do tempo até responder,
gasto numa estrutura que nenhuma tool precisa para ser anunciada.

No modo padrão a carga acontece **dentro da primeira `vault_search`**, uma vez,
com retentativa se falhar (`service.cargaUnica`).

## Cache parcial

`invertedCacheState` decide o que fazer com o que veio do disco:

- **pronta** — `hdr.NoteCount >= idx.NoteCount()`: cobre o cofre inteiro.
- **retomar** — utilizável mas incompleto; serve de ponto de partida, e
  `buildInvertedIndex` só lê do disco o que `HasDoc` ainda não cobre.

A comparação é `>=` e não `==` porque notas apagadas deixam entradas velhas no
cache; as sobras não vazam para o resultado porque a busca só devolve o que o
índice de metadados confirma.

Durante a construção, o cache é gravado a cada **60 s** — é a única rede contra
encerramento abrupto, já que a saída por cancelamento não grava parcial.

## Memória transitória

`devolveMemoriaTransitoria` chama `debug.FreeOSMemory()` depois de o índice ficar
pronto. Montar o índice do cofre de referência aloca 737 MB para deixar ~500 MB
vivos; a diferença é lixo que o Go devolveria no tempo dele. Medido: **−195 MB**
no RSS em repouso, custo de 67–73 ms.

> Essa medição é anterior ao toolchain 1.26 (Green Tea GC) e **não foi
> re-aferida**. Ver [Achados em aberto](../notes/achados-abertos.md).

## Ver também

- [Os dois índices](../concepts/os-dois-indices.md) · [Encerramento](encerramento.md)
- [Watcher](../features/watcher.md) — por que `New` vem antes e `Run` depois.
- [Parser](../features/parser.md) — o que o worker pool de `Build` executa por nota.
