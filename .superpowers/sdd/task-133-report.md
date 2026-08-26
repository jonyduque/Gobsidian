# Task 133 — A5 + A4: prova de órfão, e um `Accept` que sobrevive

**Status:** DONE_WITH_CONCERNS — **esta tarefa expôs o defeito A5 acontecendo
dentro do gate de órfãos do próprio projeto, e corrigiu uma regressão de
desempenho que eu havia introduzido na Task 129.**
**Commit:** `570d556` — `fix(ipc,daemon): prove the socket is orphaned before unlinking, survive Accept`

---

## A5 — o socket não é desvinculado sem prova

`internal/ipc/ipc.go:104` chamava `cleanupSocketFile(path)` **incondicionalmente**
antes do `net.Listen`. Um segundo daemon removia o arquivo do primeiro e bindava
no mesmo nome: o daemon vivo ficava inalcançável, com índice em memória e
watcher rodando, e **as duas instâncias passavam a gravar concorrentemente no
MESMO cache de busca**.

### O critério NÃO pode ser o errno

Medido em 2026-08-26 com `net.Dial("unix", ...)` no Windows:

| Estado do caminho | errno |
|---|---|
| arquivo comum · socket órfão · caminho inexistente | `10061` ECONNREFUSED |
| **diretório** | **`10022`** EINVAL |

Errnos diferentes descrevem o mesmo estado, e o mesmo errno descreve estados
diferentes. **Isto corrige a prescrição do item 11 da Fase 4 do plano**, que
dizia *"`ECONNREFUSED` significa órfão e libera o unlink"*. O critério é
comportamental: alguém aceita conexão ali?

### `net.Dial`, e não `net.DialTimeout`

A primeira versão usou `net.DialTimeout` e o `check_net` reprovou:

```
internal\ipc\ipc.go:161:12: chamada de rede proibida: net.DialTimeout —
so net.Dial e net.Listen com rede "unix" sao permitidos
```

**É a garantia do RNF-30 funcionando.** Trocado por `net.Dial`, sem perda:
`connect()` num socket de domínio Unix resolve na hora, sem a espera de rede que
faz um dial TCP precisar de prazo.

### RED / GREEN

```
=== RUN   TestListenRecusaSocketDeDaemonVivo
    listen_orfao_test.go:50: o segundo Listen ROUBOU o socket de um daemon vivo
--- FAIL
=== RUN   TestListenLimpaSocketOrfao
--- PASS
```

O contrapeso já passava, e existe porque "nunca limpar" travaria toda partida
seguinte para sempre — que é o defeito de campo de 2026-08-26, com três sessões
pagando 10 s por partida durante dias.

Depois: os dois verdes, mais o teste que confirma que o arquivo do daemon vivo
**não foi tocado** na recusa.

---

## A4 — o laço de `Accept` não morre na primeira

`internal/daemon/daemon.go:152` fazia `log.Warn` e `return` em qualquer erro sem
cancelamento. O processo seguia vivo — socket bound, ticker rodando — e nenhuma
conexão era aceita nunca mais: os dials caem no backlog do SO e ninguém atende.

### O teto não é detalhe

Recuar e continuar sem limite trocaria "daemon surdo" por "daemon girando": um
erro **permanente** não classificado viraria laço quente consumindo CPU para
sempre. `maxFalhasSeguidasDeAccept = 5` conta falhas **consecutivas**; qualquer
accept bem-sucedido zera.

### O teste exercita produção, não uma réplica

A primeira versão reimplementava o laço no teste — a armadilha que este projeto
pune. `daemon.New` recebe o listener, então o dublê entra por **injeção**:

```
=== RUN   TestAcceptLoopSobreviveAErroTransitorio
    accept_transitorio_test.go:106: nenhuma conexao atendida depois de 1 erro(s)
    transitorio(s): o laco de Accept morreu e o daemon ficou surdo
--- FAIL
```

E a asserção não é "o dial conectou" — é que a **saudação chegou**, porque o SO
aceita no backlog mesmo sem ninguém atender.

---

## O gate de órfãos escondia o próprio A5

`daemon-idle` reprovou **15 de 15** ciclos depois da correção. A causa não era a
correção:

Os três cenários anteriores rodam `serve`, e **`serve` spawna um daemon — com a
ociosidade PADRÃO de 900 s**, não a curta que o cenário usa. Esse daemon
sobrevive ao cenário que o criou e segura o socket do cofre. Confirmado por
inspeção de processo:

```
  PID Cofre       IdleSeg Idade_min
42376 vault_small 900          9,4
```

Antes, o `daemon-idle` **roubava o socket dele** e o daemon de 900 s ficava vivo
e inalcançável até expirar. Era o defeito A5 rodando dentro do gate do projeto,
sem ninguém ver. Com a prova de órfão, o daemon do cenário recusa subir — e a
recusa está certa; o que faltava era a precondição, agora estabelecida (encerra
por PID, e só o que casa aquele cofre).

**Quase corrigi isso pela causa errada.** Uma âncora de edição não casou, o
script não alterou nada, e o cenário isolado passou — o que me fez concluir que
era resíduo de teste meu. Só a rodada completa dos quatro cenários mostrou a
causa sistemática.

Depois: **os quatro cenários verdes, 15 ciclos cada.**

---

## A regressão que eu tinha introduzido na Task 129

O teto de latência do RNF-04 estourou: p95 **28,5 ms** contra teto de 22 ms, três
rodadas seguidas.

Causa: a guarda de symlink do C2 rodava **duas vezes por leitura** — `ReadRange`
guardava **e** chamava `Open`, que também guardava. Com `limit=200`,
`GenerateSnippet` lê um arquivo por hit: 400 `os.Lstat` extras por busca.

Medido contra o baseline desta sessão:

| | média |
|---|---|
| baseline (antes de tudo) | 18,68 ms |
| guarda dupla | **26,55 ms (+42%)**, allocs +15,5% (p=0,000) |

**As medições intermediárias foram não-monotônicas** — 26,6 → 23,0 → 25,7 ms
*removendo* guardas —, o que é assinatura de ruído de máquina, não de sinal. Foi
isso que impediu de "consertar" pelo número errado. A conclusão veio da leitura
do código, não do número: `ReadRange` chama `Open`.

Corrigido para **uma guarda por caminho de leitura**: `Open` cobre `ReadRange`;
`ReadAll` tem a sua porque não passa por `Open`. O teto voltou a passar.

---

## Provas de mutação

```
A5: -Anchor 'if alguemEscuta(path) {' -Replacement 'if alguemEscuta(path) && false {'
    -> EXIT=0

A4: -Anchor 'if falhasSeguidas >= maxFalhasSeguidasDeAccept {'
    -Replacement 'if falhasSeguidas >= 1 {'
    -> EXIT=0
```

A segunda mutação reduz o teto a 1, que é o comportamento antigo — morrer na
primeira falha.

---

## Verificações

1. `pwsh -File scripts/check_net.ps1`: **EXIT=0** (RNF-30).
2. `pwsh -File scripts/test_orphans.ps1 -Cycles 15`: **quatro cenários verdes**,
   incluindo `daemon-idle` com `reason=idle` conferido.
3. `pwsh -File scripts/verify.ps1`: **14 de 14 [OK]**.
4. `golangci-lint` acusou `redefines-builtin-id` (variável chamada `real`) —
   corrigido. É a segunda vez que esse builtin me pega nesta sessão.
5. **A guarda de binário desatualizado do `test_orphans.ps1` disparou** e me
   obrigou a recompilar antes do gate. Funcionou exatamente como projetada.

---

## O que ficou de fora

- **O item 4 do brief da Task 126** — `cmd/gobsidian/daemon.go` participar do
  mesmo lock de `EnsureStarted` — continua aberto. A prova de órfão fecha a
  metade do "roubo do socket"; a janela do chegante tardio no lock é a outra.
- **O teste da tempestade** (≥10 pontes simultâneas ⇒ exatamente um daemon) não
  foi escrito.
- **A causa do `10022` visto em campo continua desconhecida.** Só um diretório
  no caminho do socket o reproduz, e os arquivos de campo eram
  `FileInfo + ReparsePoint`. A cadeia não fecha.
