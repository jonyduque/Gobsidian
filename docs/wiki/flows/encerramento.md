---
title: Encerramento
type: flow
status: active
description: Os quatro mecanismos que garantem que o processo morre, e o desligamento por etapas.
source_paths:
  - internal/lifecycle/lifecycle.go
  - internal/lifecycle/stdin.go
  - internal/lifecycle/signals.go
  - internal/lifecycle/parent.go
  - internal/lifecycle/shutdown.go
  - cmd/gobsidian/serve.go
source_commit: b2be492
tags: [lifecycle, shutdown, orfaos]
language: pt-BR
updated_at: '2026-08-16'
---

# Encerramento

Um servidor MCP é subprocesso de um host que pode fechar de qualquer jeito.
Processo órfão segurando um índice de centenas de MB é o pior modo de falha do
produto — invisível e cumulativo.

`internal/lifecycle` tem **mecanismos independentes** que cancelam o mesmo
context raiz. A redundância é deliberada: cada um falha em cenários diferentes.

| Mecanismo | Dispara quando | `reason=` |
|---|---|---|
| EOF em stdin | o host fecha o pipe | `stdin-eof` |
| Sinal do SO | Ctrl+C, SIGTERM, CTRL_BREAK | `signal` |
| Morte do processo pai | o host morre sem fechar nada | `parent-gone` |
| Ociosidade (só daemon) | nenhum cliente por N minutos | `idle` |

Só o **primeiro** motivo fica registrado. Os demais chegam sobre um context já
cancelado, e sobrescrever apagaria a informação de diagnóstico que interessa.

## Duas sutilezas que custaram caro

**A vigília do pai precisa de `exitTime`, não só de creation time.** O Windows
mantém PID e creation time consultáveis muito tempo depois da morte do processo —
o próprio ato de consultar segura um handle. Comparar `(pid, created)` **nunca**
detecta pai morto; deixou 5 de 5 órfãos no primeiro teste ponta a ponta.

Em Unix, comparar o `ppid` capturado no startup, não a constante 1: sob
Docker+tini, systemd ou s6 o reaper não é PID 1.

**Goroutine parada em `Read` não é desenrolável por cancelamento de context.** Por
isso `watchStdin` fica **fora** do `WaitGroup` — incluí-la travaria `Wait()`
sempre que outro mecanismo disparasse primeiro e o stdin continuasse aberto. As
vigias de sinal e de pai entram, porque fazem `select` em `ctx.Done()`.

## Desligamento por etapas

`lifecycle.Shutdown` roda etapas nomeadas, cada uma com orçamento próprio, sob um
limite duro global (6 s no `serve`):

| etapa | orçamento |
|---|---|
| `in-flight` — espera o serve loop retornar | 3 s |
| `close-pipe` | 500 ms |
| `watcher` | 500 ms |

Ela recebe `ctx` e **descarta o cancelamento** com `context.WithoutCancel`: o
context raiz já está cancelado quando ela roda, então derivar os orçamentos dele
faria toda etapa nascer expirada. `WithoutCancel` preserva os valores e joga fora
só o cancelamento.

## O espelho de stdin

O monitor de EOF e o SDK precisam ler o mesmo stdin, e dois leitores no mesmo
descritor repartem os bytes e corrompem o JSON-RPC.

A saída é `mirrorReader`: o SDK lê do espelho, o lifecycle observa a cópia.
**`io.TeeReader` não serve** — ele copia bytes, e EOF não é byte. `mirrorReader`
faz `dst.CloseWithError(err)`, que é o que propaga o fim da leitura.

O espelho é **auxiliar**: falha de escrita nele não pode virar erro da leitura
principal — mataria uma sessão saudável por um motivo que o cliente não pode
resolver.

## Código de saída

`ctx.Canceled`, `io.EOF` e `io.ErrClosedPipe` no retorno do serve loop são
**encerramento normal**, saída 0. Duas detecções de EOF independentes correm — a
do SDK e a do lifecycle — e qual vence decide o valor. Tratar qualquer uma como
falha faz um host supervisor ver erro aleatório a cada desconexão limpa.

## O gate

`scripts/test_orphans.ps1 -Cycles 100` roda os quatro cenários, e cada um
**reprova se o `reason=` não for o do mecanismo que ele nomeia** — encerrar pelo
motivo certo por acidente não conta.

O cenário `parent-death` desconecta o EOF (cadeia keeper → host → servidor);
`signal` deixa tudo vivo e só manda CTRL_BREAK; `daemon-idle` usa `--idle-seconds`
curto. **O CI hoje roda só três dos quatro** — ver
[Achados em aberto](../notes/achados-abertos.md).

## Ver também

- [Daemon e ponte](../features/daemon-e-ponte.md)
- [Armadilhas já pagas](../risks/armadilhas-pagas.md)
