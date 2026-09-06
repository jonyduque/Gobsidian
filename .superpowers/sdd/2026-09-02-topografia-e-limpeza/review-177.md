# Revisão da Task 177 — `boot.VigiarHost` e os passos de shutdown compartilhados

Revisor: rev-177 (Opus). Commit revisado: `b52b441` (base `5e89964`).
Modo: somente leitura — nenhum arquivo editado, nenhum comando que mexa em
árvore ou índice, `test_orphans.ps1` e `build.ps1` NÃO rodados.

## Progresso

- 11:35 — li o brief (`task-177-brief.md`) e o relatório
  (`task-177-report.md`), inclusive `## Concerns`.
- 11:36 — li o diff completo (`review-5e89964..b52b441.diff`, 1314 linhas) e
  `internal/boot/vigia.go`, `espelho.go`, `montar.go`, `doc.go`,
  `vigia_test.go`, `espelho_test.go`.
- 11:36 — diff dos três testes movidos contra os apagados de
  `cmd/gobsidian/serve_test.go` (`git show 5e89964:` + `diff -u`).
- 11:37 — li `cmd/gobsidian/serve.go`, `ponte.go`, `daemon.go`,
  `internal/lifecycle/lifecycle.go`, `stdin.go`, `shutdown.go`, e os hunks de
  `CLAUDE.md`, `docs/ARCHITECTURE.md`, `ESTRUTURA.md`, `ARMADILHAS.md`.
- 11:38 — `go build ./...`, `go vet`, `go test -race -count=1`,
  `go list -f '{{.Imports}}' ./internal/boot/` — saídas em
  `## Verified claims`.
- 11:38 — semântica de fechamento duplo do `io.Pipe` conferida no fonte do
  toolchain (`$GOROOT/src/io/pipe.go`), não assumida.
- 11:39 — mutação do brief (`espelho.go`) re-rodada por mim: exit 0.
- 11:39 — mutação de sondagem (`PassoFecharEspelho` vira no-op): exit 1 —
  achado N1.
- 11:40 — `golangci-lint run ./internal/boot/... ./cmd/gobsidian/...`:
  0 issues. Linhas dos achados coletadas.

## Spec compliance

**APPROVED**

| Step | Exigência | Situação |
|---|---|---|
| 1 | `vigia_test.go` em `boot_test`, corpo do brief, RED por compilação | OK. O corpo é literal ao brief exceto as duas linhas que a decisão do orquestrador impõe (`ctx, v := ...`, `case <-ctx.Done():`). RED colado no relatório; não reproduzível agora (o código existe), e o relatório não o inventa: a mensagem é a que `go test` emite para `undefined: boot.VigiarHost`. |
| 2 | `espelho.go` com `mirrorReader`/`mirrorDst` **verbatim**; `vigia.go`; `PassoWatcher` | OK. `diff -u` entre o bloco de `5e89964:cmd/gobsidian/serve.go` e `internal/boot/espelho.go` acusa **uma** linha de diferença: `import "io"`. A âncora `_ = m.dst.CloseWithError(err)` está exatamente como pedida. `PassoWatcher` em `montar.go:41-45`, nome `watcher`, 500 ms. |
| 3 | Três pontos de saída usando o andaime; `mirrorReader` apagado de `serve.go`; três testes movidos | OK. `serve.go:92` e `ponte.go:162` usam `ctx, vig := boot.VigiarHost(...)`; `daemon.go:151` continua com `lifecycle.New(parent, Options{Logger: log})` e `daemon.go:177` com `c.PassoWatcher()`. `serve.go` não tem mais `mirrorReader` (nem importa `io`). Os três testes movidos são idênticos aos apagados fora de `package`/imports — nenhuma asserção enfraquecida ou removida (diff colado abaixo). `TestShutdownExitCode` ficou em `serve_test.go`. |
| 4 | Prova de mutação da âncora do brief | OK, **re-rodada por mim**: exit 0, saída colada. |
| 5 | `test_orphans.ps1` | Fora do escopo desta revisão por decisão do orquestrador. O relatório diz "NÃO RODADO" e não afirma nada sobre o resultado — que é a resposta certa. |
| 6 | Grafo (`go list`) e documentação | OK, `go list` **re-medido por mim**: sete arestas, exatamente `config index lifecycle search service vault watcher`; sem `mcpsrv`, `ipc`, `daemon`, `doctor`. `CLAUDE.md`, `ARCHITECTURE.md` §2.14 e §7.3, `ESTRUTURA.md` e `ARMADILHAS.md` atualizados e coerentes com o código. |
| 7 | `verify.ps1` verde e commit | Relatado verde (14 etapas) e colado. Não re-rodei o gate inteiro; re-rodei o que ele reprovava e o que esta Task toca: `go build`, `go vet` (boot + cmd), `go test -race -count=1`, `golangci-lint` nos dois pacotes (0 issues). Commit em Conventional Commits, em inglês. |

Nomes e orçamentos, conferidos um a um contra o código atual:

- `close-pipe` / 500 ms — `internal/boot/vigia.go:59`.
- `watcher` / 500 ms — `internal/boot/montar.go:42`.
- `in-flight` / 3 s — `cmd/gobsidian/serve.go:141` (permanece em quem serve).
- `half-close` / 2 s — `cmd/gobsidian/ponte.go:212`.
- `close-conn` / 500 ms — `cmd/gobsidian/ponte.go:228`.
- limite duro 6 s — `serve.go:140`, `ponte.go:198`, `daemon.go:176`.

Ordem preservada: `serve` = in-flight → close-pipe → watcher; `ponte` =
close-pipe → half-close → close-conn; `daemon` = watcher. Depois do
`Shutdown`, `serve` faz `vig.LC.Wait()` e `c.Esperar()` na mesma ordem de
antes; a ponte faz `vig.LC.Wait()` no lugar de `lc.Wait()`.

Desvio de assinatura: **um só**, o já julgado pelo orquestrador
(`VigiarHost(parent, stdin, log) (context.Context, *Vigia)`, sem campo `Ctx`).
Nenhum outro nome mudou: `Vigia`, `VigiarHost`, `PassoFecharEspelho`,
`PassoWatcher` são os do brief. Os três pontos que chamam `VigiarHost`
(`serve.go:92`, `ponte.go:162`, `vigia_test.go:15`) usam o `ctx` **devolvido**;
`grep` não encontra nenhum `ctx` lido de campo de struct.

## Code quality

**APPROVED** — 0 achados bloqueantes, 1 should-fix, 1 nit.

Ponto a ponto do que a revisão pediu:

1. **`Vigia` é dono do `*io.PipeWriter`, e é o mesmo `pw`.**
   `vigia.go:46-51` constrói `&mirrorReader{src: stdin, dst: pw}` e guarda o
   MESMO `pw` no campo; `PassoFecharEspelho` (`vigia.go:60`) fecha `v.pw`.
   Não há segundo pipe, e o campo é não exportado — ninguém de fora troca um
   sem o outro.

2. **Fechamento duplo / corrida com `CloseWithError`: seguro, e conferido no
   fonte, não assumido.** `pw.Close()` e `m.dst.CloseWithError(err)` caem os
   dois em `(*pipe).closeWrite`, que faz `p.werr.Store(err)` — `onceError`,
   sob mutex, **primeiro erro vence** — e `p.once.Do(func() { close(p.done) })`
   — `sync.Once`, nunca fecha o canal duas vezes. `closeWrite` sempre devolve
   `nil`, então o passo `close-pipe` nunca reporta erro por causa de um
   fechamento anterior. Fonte colado em `## Verified claims`.
   `go test -race` nos dois pacotes está limpo.

3. **`ponte.go` mantém a ordem.** `close-pipe` (`vig.PassoFecharEspelho()`,
   linha 199) → `half-close` (2 s, `conn.CloseWrite`, linha 212) → `close-conn`
   (500 ms, linha 228). O comentário sobre o meio-fechamento ficou intacto.

4. **`daemon.go` continua sem vigiar stdin e sem `ParentPID`.**
   `daemon.go:151` é `lifecycle.New(parent, lifecycle.Options{Logger: log})` —
   `Stdin` e `ParentPID` no zero-valor, o que desliga `watchStdin` e
   `watchParent` (`lifecycle.go:64-70`). O comentário novo (`daemon.go:117-119`)
   diz **por que** o daemon não usa `VigiarHost`, o que é a informação que
   faltaria à próxima leitura.

5. **`PassoWatcher` com `Watcher` nil: não acontece por caminho de produção.**
   `Montar` só devolve `*Componentes` depois de `watcher.New` ter dado certo
   (`montar.go:161-164`); em qualquer erro anterior devolve `nil, err` e os três
   chamadores retornam antes do `Shutdown`. E a premissa "seria engolido pelo
   limite duro" não vale: `lifecycle.Shutdown` documenta e pratica **não ter
   recover** (`shutdown.go:22-27`), então um panic dentro de uma etapa derruba o
   processo com stack no stderr — não vira timeout silencioso. Nada a corrigir.

6. **`Reason()` não foi digitado de memória.** `vigia_test.go:27` compara com
   `"stdin-eof"`, que é a string literal em `internal/lifecycle/stdin.go:32`
   (`l.trigger("stdin-eof")`), e é o EOF do pipe que a produz — `pw.Close()`
   faz o lado de leitura devolver `io.EOF`, que é o ramo `errors.Is(err, io.EOF)`
   de `watchStdin`. Conferido no fonte dos dois lados.

7. **Mutações.** A do brief reprova (exit 0, re-rodada por mim). A segunda que o
   implementador escolheu (`Stdin: stdin` no lugar do `mirrorReader`) é
   pertinente e o teste passa com o espelho presente — `go test -race` verde.
   Mas existe uma terceira que **nada** na suíte pega: ver N1.

8. **Comentários velhos.** No código de produção, nenhum: `grep -rn
   'mirrorReader\|mirrorDst' --include='*.go'` só acha `internal/boot/*` e a
   citação em `ponte.go:161`, que já diz "o mirrorReader de `boot.VigiarHost`".
   Na documentação normativa, tudo atualizado. Sobra a wiki derivada — N2.

9. **`doc.go` e o texto do grafo batem com `go list`.** Medido agora, saída
   colada. `doc.go` diz "não importa mcpsrv" e "não decide quando encerrar" —
   as duas afirmações são verdadeiras contra o import set atual e contra o
   código (`lifecycle.Shutdown` é chamada só em `cmd/gobsidian`, `grep`
   confirma).

10. **A justificativa da aresta sobrevive a quem não viu esta Task.**
    `CLAUDE.md:129-140` nomeia o defeito (o andaime escrito duas vezes), **qual**
    andaime (pipe, `mirrorReader`, `lifecycle.New`, nessa ordem), por que a
    ordem importa (é o que faz o EOF do host chegar ao monitor de stdin), onde
    estava (`serveEmProcesso` e `servePonteRemota`) e o que continua fora
    (`mcpsrv`, e a decisão de **quando** encerrar). É história, não preferência.
    O parágrafo anterior foi reescrito de "não traz aresta nova nenhuma" para
    "nasceu sem aresta nova nenhuma", o que evita deixar no arquivo uma frase
    que passou a ser falsa.

11. **§2.14 e §7.3 estão corretas, e nenhum número novo é inventado.**
    §2.14 trocou "não importa `mcpsrv` nem `lifecycle`" — cuja segunda metade
    deixou de ser verdadeira — por uma frase que separa o que `boot` monta do
    que ele não decide. §7.3 ganhou dois parágrafos: o mecanismo do espelho e a
    tabela de qual passo é compartilhado e qual fica em quem serve. Os únicos
    números que aparecem (500 ms, 3 s, 2 s, 6 s) são orçamentos que já existiam
    no código, não medições — nenhum "conferido" sem evidência, nenhuma latência
    nova afirmada.

## Findings

### N1 — `PassoFecharEspelho` pode virar no-op e a suíte inteira continua verde

- **Onde:** regra em `internal/boot/vigia.go:60`; asserção que deveria
  travá-la em `internal/boot/vigia_test.go:34-36`.
- **Severidade:** should-fix.
- **Defeito:** o teste afirma o **nome** e o **orçamento** do passo e depois só
  confere que `passo.Fn(ctx)` devolve `nil`. Um `Fn` que não faz nada devolve
  `nil` também. Medido, não deduzido: trocando `return v.pw.Close()` por
  `return nil`, `go test -race -run Test ./internal/boot/` — a suíte **inteira**
  do pacote, não só o teste novo — passa (saída em `## Verified claims`). A
  regra "o passo fecha o espelho" está escrita, não verificada, e é exatamente
  a regra que o gate de órfãos depende que exista. Não é bloqueante porque
  nenhum relatório afirma o contrário: o implementador declarou as duas
  mutações que rodou e nenhuma delas alega cobrir este ponto.
- **Correção concreta:** um teste que observe o efeito, não o retorno. Com o
  stdin do host mantido ABERTO (nada de EOF por conta própria), quem cancela o
  ctx passa a ser só o fechamento do espelho:

  ```go
  func TestPassoFecharEspelhoFechaOEspelho(t *testing.T) {
      pr, pw := io.Pipe()          // stdin do host, que NUNCA fecha
      t.Cleanup(func() { _ = pw.Close(); _ = pr.Close() })
      ctx, v := boot.VigiarHost(context.Background(), pr, logSilencioso())
      go func() { _, _ = io.Copy(io.Discard, v.Stdin) }()

      if err := v.PassoFecharEspelho().Fn(context.Background()); err != nil {
          t.Fatalf("fechar o espelho: %v", err)
      }
      select {
      case <-ctx.Done():
      case <-time.After(vaulttest.Prazo):
          t.Fatal("fechar o espelho nao levou EOF ao monitor de stdin")
      }
      if r := v.LC.Reason(); r != "stdin-eof" {
          t.Fatalf("Reason = %q, quero \"stdin-eof\"", r)
      }
  }
  ```

  O mecanismo está conferido nos dois lados: `pw.Close()` faz o lado de leitura
  devolver `io.EOF` (`$GOROOT/src/io/pipe.go`, `closeWrite`), e `watchStdin`
  responde a `io.EOF` com `trigger("stdin-eof")`
  (`internal/lifecycle/stdin.go:29-33`). A prova de mutação para ele é a mesma
  âncora de sondagem: `-Path internal/boot/vigia.go -Anchor 'return v.pw.Close()'
  -Replacement 'return nil'`, que hoje devolve exit 1 e passaria a devolver 0.

### N2 — wiki derivada ainda aponta `cmd/gobsidian/serve.go` como casa do espelho

- **Onde:** `docs/wiki/flows/encerramento.md:6-12` (`source_paths`) e a seção
  "O espelho de stdin", linhas 79-88; a menção em
  `docs/wiki/risks/armadilhas-pagas.md:62-65` não cita caminho e não engana.
- **Severidade:** nit.
- **Defeito:** a página está com `status: active` e lista
  `cmd/gobsidian/serve.go` entre os `source_paths`, mas `mirrorReader` mudou
  para `internal/boot/espelho.go` e o andaime para `internal/boot/vigia.go`.
  Quem seguir os `source_paths` não acha o código descrito. Atenua muito o
  problema o fato de a página carregar `source_commit: cceb980` — ela se
  declara derivada de um commit anterior, que é o mecanismo que o projeto usa
  para staleness —, e o brief não pediu a wiki.
- **Correção concreta:** acrescentar `internal/boot/espelho.go` e
  `internal/boot/vigia.go` aos `source_paths` e uma frase na seção do espelho
  dizendo que ele é montado por `boot.VigiarHost`. Cabe nesta Task ou na
  próxima passagem de ingestão da wiki — não bloqueia o commit.

## Verified claims

Todos os comandos abaixo foram rodados por mim, em primeiro plano, na raiz do
repositório, com `HEAD` em `b52b441` e sem alteração de árvore.

### `HEAD` e build

```
$ git log --oneline -1
b52b441 refactor(boot): host watch and shutdown steps shared by serve, bridge and daemon

$ go build ./...
BUILD_OK

$ go vet ./internal/boot/ ./cmd/gobsidian/
VET_OK
```

### Testes

```
$ go test -race -count=1 ./internal/boot/ ./cmd/gobsidian/
ok  	github.com/jonyd/gobsidian/internal/boot	2.085s
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	3.892s
```

### Lint

```
$ golangci-lint run ./internal/boot/... ./cmd/gobsidian/...
0 issues.
EXIT=0
```

### Grafo (Step 6, re-medido)

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

Sete linhas, iguais às do relatório e à linha nova do `CLAUDE.md`. Sem
`mcpsrv`, `ipc`, `daemon` ou `doctor`.

### `mirrorReader` movido verbatim

```
$ git show 5e89964:cmd/gobsidian/serve.go | sed -n '/^\/\/ mirrorDst/,$p' > old_mirror.go
$ tail -n +3 internal/boot/espelho.go > new_mirror.go
$ diff -u old_mirror.go new_mirror.go
@@ -1,3 +1,5 @@
+import "io"
+
 // mirrorDst e o subconjunto de *io.PipeWriter que mirrorReader usa. Extrair
```

Uma linha de diferença — o import que o arquivo novo precisa. Nada mais.

### Os três testes movidos, contra os apagados

```
$ git show 5e89964:cmd/gobsidian/serve_test.go | sed -n '1,215p' > old_tests.txt
$ diff -u old_tests.txt internal/boot/espelho_test.go
@@ -1,12 +1,9 @@
-package main
+package boot

 import (
 	"bytes"
-	"context"
 	"errors"
-	"fmt"
 	"io"
-	"os"
 	"testing"
 	"time"
```

O resto do diff é só a cauda que o `sed -n '1,215p'` cortou. Nenhuma asserção
enfraquecida, nenhuma removida: `TestMirrorReaderCopiesToMirror`,
`TestMirrorReaderPropagatesEOF` e
`TestMirrorReaderBrokenMirrorDoesNotPoisonRead` chegaram inteiros, com os
prazos de `vaulttest.Prazo`, os `t.Cleanup` e as três asserções de
`dst.writes` do teste do latch `broken`.

### Fechamento duplo do `io.Pipe` — fonte do toolchain, não suposição

```
$ go env GOROOT
C:\Program Files\Go

$ sed -n '/func (p \*pipe) closeWrite/,/^}/p' "$(go env GOROOT)/src/io/pipe.go"
func (p *pipe) closeWrite(err error) error {
	if err == nil {
		err = EOF
	}
	p.werr.Store(err)
	p.once.Do(func() { close(p.done) })
	return nil
}

$ sed -n '/func (a \*onceError) Store/,/^}/p' "$(go env GOROOT)/src/io/pipe.go"
func (a *onceError) Store(err error) {
	a.Lock()
	defer a.Unlock()
	if a.err != nil {
		return
	}
	a.err = err
}
```

`Close` e `CloseWithError` concorrentes são seguros: o erro é gravado sob mutex
e só o primeiro fica; o canal fecha sob `sync.Once`. Nenhum caminho de panic e
nenhum erro espúrio no passo `close-pipe`.

### Mutação do brief (Step 4), re-rodada por mim às 11:39

```
$ pwsh -File scripts/mutate.ps1 -Path internal/boot/espelho.go `
    -Anchor '_ = m.dst.CloseWithError(err)' -Replacement '_ = err' `
    -Test TestMirrorReaderPropagatesEOF -Package ./internal/boot/

[...] Mutando internal/boot/espelho.go
      - _ = m.dst.CloseWithError(err)
      + _ = err

[...] go test -race -run TestMirrorReaderPropagatesEOF ./internal/boot/
----------------------------------------------------------------------
--- FAIL: TestMirrorReaderPropagatesEOF (5.00s)
    espelho_test.go:178: lado de leitura do pipe nao recebeu EOF em 5s: mirrorReader nao fechou dst
FAIL
FAIL	github.com/jonyd/gobsidian/internal/boot	5.673s
FAIL
----------------------------------------------------------------------
[OK] internal/boot/espelho.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

Bate com o que o relatório colou.

### Mutação de sondagem (achado N1), 11:39

Primeiro contra o teste novo:

```
$ pwsh -File scripts/mutate.ps1 -Path internal/boot/vigia.go `
    -Anchor 'return v.pw.Close()' -Replacement 'return nil' `
    -Test TestVigiarHostEOFDoStdinCancelaCtx -Package ./internal/boot/

[...] Mutando internal/boot/vigia.go
      - return v.pw.Close()
      + return nil

[...] go test -race -run TestVigiarHostEOFDoStdinCancelaCtx ./internal/boot/
----------------------------------------------------------------------
ok  	github.com/jonyd/gobsidian/internal/boot	1.692s
----------------------------------------------------------------------
[OK] internal/boot/vigia.go restaurado byte a byte (SHA-256 confere).

[!] O teste PASSOU com a regra mutada.
    TestVigiarHostEOFDoStdinCancelaCtx nao consegue reprovar sem essa regra: ela esta escrita, nao verificada.
EXIT=1
```

Depois contra o pacote inteiro, para não afirmar mais do que medi:

```
$ pwsh -File scripts/mutate.ps1 -Path internal/boot/vigia.go `
    -Anchor 'return v.pw.Close()' -Replacement 'return nil' `
    -Test 'Test' -Package ./internal/boot/

[...] go test -race -run Test ./internal/boot/
----------------------------------------------------------------------
ok  	github.com/jonyd/gobsidian/internal/boot	2.355s
----------------------------------------------------------------------
[OK] internal/boot/vigia.go restaurado byte a byte (SHA-256 confere).

[!] O teste PASSOU com a regra mutada.
EXIT=1
```

Os dois `mutate.ps1` restauraram `internal/boot/vigia.go` e
`internal/boot/espelho.go` byte a byte, com SHA-256 conferido pelo próprio
script. `git status --porcelain` sobre `*.go`, `CLAUDE.md` e `docs/` continua
vazio.

### Strings e chamadas conferidas por `grep`

- `"stdin-eof"` — `internal/lifecycle/stdin.go:32`, igual à do teste
  (`vigia_test.go:27`).
- `lifecycle.ParentPID()` — chamada em `internal/boot/vigia.go:41` (portanto no
  caminho de `serve` e da ponte) e em lugar nenhum de `daemon.go`.
- `lifecycle.Shutdown` — só em `cmd/gobsidian/{serve,ponte,daemon}.go`; nunca em
  `internal/boot`.
- Nenhum `net/*` novo, nenhum tipo do SDK MCP fora de `internal/mcpsrv`,
  nenhum `helpers.go`/`utils.go`/`common.go`.

### O que eu NÃO verifiquei

- `scripts/test_orphans.ps1` — proibido nesta revisão; roda destacado pelo
  orquestrador. Não afirmo nada sobre os quatro cenários.
- `scripts/verify.ps1` e `scripts/build.ps1` inteiros — não rodados. Rodei as
  etapas que esta Task toca (build, vet, `-race`, `golangci-lint`) e todas
  passaram; o "14 etapas verde" segue sendo afirmação do relatório, com a
  saída colada lá.
- O RED do Step 1 — não reproduzível sem editar a árvore.
