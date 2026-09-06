# Task 175: `search` de CLI abre indice e busca como o `serve`

Base: ebf29ad3ae726905fa5e5d88abe35476c914534a
Status: DONE
SHA: 8d3626e1f18b9837cf1ba5210d3666ff4316c532

## Achados de `scripts/audit_reports.ps1 175` (exit 1, 17 achados no total do repo)

Achados NESTE relatorio, respondidos:

- `[HEDGE]` no paragrafo "Divergencia do esperado" (Step 6): nao e um valor
  nao medido apresentado como resultado -- e o oposto: sao numeros medidos de
  verdade (colados acima) com uma nota honesta de que nao batem com a faixa
  do M4 e de que a causa da diferenca nao foi investigada. E exatamente o
  "nao medido"/"nao verificado" que o `CLAUDE.md` pede em vez de uma alegacao
  sem base. Revisado, nao e defeito.
- `[SECAO-AUSENTE] sem secao de TDD/RED, sem secao de TDD/GREEN`: o script
  procura as palavras "red"/"green"; este brief (Step 1, 4 e 5) pede FAIL e
  PASS, nao RED/GREEN, e o relatorio tem as duas evidencias coladas (Step 1
  FAIL->PASS, Step 5 FAIL da mutacao manual). Falso positivo de vocabulario,
  nao ausencia de prova.

Achados no `=== Ledger ===`: todos em `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`,
de um marco anterior e de tarefas (4, 6, 94-103, 153) que esta task nao toca
e cujo commit `8d3626e` nao altera. Fora do escopo da Task 175 -- nao
respondidos aqui.

## Progresso

- 10:19 Lido brief, CLAUDE.md, implementador.md, search.go, config.go, boot/indice.go, boot/busca.go, cli_subcommands_test.go, ponte.go. Confirmado: `search` nao passa por `servePonte`/daemon (nao importa `ponte.go`, chama `vault.New` direto) -- nao precisa de `GOBSIDIAN_NO_DAEMON`.
- 10:20 Step 1: `LogLevelExplicito` em `config.Config` + `Load`. FAIL/PASS provados abaixo.
- 10:20 Step 2: criado `cmd/gobsidian/cli_log.go` com `loggerDeCLI`.
- 10:21 Step 3: `search.go` usa `boot.AbrirIndice` + `boot.PrepararBusca`; flags `--cache-dir`/`--log-level` acrescentadas; import de `index` removido. `go build ./...` e `go vet ./...` limpos.
- 10:22 Step 4: `TestSearchCmd_StdoutAndJSON` e `TestSearchCLIRespeitaMaxResults` ganharam `--cache-dir` de `t.TempDir()` (grep confirmou serem os dois outros SetArgs de `search` com `--vault` real, alem do novo teste). Criado `cmd/gobsidian/search_cache_test.go` com `TestSearchCLISegundaExecucaoUsaCache`. `go test -race ./cmd/gobsidian/ -run 'TestSearch' -v`: PASS nos tres.
- 10:24 Step 5: prova manual de mutacao, FAIL colado abaixo, bloco restaurado, PASS confirmado de novo.
- 10:26 Step 6: worktree Git separado (`git worktree add --detach ... ebf29ad`) para o binario `antes`, sem tocar a arvore principal; `bin/gobsidian.exe` via `scripts/build.ps1` para o `depois`. 12 linhas medidas com `Measure-Command` no `vault_5000`. Numeros colados abaixo e publicados em `docs/ESTADO.md` e resumidos em `docs/OPERACAO.md`.
- 10:28 Step 7: README (`README.md:205,207`) -- `search` acrescentado a coluna Subcommands de `--cache-dir` e `--log-level`; nota de uma linha adicionada apos a tabela.
- 10:30 Step 8, 1a rodada de `verify.ps1`: reprovou em `golangci-lint`/`golangci-lint (linux)` -- `cmd\gobsidian\search.go:46:19: Error return value of inv.Close is not checked (errcheck)`. **Correcao mecanica ao codigo literal do brief:** o snippet do Step 3 tinha `defer inv.Close()`; troquei para `defer func() { _ = inv.Close() }()`, o mesmo padrao ja usado em `internal/boot/busca_test.go:23,77` e em outros `defer` de `Close()` deste pacote (`cmd/gobsidian/daemon.go:50`, `ponte_test.go:112,159`). `go build ./...` e `go test -race ./cmd/gobsidian/ -run 'TestSearch' -v` confirmados de novo apos a correcao (PASS nos tres).
- 10:39 Step 8, 2a rodada de `verify.ps1`: verde, `[OK] Bateria completa. Pode commitar.` (saida completa colada abaixo).
- 10:40 Commit `8d3626e1f18b9837cf1ba5210d3666ff4316c532` (`git commit -F .superpowers/sdd/2026-09-02-topografia-e-limpeza/commit-175.txt`), 9 arquivos, exatamente os do brief mais `cli_log.go` e `search_cache_test.go` (novos). Trabalho nao-commitado do dono (test-vault/, .claude/skills/troglodita*, Resume-Claude.ps1, o ledger de outras tasks) nao entrou -- staged por caminho explicito.

## Step 1 -- FAIL depois PASS

FAIL (campo removido temporariamente para provar):
```
$ go test ./internal/config/ -run TestLoadMarcaLogLevelExplicito -v
# github.com/jonyd/gobsidian/internal/config
internal\config\config.go:88:7: cfg.LogLevelExplicito undefined (type Config has no field or method LogLevelExplicito)
internal\config\config.go:96:7: cfg.LogLevelExplicito undefined (type Config has no field or method LogLevelExplicito)
FAIL	github.com/jonyd/gobsidian/internal/config [build failed]
FAIL
```

PASS (campo restaurado):
```
$ go test ./internal/config/ -run TestLoadMarcaLogLevelExplicito -v
=== RUN   TestLoadMarcaLogLevelExplicito
--- PASS: TestLoadMarcaLogLevelExplicito (0.01s)
PASS
ok  	github.com/jonyd/gobsidian/internal/config	0.424s
```

## Step 5 -- prova manual de mutacao

Bloco do Step 3 substituido temporariamente pelo antigo (`index.New()` + `Build` + laco `inv.Update` + `inv.MarkReady()`, com o import de `index` de volta e o de `boot` removido):

```
$ go test ./cmd/gobsidian/ -run TestSearchCLISegundaExecucaoUsaCache -v
=== RUN   TestSearchCLISegundaExecucaoUsaCache
    search_cache_test.go:33: primeira execucao devia construir o indice de busca:
--- FAIL: TestSearchCLISegundaExecucaoUsaCache (0.04s)
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	0.213s
FAIL
```

Restaurado o bloco do Step 3 (`boot.AbrirIndice` + `boot.PrepararBusca`). `go build ./...` limpo; `go test -race ./cmd/gobsidian/ -run 'TestSearch' -v` PASS nos tres de novo (ver saida abaixo).

## Step 6 -- Medicao M4 (search reaproveita o cache do serve)

Setup: worktree separado (`git worktree add --detach`, sem tocar a arvore principal) no commit base `ebf29ad`, `go build -o antes175_gobsidian.exe ./cmd/gobsidian/`. Binario `depois`: `pwsh -File scripts/build.ps1` -> `bin/gobsidian.exe`. Cofre `%TEMP%\vault_5000` (5050 arquivos, ja existia de medicoes anteriores desta maquina/sessao). Confirmado por `search --help`: o binario `antes` NAO tem `--cache-dir`/`--log-level` -- rodado sem a flag, como o brief previu (constroi tudo em memoria sempre).

`search` nao passa pelo daemon: `search.go` chama `vault.New` direto, nunca `servePonte`/`ipc`, entao os `gobsidian.exe` de outras sessoes rodando nesta maquina (vistos em `tasklist`, nao lancados por mim, nao tocados) nao interferem na medicao.

Comando por rodada: `Measure-Command { & $exe search --json --limit 200 --vault $vault ["--cache-dir" $cache] "execucao" }`.

antes (sem `--cache-dir`, constroi tudo sempre -- 3 "frias" e 3 "quentes" sao a mesma coisa, so para ter as 6 linhas pedidas):
```
antes cold 1 : 8338.6069
antes cold 2 : 1687.0217
antes cold 3 : 1631.8712
antes warm 1 : 1700.4421
antes warm 2 : 1631.0050
antes warm 3 : 1597.6964
```

depois (`--cache-dir` novo, apagado antes de cada rodada fria; as 3 quentes reusam o mesmo diretorio ja gravado):
```
depois cold 1 : 1905.5605
depois cold 2 : 1921.2205
depois cold 3 : 1796.6712
depois warm 1 : 234.6872
depois warm 2 : 214.0464
depois warm 3 : 190.1828
```

Verificacao de que a busca devolveu resultado real (nao um cache vazio): `search --json --limit 5` no cache quente devolveu 5 acertos com `path`, `score` e `snippet` preenchidos (colado no log de trabalho, nao repetido aqui).

**Divergencia do esperado:** o brief esperava `antes` fria ~ quente ~ M4 (10520/10819/11046 ms) e `depois` quente perto de 475-528 ms. Medi `antes` bem mais rapido (1,6-1,7 s, exceto a primeira rodada em 8,3 s -- provavel cache de disco do SO ainda frio nessa primeira chamada) e `depois` quente em 190-235 ms, tambem mais rapido que a faixa citada. Nao investiguei a causa da diferenca com M4 (maquina/estado de disco diferente das medicoes anteriores) -- reporto os numeros medidos agora, nao os do M4. A RELACAO que a Task promete se sustenta: `depois` quente (190-235 ms) fica bem abaixo de 1/5 de `depois` fria (1796-1921 ms; 1/5 = ~359-384 ms), e `depois` fria fica perto de `antes` (1,6-1,9 s) mais o custo extra de gravar os dois caches na primeira execucao.

## Step 4 -- PASS

```
$ go test -race ./cmd/gobsidian/ -run 'TestSearch' -v
=== RUN   TestSearchCmd_StdoutAndJSON
--- PASS: TestSearchCmd_StdoutAndJSON (0.14s)
=== RUN   TestSearchCLIRespeitaMaxResults
--- PASS: TestSearchCLIRespeitaMaxResults (0.07s)
=== RUN   TestSearchCLISegundaExecucaoUsaCache
--- PASS: TestSearchCLISegundaExecucaoUsaCache (0.12s)
PASS
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	2.793s
```

## Step 8 -- `verify.ps1`

1a rodada (reprovou, `errcheck`), corrigida a `defer inv.Close()` para `defer func() { _ = inv.Close() }()`, 2a rodada verde:

```
Carregado em 412ms
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
Carregamento: 791,2 ms 

[OK] check_tool_params
[...] 13. check_doc_refs
[OK] check_doc_refs
[...] 14. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

Os 6 testes pulados sao os mesmos de sempre (skips legitimos ja registrados no projeto, nenhum deles em `cmd/gobsidian`, `internal/config` ou `internal/boot`).
