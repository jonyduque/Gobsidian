# Task 164 — Sleep como unica sincronizacao nos testes do watcher; mtime em `delete_test`

## Progresso

- 00:42 — inicio, sobre `d12f84f` (a Task 162). Inventario inicial: `grep -n "time.Sleep" internal/watcher/*_test.go internal/service/delete_test.go` deu 33 ocorrencias.
- 00:44 — lidos `overflow_test.go` (o mecanismo que o Step 2 manda copiar: `w.handleFSError` direto, sem tocar `fsWatcher.Errors`), `watcher.go` (`New` registra os watches, `Run` so consome), `counters_test.go`, `debounce_test.go`, `burst_test.go`, `rename_test.go`, `watcher_test.go`, `atalho_busca_test.go`, `delete_test.go`.
- 00:46 — `internal/watcher/espera_test.go` criado com `EsperarAte` e `EsperarWatcherAtivo`.
- 00:48 — sitios do watcher convertidos; `go vet` limpo; `go test -race -count=1 ./internal/watcher/` verde. Sobraram DOIS `time.Sleep` no pacote, os dois com o motivo escrito na linha de cima.
- 00:51 — `delete_test.go` feito. Medida a taxa de falso-passe do mtime: 23 de 200 reescritas consecutivas com mtime IDENTICO nesta maquina.
- 00:52 — `-count=20 -race` do comando do brief: `ok` nos dois pacotes. Descoberto que a regex do brief nao casa `TestWatcher_Burst` nem `TestWatcher_RenameEndToEnd` (detalhe abaixo); rodado tambem o pacote `watcher` INTEIRO com `-count=20 -race`: `ok` em 63,6 s.
- 00:53 — duas provas de mutacao com exit 0.
- 00:58 — `verify.ps1` verde (exit 0) na primeira tentativa.
- 00:59 — commit `af93306`.

Fix round 1 (R=1), review `review-164.md`:

- 01:13 — leitura de B1-N5.
- 01:14 — N3 (`t.Cleanup(cancel)`) e N4 (`TestCounters_Reconciliations` apagado) aplicados; `gofmt -l` vazio, `go vet` limpo. N1 (`CLAUDE.md`) reescrito com a medicao ao lado.
- 01:16 — B2: programa do mtime re-rodado tres vezes (12, 9 e 2 de 200 - a taxa nao e constante; secao propria abaixo).
- 01:19 — `-count=20 -race` re-rodado nos dois pacotes sobre o codigo desta rodada: `ok`, zero `DATA RACE`.
- 01:21 — `verify.ps1` verde (exit 0).

## Status

**DONE.** SHA `af93306`.

## Inventario: sitio, o que esperava, sinal observavel, acao

| Sitio (antes) | Espera por | Sinal observavel | Acao |
|---|---|---|---|
| `counters_test.go:94` (`setupTestWatcher`, 50 ms) | o laco de `Run` arrancar | `w.Stats().Active` — `Run` o marca na primeira linha | `EsperarWatcherAtivo(t, w)` |
| `counters_test.go:112` (laco 50x20 ms) | o watch ver o arquivo criado | `Stats().EventsReceived > 0` | `EsperarAte`, com `diagnostico(w)` na falha |
| `counters_test.go:134` | o filtro descartar `desktop.ini` | `Stats().EventsDropped > 0` | `EsperarAte` |
| `counters_test.go:156` | `Apply` indexar | `Stats().EventsProcessed > 0` | `EsperarAte` |
| `counters_test.go:178,193` | indexar, depois pular o reenvio | `EventsProcessed` e `EventsSkipped` | `EsperarAte` (dois) |
| `counters_test.go:206` (escrita em `fsWatcher.Errors`) + `:208` (100 ms) | o overflow ser tratado | nenhum — o tratamento e **sincrono** | chama `w.handleFSError(...)`, a producao. Sleep e escrita no canal removidos |
| `counters_test.go:221-227` (4 escritas em `fsWatcher.Events`) + `:237` | o filtro classificar os quatro | nenhum — `emite` e **sincrona** | chama `w.emite(ctx, e)`, a producao. Sleep e escritas no canal removidos |
| `counters_test.go:266` (100 ms apos `cancel`) | o laco de `Run` sair | `!w.Stats().Active` — marcado no `defer` de `Run` | `EsperarAte` |
| `counters_test.go:294` | o debouncer coalescer | `Stats().EventsCoalesced > 0` | `EsperarAte`, e agora reprova nomeando a coalescencia se estourar |
| `counters_test.go:369` (`esperaContador`) | contador chegar em `quer` | o proprio contador | reimplementado sobre `EsperarAte`; devolve o ultimo valor, que a mensagem do chamador imprime |
| `counters_test.go:448,478` | 3 notas indexadas / `chegou/sub/d.md` indexada | `idx.NoteCount()` e `idx.Get` | `EsperarAte` |
| `debounce_test.go:43` (100 ms, "Wait for tick") | o flush do debouncer (tick de 50 ms) | o **lote** escrito em `out` | `select` sobre `out` ate juntar 3 caminhos, com `time.After(vaulttest.Prazo)` como teto |
| `debounce_test.go:111` (1 ms) | nada — e **estimulo** | — | mantido, com o motivo na linha de cima: espaca as escritas para cobrirem 50 ms contra um tick de 10 ms; o relogio e a variavel independente do teste |
| `burst_test.go:39` (100 ms) | o laco de `Run` arrancar | `Active` | `EsperarWatcherAtivo` |
| `burst_test.go:61` (laco, 60 s) | as 500 notas convergirem | `idx.NoteCount() == 500` | `EsperarAte(60*time.Second, ...)`; o orcamento de 60 s e o comentario que o explica ficaram |
| `rename_test.go:430` (**ausente**) | o laco de `Run` arrancar | `Active` | acrescentado `watcher.EsperarWatcherAtivo` |
| `rename_test.go:448` (laco, 3 s) | a correlacao de rename | `destino.md` no indice **e** `origem.md` fora | `watcher.EsperarAte`; as duas metades juntas, porque so a primeira passaria com um rename tratado como criacao |
| `watcher_test.go:44` (100 ms) | arranque | `Active` | `EsperarWatcherAtivo` |
| `watcher_test.go:60` (laco) | a nota chegar ao indice | `idx.Get(canon)` | `EsperarAte` + `diagnostico(w)` |
| `watcher_test.go:111,142` (50 ms) | arranque, antes de `cancel`/`Close` | `Active` | `EsperarWatcherAtivo` |
| `watcher_test.go:173` (50 ms) | arranque | `Active` | `EsperarWatcherAtivo` |
| `watcher_test.go:180` (100 ms) | o **watch do diretorio novo** ser registrado | `w.fsWatcher.WatchList()` (leitura da API do fsnotify, nao escrita em canal) | `EsperarAte` sobre `estaVigiado(w, subDir)` |
| `watcher_test.go:195` (laco) | a subnota ser indexada | `idx.Get(canon)` | `EsperarAte` |
| `watcher_test.go:242` (100 ms) | arranque | `Active` | `EsperarWatcherAtivo` |
| `watcher_test.go:273` (`esperaTermo`) | posting entrar/sair da busca | `inv.Postings(termo)` | reimplementado sobre `EsperarAte` |
| `atalho_busca_test.go:83` (laco, 5 s) | o indice de busca se recompor | `inv.HasDoc("nota.md")` | `EsperarAte` |
| `atalho_busca_test.go:132` (laco, 5 s) | o atalho pular o evento | `skipped.Load() > 0` | `EsperarAte` |
| `overflow_test.go:117` (laco, 3 s) | a reconciliacao corrigir o tamanho | `idx.Get("nota.md").Size` | `EsperarAte` |
| `delete_test.go:125` (10 ms) | **nada** — ver abaixo | — | removido, com o porque escrito no lugar |
| `delete_test.go:213` (mtime antes/depois) | — (comparacao inerte) | — | `os.Chtimes` para 2020 antes da chamada; afirma que o mtime antigo sobreviveu |

> **Nota do fix round 1:** a linha de `counters_test.go:206` descreve o que
> `af93306` fez com `TestCounters_Reconciliations`. Esse teste foi APAGADO na
> rodada seguinte (N4) por ser subconjunto estrito de
> `TestRun_OverflowSchedulesExactlyOne`; a tabela fica como registro do que foi
> convertido, e a assercao vive hoje em `overflow_test.go`.

### Os dois `time.Sleep` que sobraram, e por que

```
$ grep -n "time.Sleep" internal/watcher/*_test.go
internal/watcher/debounce_test.go:132:		time.Sleep(1 * time.Millisecond)
internal/watcher/espera_test.go:47:		time.Sleep(10 * time.Millisecond)
internal/watcher/espera_test.go:60:// Antes disto, seis testes escreviam `time.Sleep(50 * time.Millisecond)` ou
internal/watcher/espera_test.go:61:// `time.Sleep(100 * time.Millisecond)` com o comentario "Wait for watcher to
internal/watcher/overflow_test.go:110:	// Espera pelo estado do indice, e nao por um relogio: time.Sleep fixo como
internal/watcher/watcher_test.go:274:// esperaTermo espera pelo estado do indice de busca. time.Sleep fixo como
```

Quatro das seis linhas sao comentario. Restam duas chamadas, ambas com o motivo
na linha de cima:

- `espera_test.go:47` — intervalo de **polling** dentro de `EsperarAte`. A
  condicao foi consultada na linha acima e sera de novo na volta seguinte; nada
  depende de 10 ms serem suficientes para coisa alguma.
- `debounce_test.go:132` — **estimulo**, nao sincronizacao: espaca as escritas de
  `TestDebounce_NoStarvation` para que cubram 50 ms contra um tick de 10 ms. Aqui
  o relogio e a variavel independente do teste, e nao ha sinal a esperar.

Em `internal/service/delete_test.go` nao sobrou nenhum.

### Desvios do brief, declarados

**1. `EsperarAte` devolve `bool` em vez de chamar `t.Fatalf`.** O corpo literal
do brief centraliza a espera **e** a mensagem: todo sitio passaria a reprovar com
"condicao nao valeu em 5s". Este pacote gastou commits construindo o contrario —
`diagnostico(w)` despeja os oito contadores e separa "o watch nunca viu o evento"
de "o filtro comeu o evento" de "Apply nao indexou"; `bufferDeLog` existe porque
`io.Discard` ja tornou um estouro de prazo indiagnosticavel aqui em 2026-08-26.
Centralizar a ESPERA sem centralizar a MENSAGEM preserva isso. `EsperarWatcherAtivo`
si reprova sozinha, porque ali a mensagem e sempre a mesma — e leva o
`diagnostico(w)` junto.

**2. `EsperarAte` e `EsperarWatcherAtivo` sao exportadas.** Metade dos testes
deste diretorio esta em `package watcher_test` (`apply_test.go`, `rename_test.go`,
`rename_prefiltro_test.go`), e essa metade so alcanca identificadores exportados.
Nada disso existe no binario de producao: `_test.go` so entra no binario de teste.
A alternativa era uma segunda copia da funcao para o pacote externo, que e a
forma que este projeto proibe por escrito.

**3. Escopo maior que a lista de `Files` do brief.** O brief lista
`counters_test.go`, `debounce_test.go`, `burst_test.go`, `rename_test.go` e
`delete_test.go`, mas a verificacao que ele exige e sobre
`internal/watcher/*_test.go` inteiro. Foram incluidos `watcher_test.go`
(seis sleeps fixos, o mesmo defeito), `atalho_busca_test.go` e `overflow_test.go`
(lacos de polling, convertidos para a mesma funcao). Sem isso a verificacao do
proprio brief nao fecharia.

**4. `TestDeleteNote_TrashNameCollision` ganhou assercao de conteudo.** Ao
explicar por que o sleep de 10 ms nao esperava por nada, ficou visivel que o
teste afirmava `res1.TrashPath != res2.TrashPath` e a existencia dos dois
arquivos, mas nao que o segundo nao tinha sobrescrito o primeiro. Agora le os
dois e confere "Versao 1" e "Versao 2".

## O `-count=20 -race`

O comando exato do brief:

```
$ go test -race -count=20 ./internal/watcher/ ./internal/service/ -run 'TestCounters|TestDebounce|TestBurst|TestRename|TestDelete' 2>&1 | tail -5
ok  	github.com/jonyd/gobsidian/internal/watcher	8.089s
ok  	github.com/jonyd/gobsidian/internal/service	4.021s
```

**Esse comando cobre menos do que parece, e isso importa.** `-run` do Go casa sem
ancora: `TestBurst` nao casa `TestWatcher_Burst`, e `TestRename` nao casa
`TestWatcher_RenameEndToEnd` nem `TestCorrelateRenames`. Enumerado:

```
$ go test -count=1 -v ./internal/watcher/ ./internal/service/ \
    -run 'TestCounters|TestDebounce|TestBurst|TestRename|TestDelete' 2>&1 | grep -c "^=== RUN"
19
```

Os 19 sao os nove `TestCounters_*`, os dois `TestDebounce_*` e oito `TestDelete*`.
Ficaram de fora justamente os dois testes que o brief nomeia por arquivo
(`burst_test.go`, `rename_test.go`), mais `TestPastaQueChegaComArquivosDentro`,
`TestWatcher`, `TestWatcherUpdatesSearchIndex`, `TestAtalhoDoApply*` e
`TestApply_ReconcileSignal`. Entao o pacote **inteiro** foi rodado 20 vezes:

```
$ go test -race -count=20 ./internal/watcher/
ok  	github.com/jonyd/gobsidian/internal/watcher	63.626s

real	1m7.174s
```

`ok` nas 20, sem `DATA RACE`. Saidas cruas em `<scratchpad>/count20.txt` e
`<scratchpad>/count20-watcher.txt`.

## Provas de mutacao

O brief diz que esta tarefa nao tem prova de mutacao, "ela troca sincronizacao de
teste, nao regra de produto". Duas foram rodadas mesmo assim, porque as esperas
novas **sao** assercoes — os sleeps que elas substituem nao eram, e a diferenca
precisa de evidencia.

### A espera de arranque reprova quando o arranque nao acontece

```
pwsh -File scripts/mutate.ps1 -Path internal/watcher/watcher.go `
  -Anchor "`tw.active.Store(true)`n" -Replacement '' `
  -Test TestCounters_EventsReceived -Package ./internal/watcher/
```

```
--- FAIL: TestCounters_EventsReceived (5.01s)
    counters_test.go:94: o laco de Run nao ficou ativo em 5s; nada do que vem depois deste ponto seria observado
          contadores: Active=false Received=0 Dropped=0(map[chmod:0 excluded:0 outside_vault:0 unknown_op:0]) Coalesced=0 Processed=0 Skipped=0 Reconciliations=0
    counters_test.go:74: --- log do watcher ---
        --- fim do log ---
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	5.645s
FAIL
----------------------------------------------------------------------
[OK] internal/watcher/watcher.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT_A=0
```

Com o `time.Sleep(50 * time.Millisecond)` que estava ali, essa mesma mutacao
passava despercebida em `TestCounters_EventsReceived` — so
`TestCounters_ActiveState` a pegava. Agora todo teste que monta o watcher a pega,
e nomeando a causa.

### O contador de reconciliacao continua coberto apos a reescrita

```
pwsh -File scripts/mutate.ps1 -Path internal/watcher/watcher.go `
  -Anchor 'w.reconciliations.Add(1)' -Replacement '_ = w' `
  -Test TestCounters_Reconciliations -Package ./internal/watcher/
```

```
        time=2026-09-06T00:53:53.417-03:00 level=WARN msg="Overflow de fsnotify detectado"
        time=2026-09-06T00:53:53.435-03:00 level=WARN msg="reconciliação agendada"
        time=2026-09-06T00:53:53.435-03:00 level=INFO msg="Iniciando reconciliação completa do cofre devido a overflow"
        time=2026-09-06T00:53:53.436-03:00 level=WARN msg="Reconciliação concluída" updated=0 removed=0 skipped=0
        --- fim do log ---
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	0.674s
FAIL
----------------------------------------------------------------------
[OK] internal/watcher/watcher.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT_B=0
```

(Este teste foi apagado no fix round 1; a mutacao equivalente contra o teste que ficou esta na secao "Prova de mutacao apos N4".) Trocar a escrita em `fsWatcher.Errors` por `handleFSError` nao afrouxou nada — e
o teste passou de 100 ms para 0,67 s de pacote inteiro, porque a espera sumiu.

## O mtime de `delete_test.go`, medido

A assercao antiga tirava um `os.Stat` antes, chamava `DeleteNote(DryRun)` e
comparava com um `os.Stat` depois. Nada entre os dois move o relogio, entao uma
reescrita dentro do mesmo tique de mtime passaria. Isso e frequente o bastante
para importar. Medido nesta maquina (Windows, NTFS, `%TEMP%`), 200 pares de
reescritas consecutivas do mesmo arquivo:

```
dir=C:\Users\jonyd\AppData\Local\Temp\mres1221387508
reescritas consecutivas com mtime IDENTICO: 23 de 200
```

**11,5% dos pares tem mtime identico.** Nao e "podia acontecer": e a taxa medida.
O conserto e o que o brief pede — `os.Chtimes` para 2020-01-02 antes da chamada, e
a assercao de que o mtime antigo sobreviveu. Qualquer reescrita carimba a hora
atual, e a resolucao do sistema de arquivos deixa de fazer parte da conta.

Nao ha mutacao de produto para esta: nao existe caminho em que o `DryRun`
reescreva o arquivo em vez de mover — se ele movesse, o `os.Stat` anterior a
assercao ja reprovaria com "alvo.md sumiu apos dry_run". O que a mudanca compra e
uma assercao que **pode** falhar quando o defeito for "tocou sem mover", contra
uma que dependia do relogio para poder falhar. Isso esta dito assim de proposito:
nao ha aqui um falso-passe historico a exibir, ha uma taxa de falso-passe medida.

## `verify.ps1` — completo, exit 0

```
Carregado em 402ms
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. contagem de testes pulados
[!] 6 testes pulados
     --- SKIP: TestAjudanteSeguraTrava (0.00s)
     --- SKIP: TestListenRestringePermissaoUnix (0.00s)
     --- SKIP: TestSignalCancelsContext (0.10s)
     --- SKIP: TestPerfilDeHeapServindo (0.00s)
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.00s)
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
[...] 4. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 5. go vet (windows)
[OK] go vet (windows)
[...] 6. go vet (linux)
[OK] go vet (linux)
[...] 7. go vet (darwin)
[OK] go vet (darwin)
[...] 8. gofmt
[OK] gofmt
[...] 9. golangci-lint
[OK] golangci-lint
[...] 10. golangci-lint (linux)
[OK] golangci-lint (linux)
[...] 11. check_net (RNF-30)
[OK] check_net (RNF-30)
[...] 12. check_tool_params
[OK] check_tool_params
[...] 13. check_doc_refs
[OK] check_doc_refs
[...] 14. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
EXIT=0
```

Os 6 pulados sao os mesmos de antes da tarefa — nenhum `t.Skip` novo foi
introduzido, e nenhum existente foi tocado.

## `git diff --stat` do commit `af93306`

```
 internal/service/delete_test.go       |  51 ++++++++--
 internal/watcher/atalho_busca_test.go |  27 ++----
 internal/watcher/burst_test.go        |  11 +--
 internal/watcher/counters_test.go     | 175 +++++++++++++---------------------
 internal/watcher/debounce_test.go     |  29 +++++-
 internal/watcher/espera_test.go       |  70 ++++++++++++++
 internal/watcher/overflow_test.go     |  19 ++--
 internal/watcher/rename_test.go       |  27 ++++--
 internal/watcher/watcher_test.go      |  89 +++++++++--------
 9 files changed, 291 insertions(+), 207 deletions(-)
```

So `_test.go`. Nenhum arquivo de produto no commit — as duas mutacoes foram
aplicadas e restauradas byte a byte por `mutate.ps1`, com SHA-256 conferido nas
duas. Nenhuma aresta nova de import no produto: `vaulttest` entrou so em arquivos
`_test.go`, que e o padrao ja usado em `index`, `search`, `service`, `vault`,
`daemon`, `ipc` e `cmd/gobsidian`.


---

## Fix round 1 - os cinco itens da revisao

### B1 - o relatorio ja tinha os cinco itens quando a revisao terminou

O revisor leu o arquivo no estado de 00:51, ainda com o cabecalho "EM ANDAMENTO",
e o achado esta correto para o que ele viu. A versao que esta neste arquivo desde
00:59 tem os cinco itens do contrato: **Status** (`DONE`), **SHA** (`af93306`),
a **tabela de sinais** por sitio, a **saida do `-count=20 -race`** e a **ultima
linha do `verify.ps1`**. Nao ha o que consertar no conteudo; o defeito real foi
de sequencia - o relatorio ficou incompleto no disco enquanto o commit ja
existia, e quem leu no meio viu um entregavel que se declarava incompleto sobre
um trabalho que estava pronto. As secoes abaixo acrescentam o que a revisao pediu
de novo, e as medicoes foram TODAS refeitas sobre o codigo desta rodada, porque
N3 e N4 mudaram `counters_test.go` depois de `af93306`.

### B2 - o numero do mtime, com o programa que o produziu, e a variacao que ele tem

O relatorio publicava "23 de 200" sem a conta ao lado. O programa e este, um
modulo descartavel em `%TEMP%` (`module mres`, `go 1.26.5`), rodado com `go run .`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	dir, _ := os.MkdirTemp("", "mres")
	defer os.RemoveAll(dir)
	p := filepath.Join(dir, "a.md")
	iguais := 0
	const n = 200
	for i := 0; i < n; i++ {
		_ = os.WriteFile(p, []byte("Conteudo"), 0644)
		fi1, _ := os.Stat(p)
		_ = os.WriteFile(p, []byte("Conteudo"), 0644)
		fi2, _ := os.Stat(p)
		if fi1.ModTime().Equal(fi2.ModTime()) {
			iguais++
		}
	}
	fmt.Printf("dir=%s\nreescritas consecutivas com mtime IDENTICO: %d de %d\n", dir, iguais, n)
}
```

Re-rodado agora, tres vezes seguidas:

```
$ go run .
dir=C:\Users\jonyd\AppData\Local\Temp\mres1408865695
reescritas consecutivas com mtime IDENTICO: 12 de 200
$ go run .
dir=C:\Users\jonyd\AppData\Local\Temp\mres2250309662
reescritas consecutivas com mtime IDENTICO: 9 de 200
$ go run .
dir=C:\Users\jonyd\AppData\Local\Temp\mres3623103528
reescritas consecutivas com mtime IDENTICO: 2 de 200
```

**Corrijo a afirmacao anterior.** O relatorio dizia "11,5% dos pares tem mtime
identico. Nao e 'podia acontecer': e a taxa medida". Uma taxa e justamente o que
aquele numero NAO e: cinco execucoes do mesmo programa nesta mesma maquina deram
23, 25 (a do revisor), 12, 9 e 2 de 200 - de 12,5% a 1%. O que varia ai e carga
da maquina, nao uma constante do sistema de arquivos. O enunciado honesto e o
mais fraco, e e suficiente: **a colisao de mtime entre duas reescritas
consecutivas ocorre nesta maquina em TODA execucao de 200 pares que fizemos,
entre 1% e 12,5% das vezes.** O argumento nao precisa de mais que isso - uma
assercao que depende de o relogio ter avancado pode nao falhar quando devia -, e
o conserto (`os.Chtimes` para 2020) tira o relogio da conta inteiramente, entao
a taxa exata nao muda decisao nenhuma.

O mesmo "23 de 200" esta na mensagem de `af93306`, que nao pode ser reescrita.
Este paragrafo e a correcao de registro.

### N1 - `CLAUDE.md` passou a listar `watcher`, com a medicao ao lado

```
$ git grep -l "internal/vaulttest" -- '*_test.go' | cut -d/ -f1-2 | sort -u
cmd/gobsidian
internal/daemon
internal/index
internal/ipc
internal/search
internal/service
internal/vault
internal/vaulttest
internal/watcher
```

(`internal/vaulttest` aparece porque tem `_test.go` proprio; a enumeracao do
`CLAUDE.md` e de quem o importa de fora.) E a metade que decide o grafo:

```
$ git grep -l "internal/vaulttest" -- '*.go' ':!*_test.go'
$
```

Vazio: nenhuma aresta de producao. O paragrafo do `CLAUDE.md` agora traz os dois
comandos e `medido em 2026-09-06`, na mesma forma que o resto do bloco ja usava.
`python -c "open('CLAUDE.md',encoding='utf-8').read()"` -> `[OK] UTF-8 valido`.

### N2 - a contagem "thirteen" da mensagem de `af93306` esta errada; sao 12

A mensagem do commit diz "Thirteen fixed sleeps stood in for synchronization".
A contagem certa, sobre o diff: **31** linhas `time.Sleep(` removidas, das quais
**19** eram o intervalo de polling de lacos de espera escritos a mao e **12**
eram o sleep isolado servindo de sincronizacao. O 13 nao conta nada - nem os
sleeps isolados, nem os `EsperarWatcherAtivo` (que sao 6, mais um setimo novo em
`rename_test.go`). **A historia nao esta sendo reescrita:** `af93306` fica como
esta, e esta linha e o registro de que aquele numero e uma contagem errada, e nao
um recorte diferente.

### N3 - `t.Cleanup(cancel)` antes da espera de arranque

`internal/watcher/counters_test.go`, em `setupTestWatcher`: o `t.Cleanup(cancel)`
foi registrado logo depois do `go w.Run(ctx)` e ANTES do `EsperarWatcherAtivo`.
Se a espera reprovar, ela reprova antes do `return`, o chamador nunca recebe o
`cancel` para pôr em `defer`, e a goroutine de `Run` mais o handle do fsnotify
ficavam ate o fim do binario de teste - vezes 20 sob `-count=20`. O `cancel`
continua sendo devolvido ao chamador; chamar um `CancelFunc` duas vezes e no-op.
O motivo esta no comentario, na linha de cima.

### N4 - `TestCounters_Reconciliations` apagado

**Quem carrega a assercao agora:** `TestRun_OverflowSchedulesExactlyOne`
(`internal/watcher/overflow_test.go`), que chama
`w.handleFSError(fsnotify.ErrEventOverflow)` cinco vezes, confere o valor EXATO
de `Reconciliations` e ainda confere o controle - um erro que nao e overflow nao
mexe no contador. O de `counters_test.go` conferia so `!= 0` sobre uma unica
chamada do mesmo metodo: subconjunto estrito, custo de manutencao sem sinal
proprio.

A prova de mutacao do contador, que este relatorio publicou contra o teste
apagado, foi re-rodada contra o teste que ficou (saida na secao seguinte).

### N5 - `TestDebounce_Coalescence` mudou de FORMA, e nao so de espera

Registro que faltava. **Antes:** dorme 100 ms, cancela o debouncer, drena `out`,
exige 3 caminhos. **Depois:** bloqueia em `out` ate juntar 3 caminhos (ou
`vaulttest.Prazo` = 5 s), so entao cancela, drena o resto e exige 3.

**Ficou mais forte, e essa e a razao da mudanca:** o sleep de 100 ms nao podia
reprovar util - se o lote nunca chegasse, `out` estaria vazio e a mensagem seria
"queria 3, tem 0", sem separar "o debouncer nao emitiu" de "o teste olhou cedo
demais". A espera nova reprova com `t.Fatalf` nomeando o numero parcial e o
prazo, e a drenagem posterior continua pegando um lote extra se ele existir.

O buraco teorico, declarado: o teste para o debouncer assim que o terceiro
caminho aparece, entao um lote duplicado emitido DEPOIS disso escaparia. Nao pode
ocorrer aqui - os 12 eventos entram em `in` (cap 100) antes do primeiro tick de
50 ms, entao ha um lote so -, mas se o gerador de estimulo mudar, essa premissa
muda com ele.

---

## Prova de mutacao apos N4 - o contador de reconciliacao segue coberto

O comando, agora contra o teste que sobreviveu:

```
pwsh -File scripts/mutate.ps1 -Path internal/watcher/watcher.go `
  -Anchor 'w.reconciliations.Add(1)' -Replacement '_ = w' `
  -Test TestRun_OverflowSchedulesExactlyOne -Package ./internal/watcher/
```

```
[...] Mutando internal/watcher/watcher.go
      - w.reconciliations.Add(1)
      + _ = w

[...] go test -race -run TestRun_OverflowSchedulesExactlyOne ./internal/watcher/
----------------------------------------------------------------------
--- FAIL: TestRun_OverflowSchedulesExactlyOne (0.03s)
    overflow_test.go:152: reconciliations counter quer 5, obteve 0
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	0.661s
FAIL
----------------------------------------------------------------------
[OK] internal/watcher/watcher.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada - a regra esta verificada.
EXIT=0
```

Apagar `TestCounters_Reconciliations` nao descobriu o contador: o vizinho reprova
nomeando o valor exato (`quer 5, obteve 0`), que e mais do que o teste apagado
dizia (`!= 0`).

## Medicoes desta rodada (sobre o codigo com N3 e N4 aplicados)

`-count=20 -race`, pacote `watcher` inteiro:

```
$ go test -race -count=20 ./internal/watcher/
ok  	github.com/jonyd/gobsidian/internal/watcher	65.601s
EXIT_W=0
$ grep -c 'DATA RACE' <saida>
0
```

`-count=20 -race`, `TestDelete` em `service`:

```
$ go test -race -count=20 ./internal/service/ -run 'TestDelete'
ok  	github.com/jonyd/gobsidian/internal/service	3.434s
EXIT_S=0
$ grep -c 'DATA RACE' <saida>
0
```

O comando literal do brief (que, como ja registrado acima, cobre 19 testes e nao
casa `TestWatcher_Burst` nem `TestWatcher_RenameEndToEnd`):

```
$ go test -race -count=20 ./internal/watcher/ ./internal/service/ \
    -run 'TestCounters|TestDebounce|TestBurst|TestRename|TestDelete'
ok  	github.com/jonyd/gobsidian/internal/watcher	7.660s
ok  	github.com/jonyd/gobsidian/internal/service	4.002s
```

### `verify.ps1` - fix round 1

Ultima linha, exit 0:

```
[OK] Bateria completa. Pode commitar.
EXIT=0
```

Os 14 passos sairam `[OK]`; os seis pulados sao os mesmos seis de sempre, todos
preexistentes.
