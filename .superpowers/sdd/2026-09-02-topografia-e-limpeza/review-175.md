# Review: Task 175 -- `search` de CLI abre indice e busca via `boot`

Commit: `8d3626e` (base `ebf29ad`)

## Spec compliance

**Verdict: PASS.**

- Step 1 (`LogLevelExplicito`): campo e comentario literais do brief em
  `internal/config/config.go:29,55,82-95`; `Load` marca `true` nos DOIS ramos
  (env `:384` e flag `:392`), confirmado no diff e por leitura direta do
  arquivo. `TestLoadMarcaLogLevelExplicito` roda PASS aqui
  (`go test ./internal/config/... -run TestLoadMarcaLogLevelExplicito -v`).
- Step 2 (`cli_log.go`/`loggerDeCLI`): texto identico ao brief, escreve em
  `cmd.ErrOrStderr()`, nivel `Warn` por padrao e `cfg.LogLevel` quando
  `LogLevelExplicito`.
- Step 3 (`search.go`): `boot.AbrirIndice` + `inv.MarkBuilding()` +
  `boot.PrepararBusca` substituem `index.New()+Build`+laco `inv.Update`;
  import de `index` removido; flags `--cache-dir`/`--log-level` acrescentadas
  como no brief. `go build ./...` e `go vet ./...` limpos (confirmado aqui).
- Step 4: `TestSearchCmd_StdoutAndJSON` e `TestSearchCLIRespeitaMaxResults`
  ganharam `--cache-dir` de `t.TempDir()` (o brief so citava o primeiro; o
  segundo tambem passa `--vault` real e precisava do mesmo tratamento --
  correcao correta e mais completa que o brief). `search_cache_test.go` e
  literal ao brief. `go test -race ./cmd/gobsidian/ -run 'TestSearch' -v`:
  PASS nos tres, confirmado aqui e no relatorio.
- Step 5: prova manual de mutacao colada em tempo passado, com a mensagem de
  falha correta (`primeira execucao devia construir...`). Nao reproduzi a
  mutacao eu mesmo (fora do escopo: revisao e read-only e nao pode editar
  codigo), mas a cadeia que explica POR QUE falharia bate: revertendo para o
  bloco antigo, nenhuma linha `origem=` e emitida (so `boot.PrepararBusca`
  loga `origem=cache`/`origem=construcao`, ver `internal/boot/busca.go:144,249`)
  -- a mensagem de FAIL colada e exatamente essa.
- Step 6: 12 linhas presentes e coladas no relatorio, publicadas em
  `docs/ESTADO.md` (+1 linha, formato consistente com as entradas vizinhas) e
  resumidas em secao propria de `docs/OPERACAO.md`. O relatorio e os dois
  documentos **dizem explicitamente** que os numeros nao batem com a faixa do
  M4 e por que (maquina/estado de disco diferente, primeira chamada "antes"
  em 8,3 s sugerindo I/O de SO frio) -- nao ha alegacao de equivalencia ao M4
  em lugar nenhum dos tres textos. A RELACAO que a Task promete (quente <
  1/5 de fria) esta calculada e citada com os numeros: fria ~1797-1921 ms,
  1/5 ~359-384 ms, quente medida 190-235 ms. Sem hedge (nao ha "tende a" ou
  "deveria" ao lado do resultado).
- Step 7 (README): `search` entrou na coluna Subcommands das linhas
  ja existentes de `--cache-dir` e `--log-level` (matriz, nao linha nova --
  ja era a leitura orquestrada) e a nota de uma linha foi acrescentada.
- Step 8: `verify.ps1` verde colado, 2a rodada, com a 1a rodada (reprovada em
  errcheck) tambem colada e a correcao justificada (`defer func(){ _ =
  inv.Close() }()`, ja o padrao em `busca_test.go`, `daemon.go:50`,
  `ponte_test.go`). Commit `8d3626e` tem exatamente os 9 arquivos do brief
  (confirmado com `git show --stat`), staged por caminho explicito -- nada
  do trabalho nao commitado do dono (test-vault/, troglodita*, etc.) entrou
  (confirmado com `git status` antes/depois).

## Code quality

**Verdict: PASS.**

- `stdout` continua so do resultado (`cmd.OutOrStdout()`), log so em
  `cmd.ErrOrStderr()` via `loggerDeCLI` -- nenhum `fmt.Print*` no caminho de
  log.
- `config` nao ganhou import novo (so o campo bool).
- Nenhuma aresta de import nova: `cmd/gobsidian -> internal/boot` ja existia
  via `serve.go` e `daemon.go`; `search.go` reusa a mesma aresta.
- `PrepararBusca` e sincrona (nao dispara goroutine por si so); em
  `search.go` ela bloqueia o `RunE` ate `inv` ficar pronto ou o `ctx`
  cancelar, exatamente como o comentario da funcao promete. Se o `ctx`
  cancelar no meio (`construirBusca`, `internal/boot/busca.go:196-205`), a
  funcao sai SEM chamar `inv.MarkReady()` e sem gravar; a chamada seguinte a
  `svc.Search` (`internal/service/search.go:172-175`) checa
  `s.inverted.Building()` e devolve `CodeIndexBuilding` em vez de servir um
  indice pela metade -- `search` nunca consulta um `inv` nao-pronto.
  Gravacao de cache passa por `vault.ReplaceFile` (temp+rename) nos dois
  lados (`internal/search/persist.go:77`, `internal/index/persist.go:115`),
  entao um cancelamento durante a escrita nao deixa arquivo parcial visivel
  -- isso e uma garantia do primitivo, nao algo que esta Task precisou
  adicionar.
- `cli_log.go` e nome legitimo sob a regra "sem helpers/utils/common": tem
  uma unica funcao, `loggerDeCLI`, com responsabilidade unica e nomeada (o
  logger dos subcomandos de CLI) -- nao e um saco de funcoes desconexas
  agrupadas por "seria util".
- `TestSearchCLISegundaExecucaoUsaCache` prende o comportamento certo: a
  asercao depende da linha `origem=` que so `PrepararBusca` emite: se ela
  fosse pulada, nenhuma das duas comparacoes (`origem=construcao` /
  `origem=cache`) apareceria em stderr, e o Step 5 comprova exatamente isso
  em tempo passado.
- `TestSearchCmd_StdoutAndJSON` e `TestSearchCLIRespeitaMaxResults` (as
  outras duas ocorrencias de `SetArgs` com `--vault` real em `search`)
  ganharam `--cache-dir` de `t.TempDir()`; as demais ocorrencias sem
  `--cache-dir` (`index`/`inspect`, linhas 41/60/130/148 de
  `cli_subcommands_test.go`) nao usam `boot`/cache algum -- confirmado que
  `index.go`/`inspect.go` ainda chamam `index.New()+Build` direto, sem
  `CacheDir` -- fora do escopo desta Task (Task 176, conforme o brief).

## Findings

Nenhum achado bloqueante, de correcao obrigatoria ou nit.

## Verdict

APROVADO -- spec compliance PASS, code quality PASS, nenhum finding.
