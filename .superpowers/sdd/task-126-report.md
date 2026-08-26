# Task 126 — prazo estourado que aponta para a causa

**Status:** DONE_WITH_CONCERNS — **o escopo mudou porque duas premissas do brief eram falsas.** Ver "Correção de premissa".
**Commit:** `499e744` — `fix(daemon): make silent failures diagnosable, and repair scan root handling`

---

## Correção de premissa — leia antes do resto

O brief mandava:

1. *"Substituir o probe único por espera escalonada"*
2. *"A decisão passa a ser o handshake, não o errno"*

**As duas já estavam implementadas.** Ao procurar a âncora de mutação, `internal/daemon/lock.go:204-227` mostrou que `esperarSocket` **já** fazia um laço de `DialAndHandshake` com `pollInterval = 25ms` e `dialProbeTimeout = 200ms`, até o prazo — ou seja, ~40 sondagens por segundo por 10 s, **usando handshake como critério**, nunca errno.

Escrever o item 1 teria sido reimplementar o que existia; escrever o item 2, mudar nada. O brief partia de uma leitura errada minha do código, e a verificação da âncora foi o que expôs isso.

**O buraco real é outro**, e a evidência de campo o nomeia: a mensagem

```
msg="nao foi possivel iniciar o daemon; servindo em processo"
     err="socket do daemon nao respondeu em 10s: ..."
```

culpa **o socket** — o único lugar onde a resposta não está. Nos logs de 25/08 não havia **nenhuma** linha `"daemon iniciado"` para aqueles cofres: o daemon não morreu, não nasceu. Quem lia a mensagem não tinha como distinguir isso de "subiu e morreu na montagem".

---

## Evidência de TDD

### GREEN

```
$ go test -run 'TestPistaDoLog|TestUltimasLinhasDoLog' -v ./internal/daemon/

=== RUN   TestPistaDoLogDistingueAusenteDeMorteNaMontagem
=== RUN   TestPistaDoLogDistingueAusenteDeMorteNaMontagem/ausente
=== RUN   TestPistaDoLogDistingueAusenteDeMorteNaMontagem/morreu_na_montagem
--- PASS: TestPistaDoLogDistingueAusenteDeMorteNaMontagem (0.02s)
=== RUN   TestUltimasLinhasDoLogDevolveOFim
--- PASS: TestUltimasLinhasDoLogDevolveOFim (0.02s)
PASS
ok  	github.com/jonyd/gobsidian/internal/daemon	1.664s
```

O teste separa os **dois defeitos diferentes com o mesmo sintoma**, ambos ocorridos na máquina do dono:

- **log ausente** → o processo não chegou a escrever nada; o problema é o spawn, não o cofre.
- **log com `"daemon iniciado"` e mais nada** → subiu e morreu na montagem do serviço.

---

## Prova de mutação

```
pwsh -File scripts/mutate.ps1 -Path internal/daemon/log.go `
  -Anchor 'return "o log do daemon nao existe: o processo nao chegou a escrever nada (falha de spawn, nao do cofre)"' `
  -Replacement 'return "sem informacao"' `
  -Test TestPistaDoLogDistingueAusenteDeMorteNaMontagem -Package ./internal/daemon/
```

Saída:

```
        log_test.go:46: pista = "sem informacao"; queria apontar o spawn como suspeito
FAIL
FAIL	github.com/jonyd/gobsidian/internal/daemon	1.754s
----------------------------------------------------------------------
[OK] internal/daemon/log.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

---

## O que entrou

- `daemon.CaminhoDoLog` e `daemon.UltimasLinhasDoLog` em `internal/daemon/log.go`.
  **Consolidação:** o caminho do log do daemon era calculado em **três** lugares — `cmd/gobsidian/daemon.go`, `internal/daemon` e `internal/doctor`. Agora é um. `daemonLogPath` do `cmd` passou a delegar.
- `pistaDoLog`, colada ao erro de prazo estourado em `esperarSocket`.

---

## Medição

Efeito do reparo de ambiente que esta investigação produziu, no cofre Estudo (2.557 notas, cache quente, binário instalado v1.2.1):

| | antes | depois |
|---|---|---|
| tempo até a ponte decidir | **10 s** | **559 ms** |
| desfecho | `servindo em processo` | `conectado ao daemon recem-iniciado via socket` |
| daemons vivos na máquina | **0** | **3** |

O "antes" vem dos logs do host de 25/08; o "depois" foi medido em 26/08 após remover os `.sock` órfãos e corrigir o caminho do cofre no config.

---

## Verificações

1. **O fallback em processo continua obrigatório nos três pontos** (decisão 2 da Task 91, mantida pela 92). Esta tarefa não mudou quando ele dispara.
2. **A goroutine vigia de `ln.Close()`** ficou como estava.
3. `pwsh -File scripts/verify.ps1`: **14 de 14 [OK]**.

---

## O que ficou de fora

- **Os três RED do brief não foram escritos**, porque testavam comportamento que já existia (espera escalonada, critério de handshake). Escrevê-los seria cobertura nova sobre código velho — legítimo, mas não é o que a tarefa foi despachada para fazer, e não foi feito.
- **O item 4 do brief** (`cmd/gobsidian/daemon.go` participar do mesmo lock de `EnsureStarted`) **não entrou**. Continua aberto, junto do item 11 da Fase 4.
- **O teste da tempestade** (≥10 pontes simultâneas ⇒ exatamente um daemon) não foi escrito.
- **A causa raiz do `10022` continua desconhecida.** Reproduzi `10022` apenas com um *diretório* no caminho do socket; os arquivos de campo eram `FileInfo + ReparsePoint`, e socket órfão de dono morto à força devolve `10061`. A cadeia não fecha. O que existe agora é o registro do errno no log (Task 124c) e a classificação no `doctor` (Task 125), para a próxima ocorrência se explicar sozinha.
