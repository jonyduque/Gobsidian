# Task 177 — `boot.VigiarHost` e os passos de shutdown compartilhados

## Progresso

- 11:17 — li o brief, `cmd/gobsidian/serve.go`, `ponte.go`, `daemon.go`,
  `internal/boot/montar.go`, `internal/lifecycle/lifecycle.go`,
  `cmd/gobsidian/serve_test.go`, `internal/boot/indice_test.go`,
  `internal/boot/doc.go`. Confirmadas as correcoes de linha do orquestrador.
- 11:18 — Step 1: `internal/boot/vigia_test.go` escrito (pacote `boot_test`,
  usa o `logSilencioso()` de `indice_test.go`). RED por compilacao, colado
  abaixo.
- 11:19 — Step 2: `internal/boot/espelho.go` (mirrorReader/mirrorDst verbatim
  de `serve.go`), `internal/boot/vigia.go` (Vigia/VigiarHost/
  PassoFecharEspelho), `(*Componentes).PassoWatcher` em `montar.go`, `doc.go`
  atualizado. Os tres `TestMirrorReader*` movidos para
  `internal/boot/espelho_test.go` (pacote `boot`, interno — mirrorReader e
  nao exportado), com `eofReader`/`chunkReader`/`spyDst`.
  `TestShutdownExitCode` ficou em `cmd/gobsidian/serve_test.go`.
- 11:20 — Step 3: os tres pontos de saida usam o andaime compartilhado.
  `go test -race ./cmd/... ./internal/boot/` verde.
- 11:21 — Step 4: as duas provas de mutacao, exit 0.
- 11:22 — Step 6: `go list` mostra as sete arestas. `CLAUDE.md`,
  `docs/ARCHITECTURE.md`, `docs/ESTRUTURA.md` e `docs/ARMADILHAS.md`
  atualizados; UTF-8 validado em cada um.
- 11:23 — Step 7 (1a tentativa): `verify.ps1` REPROVOU nas etapas 9 e 10
  (`golangci-lint`, `contextcheck`, 3 achados). Causa e correcao abaixo.
- 11:27 — correcao aplicada: `VigiarHost` devolve `(context.Context, *Vigia)`;
  o campo `Ctx` sai do struct. Linter limpo (0 issues).
- 11:29 — provas de mutacao RE-RODADAS sobre o codigo final (exit 0 nas duas),
  `go list` re-medido, `verify.ps1` VERDE (14 etapas).
- 11:33 — commit `b52b441` por caminhos explicitos, via `git commit -F`.
- 11:35 — `audit_reports.ps1 177`.

## Status

DONE_WITH_CONCERNS

O trabalho esta completo e o gate esta verde, mas **desviei da assinatura
literal do brief** para o gate passar. Detalhe em `## Concerns`.

`scripts/test_orphans.ps1` (Step 5 do brief) **nao foi rodado**: por instrucao
explicita do orquestrador, esse gate roda destacado depois do commit.

## Commits

- `b52b441` — `refactor(boot): host watch and shutdown steps shared by serve,
  bridge and daemon` — 14 arquivos, +451 −318, 4 arquivos novos
  (`internal/boot/espelho.go`, `espelho_test.go`, `vigia.go`, `vigia_test.go`).

Mensagem em
`.superpowers/sdd/2026-09-02-topografia-e-limpeza/commit-177.txt`.

## Step 1 — RED (falha de compilacao)

```
$ go test ./internal/boot/ -run TestVigiarHostEOFDoStdinCancelaCtx
# github.com/jonyd/gobsidian/internal/boot_test [github.com/jonyd/gobsidian/internal/boot.test]
internal\boot\vigia_test.go:15:12: undefined: boot.VigiarHost
FAIL	github.com/jonyd/gobsidian/internal/boot [build failed]
FAIL
```

## Step 2/3 — GREEN

```
$ go test -race ./internal/boot/ -v -run 'TestMirrorReader|TestVigiarHost'
=== RUN   TestMirrorReaderCopiesToMirror
--- PASS: TestMirrorReaderCopiesToMirror (0.00s)
=== RUN   TestMirrorReaderPropagatesEOF
--- PASS: TestMirrorReaderPropagatesEOF (0.00s)
=== RUN   TestMirrorReaderBrokenMirrorDoesNotPoisonRead
--- PASS: TestMirrorReaderBrokenMirrorDoesNotPoisonRead (0.00s)
=== RUN   TestVigiarHostEOFDoStdinCancelaCtx
--- PASS: TestVigiarHostEOFDoStdinCancelaCtx (0.02s)
PASS
ok  	github.com/jonyd/gobsidian/internal/boot	1.712s
```

```
$ go test -race ./cmd/... ./internal/boot/
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	4.698s
ok  	github.com/jonyd/gobsidian/internal/boot	1.808s
```

## Step 4 — Provas de mutacao (rodadas sobre o codigo FINAL, 11:29)

### Mutacao 1: `internal/boot/espelho.go` (a ancora do brief)

```
[...] Mutando internal/boot/espelho.go
      - _ = m.dst.CloseWithError(err)
      + _ = err

[...] go test -race -run TestMirrorReaderPropagatesEOF ./internal/boot/
----------------------------------------------------------------------
--- FAIL: TestMirrorReaderPropagatesEOF (5.00s)
    espelho_test.go:178: lado de leitura do pipe nao recebeu EOF em 5s: mirrorReader nao fechou dst
FAIL
FAIL	github.com/jonyd/gobsidian/internal/boot	5.750s
FAIL
----------------------------------------------------------------------
[OK] internal/boot/espelho.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

### Mutacao 2: `internal/boot/vigia.go` (o teste NOVO)

O brief nao deu ancora para este. Escolhi a que remove o elo stdin→cancel sem
quebrar a compilacao: sem o `mirrorReader` entre o stdin do host e a ponta de
escrita do pipe, o EOF do host nunca chega ao monitor do lifecycle.

```
[...] Mutando internal/boot/vigia.go
      - Stdin: &mirrorReader{src: stdin, dst: pw},
      + Stdin: stdin,

[...] go test -race -run TestVigiarHostEOFDoStdinCancelaCtx ./internal/boot/
----------------------------------------------------------------------
--- FAIL: TestVigiarHostEOFDoStdinCancelaCtx (5.02s)
    vigia_test.go:25: EOF em stdin nao cancelou o ctx
FAIL
FAIL	github.com/jonyd/gobsidian/internal/boot	5.709s
FAIL
----------------------------------------------------------------------
[OK] internal/boot/vigia.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

## Step 5 — Orfaos

NAO RODADO nesta sessao, por instrucao explicita do orquestrador: ele roda
`scripts/test_orphans.ps1` destacado depois do commit. Nao afirmo nada sobre o
resultado dele.

## Step 6 — Grafo

```
$ go list -f '{{.Imports}}' ./internal/boot/ | tr ' ' '\n' | grep gobsidian
github.com/jonyd/gobsidian/internal/config
github.com/jonyd/gobsidian/internal/index
github.com/jonyd/gobsidian/internal/lifecycle
github.com/jonyd/gobsidian/internal/search
github.com/jonyd/gobsidian/internal/service
github.com/jonyd/gobsidian/internal/vault
github.com/jonyd/gobsidian/internal/watcher
```

Exatamente as sete esperadas: `config index lifecycle search service vault
watcher`. Nem `mcpsrv`, nem `ipc`, nem `daemon`, nem `doctor`. Medido tambem
com `GOOS=windows` — mesma saida, sete linhas.

Documentacao atualizada:

- `CLAUDE.md`: linha do grafo vira
  `boot     → config, index, lifecycle, search, service, vault, watcher`;
  paragrafo novo justificando a aresta (andaime escrito duas vezes, a ordem
  entre as pecas e o que faz o EOF chegar); a arvore de `internal/boot/`
  descreve a vigilia; o cabecalho do bloco registra que a linha do `boot`
  mudou no mesmo dia (Task 177).
- `docs/ARCHITECTURE.md`: paragrafo em §7.3 sobre `boot.VigiarHost` e os
  passos compartilhados vs. os que ficam em quem serve; §2.14 corrigida (dizia
  "boot nao importa mcpsrv nem lifecycle" — a segunda metade deixou de ser
  verdade).
- `docs/ESTRUTURA.md`: `espelho.go` e `vigia.go` na arvore; a linha do pacote
  corrigida pelo mesmo motivo.
- `docs/ARMADILHAS.md`: a entrada do `io.TeeReader` agora aponta onde o
  `mirrorReader` mora.

UTF-8 validado nos quatro com
`python -c "open('<arquivo>',encoding='utf-8').read()"`.

## Step 7 — `verify.ps1`

Primeira tentativa (11:23) REPROVOU:

```
[...] 9. golangci-lint
WARNING: [!] golangci-lint
     cmd\gobsidian\ponte.go:199:20: Non-inherited new context, use function like `context.WithXXX` instead (contextcheck)
     cmd\gobsidian\serve.go:99:23: Non-inherited new context, use function like `context.WithXXX` instead (contextcheck)
     cmd\gobsidian\serve.go:111:19: Non-inherited new context, use function like `context.WithXXX` instead (contextcheck)
     3 issues:
     * contextcheck: 3
...
WARNING: [!] 2 etapa(s) reprovada(s):
    golangci-lint
    golangci-lint (linux)
```

Causa: o brief especifica `Vigia` com um campo `Ctx` e
`VigiarHost(...) *Vigia`. Com `ctx := vig.Ctx`, o `ctx` passado adiante vem de
um campo de struct, e o `contextcheck` nao consegue provar que ele descende do
`parent` da funcao — o mesmo que a documentacao do pacote `context` diz
("Do not store Contexts inside a struct type").

Correcao: `VigiarHost` devolve `(context.Context, *Vigia)`, exatamente a forma
que `lifecycle.New(parent, opts) (context.Context, *Lifecycle)` ja usa neste
projeto, e o campo `Ctx` sai. Nenhum `//nolint`.

Segunda tentativa (11:29) VERDE, 14 etapas:

```
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
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.01s)
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
```

Os 6 pulados sao os mesmos de antes desta Task — nenhum deles esta em
`internal/boot` nem em `cmd/gobsidian`; nenhum teste novo ou movido pula.

## Concerns

1. **Desvio da assinatura literal do brief (o unico).** O brief pede
   `type Vigia struct { Ctx context.Context; ... }` e
   `func VigiarHost(...) *Vigia`. Entreguei
   `func VigiarHost(...) (context.Context, *Vigia)` sem o campo `Ctx`, porque
   a forma do brief reprova `golangci-lint`/`contextcheck` em tres pontos, e
   `verify.ps1` verde nao e negociavel. A alternativa seria um `//nolint`, que
   silenciaria um aviso legitimo: a diretriz do pacote `context` e a mesma.
   A forma entregue e a que `lifecycle.New` ja usa aqui. O teste do Step 1 foi
   ajustado em duas linhas (`ctx, v := ...` e `case <-ctx.Done():`); o resto
   dele — inclusive as assercoes de `Reason()`, nome e orcamento do passo — e
   literal como no brief.

2. **O daemon nao ganhou `Vigia`, so `PassoWatcher`.** E o que o brief e a
   correcao do orquestrador mandam: nao ha stdin de host para vigiar. Deixei o
   motivo no comentario de `runDaemon`, para que a proxima leitura nao ache
   que foi esquecimento.

3. **`espelho_test.go` ficou no pacote `boot` (interno), nao em `boot_test`.**
   `mirrorReader`, `mirrorDst` e o campo `broken` sao nao exportados, e
   `TestMirrorReaderBrokenMirrorDoesNotPoisonRead` afirma sobre `m.broken`.
   `vigia_test.go` ficou em `boot_test` e usa so a API exportada mais o
   `logSilencioso()` que ja existia la. Foi a preferencia que o orquestrador
   indicou.

4. **Orfaos nao medidos por mim.** Ver `## Step 5`.

## audit_reports

`pwsh -File scripts/audit_reports.ps1 177`, 11:34, exit 0.

**Zero achados neste relatorio.** A secao `=== Relatorios (1) ===` saiu vazia —
o script encontrou `task-177-report.md` e nao apontou nada nele.

Os 14 achados da execucao sao TODOS da secao `=== Ledger ===`, e todos em
`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` — o marco anterior.
Nenhum cita a Task 177 nem o marco `2026-09-02-topografia-e-limpeza`. Sao
pre-existentes a esta Task e nao os toquei:

```
=== Relatorios (1) ===

=== Ledger ===
  ...progress.md:9: [SHA-NAO-CONFERE] Task 4 ...
  ...progress.md:11: [SHA-NAO-CONFERE] Task 6 ...
  ...progress.md:284,287,290: [RELATORIO-AUSENTE] Tasks 94, 95, 96
  ...progress.md:315,318,321,324,327,330,333: [RELATORIO-AUSENTE] Tasks 97-103
  ...progress.md:988: [SHA-FANTASMA] deadbee nao existe no repositorio
  ...progress.md:5158: [SHA-NAO-CONFERE] Task 153 ...

[!] 14 achado(s). Nenhum e automaticamente um defeito — cada um e
    uma frase ou um SHA que precisa de uma pessoa confirmando.
[i] Onde nao houve medicao, 'nao medido' e a resposta certa e nao e sinalizada.
EXIT=0
```

---

## Round 1 — achado N1 da revisao

### Progresso

- 11:46 — li `review-177.md` secao `### N1`. Conferi a string do motivo em
  `internal/lifecycle/stdin.go`: `l.trigger("stdin-eof")` na linha 32, dentro
  do ramo `errors.Is(err, io.EOF)` de `watchStdin`. Nao digitada de memoria.
- 11:47 — `TestPassoFecharEspelhoFechaOEspelho` acrescentado a
  `internal/boot/vigia_test.go`. O corpo e o da revisao; so o comentario de
  cabecalho e meu. Compilou como esta — nenhum ajuste de nome ou import foi
  preciso (`context`, `io`, `testing`, `time`, `boot`, `vaulttest` ja estavam
  importados, e `logSilencioso()` ja mora em `indice_test.go`, mesmo pacote
  `boot_test`).
- 11:48 — prova de mutacao, exit 0. Colada abaixo.
- 11:49 — `go test -race -count=1 ./internal/boot/ ./cmd/gobsidian/` verde.
- 11:50 — `verify.ps1 -SkipCross -SkipNet` verde, 11 etapas.
- 11:51 — commit `ff8f234`, so `internal/boot/vigia_test.go`.

### Status da rodada

DONE

### Commits

- `ff8f234` — `test(boot): the close-pipe step must actually reach the stdin
  watch [sem-doc]` — 1 arquivo, +34, 0 remocoes.

O sufixo ` [sem-doc]` entrou por ser commit so de teste, na forma que os
commits anteriores do repositorio ja usam. Mensagem em
`.superpowers/sdd/2026-09-02-topografia-e-limpeza/commit-177-fix.txt`.

### Prova de mutacao (a ancora que a revisao pediu)

```
$ pwsh -File scripts/mutate.ps1 -Path internal/boot/vigia.go \
    -Anchor 'return v.pw.Close()' -Replacement 'return nil' \
    -Test TestPassoFecharEspelhoFechaOEspelho -Package ./internal/boot/

[...] Mutando internal/boot/vigia.go
      - return v.pw.Close()
      + return nil

[...] go test -race -run TestPassoFecharEspelhoFechaOEspelho ./internal/boot/
----------------------------------------------------------------------
--- FAIL: TestPassoFecharEspelhoFechaOEspelho (5.03s)
    vigia_test.go:66: fechar o espelho nao levou EOF ao monitor de stdin
FAIL
FAIL	github.com/jonyd/gobsidian/internal/boot	5.676s
FAIL
----------------------------------------------------------------------
[OK] internal/boot/vigia.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

A mesma mutacao devolvia exit 1 antes desta rodada — e o numero que a revisao
mediu, e a razao do achado. Agora o passo esta **verificado**, nao so escrito.

### Testes

```
$ go test -race -count=1 -run TestPassoFecharEspelhoFechaOEspelho -v ./internal/boot/
=== RUN   TestPassoFecharEspelhoFechaOEspelho
--- PASS: TestPassoFecharEspelhoFechaOEspelho (0.02s)
PASS
ok  	github.com/jonyd/gobsidian/internal/boot	1.653s
```

```
$ go test -race -count=1 ./internal/boot/ ./cmd/gobsidian/
ok  	github.com/jonyd/gobsidian/internal/boot	2.204s
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	3.919s
```

### `verify.ps1 -SkipCross -SkipNet`

```
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
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.01s)
[...] 4. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 5. go vet (windows)
[OK] go vet (windows)
[i] vet cruzado pulado (-SkipCross)
[...] 6. gofmt
[OK] gofmt
[...] 7. golangci-lint
[OK] golangci-lint
[...] 8. golangci-lint (linux)
[OK] golangci-lint (linux)
[i] check_net pulado (-SkipNet)
[...] 9. check_tool_params
[OK] check_tool_params
[...] 10. check_doc_refs
[OK] check_doc_refs
[...] 11. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

Os 6 pulados sao os mesmos das rodadas anteriores; nenhum teste novo pula.
O gate rodou com `-SkipCross -SkipNet` por ser mudanca so de teste, como o
orquestrador instruiu — `go vet` (linux/darwin) e `check_net` NAO foram
rodados nesta rodada, e nao afirmo nada sobre eles.

### Concerns da rodada

Nenhuma. O achado N1 esta fechado com a ancora que a propria revisao indicou.

`scripts/test_orphans.ps1` continua NAO rodado por mim.
