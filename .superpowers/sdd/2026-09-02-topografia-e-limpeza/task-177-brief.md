### Task 177: e PR2 — `boot.VigiarHost` e os passos de shutdown que `serve`, ponte e `daemon` repetem

**Files:**
- Create: `internal/boot/vigia.go` (`Vigia`, `VigiarHost`, `PassoFecharEspelho`), `internal/boot/espelho.go` (`mirrorReader`, `mirrorDst` movidos de `serve.go`), `internal/boot/vigia_test.go`
- Move: os testes `TestMirrorReaderCopiesToMirror:76`, `TestMirrorReaderPropagatesEOF:143`, `TestMirrorReaderBrokenMirrorDoesNotPoisonRead:189` de `cmd/gobsidian/serve_test.go` → `internal/boot/espelho_test.go` (pacote `boot`; `boundedWait` → `vaulttest.Prazo` com `select`)
- Modify: `internal/boot/montar.go` (`(*Componentes).PassoWatcher() lifecycle.Step`)
- Modify: `cmd/gobsidian/serve.go:381-430` (`serveEmProcesso`), `cmd/gobsidian/ponte.go:155-257` (`servePonteRemota`), `cmd/gobsidian/daemon.go:150-184`
- Modify: `CLAUDE.md` (grafo: `boot` ganha `lifecycle`), `docs/ARCHITECTURE.md`

**Interfaces:**
- Consumes: `lifecycle.New(parent, Options{Stdin, ParentPID, Logger}) (ctx, *Lifecycle)`, `lifecycle.ParentPID()`, `lifecycle.Step{Name, Budget, Fn}`, `lifecycle.Shutdown(ctx, log, hardLimit, steps...)`, `(*Lifecycle).Reason()/Wait()`; `boot.Componentes` (Task 174); `vaulttest.Prazo` (Task 159).
- Produces:

```go
// Vigia e o que o processo precisa para saber quando o host foi embora:
// o ctx que cancela, o Lifecycle que explica por que, e o stdin espelhado
// que o servidor MCP le enquanto o lifecycle vigia o EOF.
type Vigia struct {
	Ctx   context.Context
	LC    *lifecycle.Lifecycle
	Stdin io.Reader
	pw    *io.PipeWriter
}

// VigiarHost liga a vigilia: EOF em stdin ou morte do PID pai cancelam Ctx.
func VigiarHost(parent context.Context, stdin io.Reader, log *slog.Logger) *Vigia

// PassoFecharEspelho fecha o pipe para que o leitor de stdin do servidor
// receba EOF; 500 ms de orcamento.
func (v *Vigia) PassoFecharEspelho() lifecycle.Step

// PassoWatcher fecha o watcher; 500 ms de orcamento.
func (c *Componentes) PassoWatcher() lifecycle.Step
```

Aresta nova: `boot → lifecycle`. Justificativa: o mesmo andaime pipe + `mirrorReader` + `lifecycle.New` está em `serve.go:384-391` e em `ponte.go:161-168`, e os passos `close-pipe`/`watcher` estão nos três pontos de saída. `lifecycle` continua folha.

- [ ] **Step 1: Teste da vigília — falha por compilação**

`internal/boot/vigia_test.go`:

```go
package boot_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/boot"
	"github.com/jonyd/gobsidian/internal/vaulttest"
)

func TestVigiarHostEOFDoStdinCancelaCtx(t *testing.T) {
	pr, pw := io.Pipe()
	v := boot.VigiarHost(context.Background(), pr, logSilencioso())
	// O servidor leria v.Stdin; aqui basta drenar para o espelho nao travar.
	go func() { _, _ = io.Copy(io.Discard, v.Stdin) }()
	if _, err := pw.Write([]byte("{}\n")); err != nil {
		t.Fatal(err)
	}
	_ = pw.Close()
	select {
	case <-v.Ctx.Done():
	case <-time.After(vaulttest.Prazo):
		t.Fatal("EOF em stdin nao cancelou o ctx")
	}
	if r := v.LC.Reason(); r != "stdin-eof" {
		t.Fatalf("Reason = %q, quero \"stdin-eof\"", r)
	}
	passo := v.PassoFecharEspelho()
	if passo.Name != "close-pipe" || passo.Budget != 500*time.Millisecond {
		t.Fatalf("PassoFecharEspelho = {%q %v}", passo.Name, passo.Budget)
	}
	if err := passo.Fn(context.Background()); err != nil {
		t.Fatalf("fechar o espelho: %v", err)
	}
}
```

`logSilencioso` é o de `indice_test.go` (mesmo pacote `boot_test`). A assinatura de `Step.Fn` é a real de `lifecycle` (conferir se recebe `ctx`; ajustar a chamada). Run: `go test ./internal/boot/ -run TestVigiarHostEOFDoStdinCancelaCtx` — Expected: FAIL de compilação.

- [ ] **Step 2: Mover `mirrorReader` e criar `Vigia`**

`internal/boot/espelho.go`: `mirrorReader` e `mirrorDst` de `serve.go`, verbatim, com o comentário de porquê o espelho existe. A linha `_ = m.dst.CloseWithError(err)` fica **exatamente** assim — âncora da mutação.

`internal/boot/vigia.go`:

```go
func VigiarHost(parent context.Context, stdin io.Reader, log *slog.Logger) *Vigia {
	pr, pw := io.Pipe()
	ctx, lc := lifecycle.New(parent, lifecycle.Options{
		Stdin:     pr,
		ParentPID: lifecycle.ParentPID(),
		Logger:    log,
	})
	return &Vigia{
		Ctx:   ctx,
		LC:    lc,
		Stdin: &mirrorReader{src: stdin, dst: pw},
		pw:    pw,
	}
}

func (v *Vigia) PassoFecharEspelho() lifecycle.Step {
	return lifecycle.Step{Name: "close-pipe", Budget: 500 * time.Millisecond, Fn: func(context.Context) error {
		return v.pw.Close()
	}}
}
```

Isto é o que `serve.go:384-391` faz hoje; copiar dali a ordem exata (quem é `src`, quem é `dst`, o que `lifecycle.New` recebe) em vez de confiar no esboço.

`internal/boot/montar.go`:

```go
func (c *Componentes) PassoWatcher() lifecycle.Step {
	return lifecycle.Step{Name: "watcher", Budget: 500 * time.Millisecond, Fn: func(context.Context) error {
		return c.Watcher.Close()
	}}
}
```

Run: `go test -race ./internal/boot/ -v` — Expected: PASS em todos (inclusive os três `TestMirrorReader*` movidos).

- [ ] **Step 3: Os três pontos de saída usam `Vigia`**

`serveEmProcesso` (`serve.go:381`): `vig := boot.VigiarHost(parent, os.Stdin, log)`; `ctx := vig.Ctx`; `c, err := boot.Montar(ctx, cfg, log)`; `mcpsrv.New(ctx, c.Service, cfg, log)` lê de `vig.Stdin` (onde hoje lê de `pr`); `lifecycle.Shutdown(ctx, log, 6*time.Second, <passo in-flight 3 s como hoje>, vig.PassoFecharEspelho(), c.PassoWatcher())`; `vig.LC.Wait(); c.Esperar()`.

`servePonteRemota` (`ponte.go:155`): mesmo andaime; os passos `half-close` (2 s, `conn.CloseWrite`) e `close-conn` ficam onde estão, `close-pipe` vira `vig.PassoFecharEspelho()`.

`daemon.go:150-184`: não tem stdin; continua com `lifecycle.New(parent, Options{Logger})`, e o passo `watcher` vira `c.PassoWatcher()`.

Apagar `mirrorReader`/`mirrorDst` de `serve.go`; `git mv` dos três testes para `internal/boot/espelho_test.go`; `TestShutdownExitCode:235` fica em `serve_test.go`.

Run: `go build ./... && go vet ./... && go test -race ./cmd/... ./internal/boot/` — Expected: PASS.

- [ ] **Step 4: Prova de mutação**

```powershell
pwsh -File scripts/mutate.ps1 -Path internal/boot/espelho.go -Anchor '_ = m.dst.CloseWithError(err)' -Replacement '_ = err' -Test TestMirrorReaderPropagatesEOF -Package ./internal/boot/
```

Expected: exit 0 (sem propagar o EOF ao espelho, o lifecycle nunca vê stdin fechar). Colar.

- [ ] **Step 5: Órfãos — o teste que importa nesta Task**

Run: `pwsh -File scripts/test_orphans.ps1` — Expected: quatro `[OK]`. Colar os quatro. Se um falhar, é esta Task: o andaime mudou de lugar e algo ficou fora de ordem (o `mirrorReader` precisa existir **antes** do `lifecycle.New` receber o `pr`).

- [ ] **Step 6: Grafo e documentação**

Run: `go list -f '{{.Imports}}' ./internal/boot/ | tr ' ' '\n' | grep gobsidian` — Expected: `config index lifecycle search service vault watcher`.
`CLAUDE.md`: linha `boot → config, index, lifecycle, search, service, vault, watcher` e a justificativa da aresta em uma frase (o andaime duplicado). `docs/ARCHITECTURE.md`: parágrafo sobre `VigiarHost` na seção de ciclo de vida.

- [ ] **Step 7: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/boot/ cmd/gobsidian/serve.go cmd/gobsidian/serve_test.go cmd/gobsidian/ponte.go cmd/gobsidian/daemon.go CLAUDE.md docs/ARCHITECTURE.md
git commit -m "refactor(boot): host watch and shutdown steps shared by serve, bridge and daemon"
```

#### Verificações

- Step 1 FAIL→PASS colado; `mutate.ps1` exit 0 colado.
- `test_orphans.ps1` quatro `[OK]` colados.
- `go list` do Step 6 colado.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. `git mv` por caminho é permitido.
- Orçamentos de shutdown (6 s total, 3 s in-flight, 500 ms pipe, 500 ms watcher, 2 s half-close) **não mudam**: são os medidos em `test_orphans.ps1`.
- `boot` não importa `mcpsrv`, `ipc`, `daemon`, `doctor`.
- Não rodar `test_orphans.ps1` em paralelo com qualquer medição.

#### Comando de mutação

Step 4 (`mutate.ps1`, âncora `_ = m.dst.CloseWithError(err)` em `internal/boot/espelho.go`).

#### Contrato de relatório

`task-177-report.md`: status, SHA, saídas dos Steps 1, 4, 5, 6, última linha do `verify.ps1`.

---
## Fase 5 — Contratos e fecho (Tasks 178–181)

Modelos: 178 Sonnet; 179 Sonnet; 180 Opus; 181 Sonnet. A Fase 5 muda o contrato
das tools em três pontos decididos pelo dono (`hits` some; `include` valida;
tags dobram) e fecha a dívida 1.12 com testes de caixa-branca do codec.

