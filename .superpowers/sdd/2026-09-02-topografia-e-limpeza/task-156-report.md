# Task 156 report

## Status
DONE

## Commit
b8ed7f6 fix(cli): search honours --max-results; index and inspect stop declaring flags they ignore

(Doc-only follow-up, separate commit per orchestrator instruction:
b354dcf docs(wiki): shutdownExitCode delegates to ipc.EhDesconexaoLimpa, os.ErrClosed is a clean disconnect)

## Evidência de TDD

RED — reproduzido depois do commit trocando temporariamente
`search.go`/`index.go`/`inspect.go` pela versão pré-fix (`git show 3a1ab3e:...`,
nunca `git checkout`), mantendo o teste novo:

```
$ go test ./cmd/gobsidian/ -run 'TestSearchCLIRespeitaMaxResults|TestIndexEInspectNaoAceitamFlagsQueIgnoram' -v
=== RUN   TestSearchCLIRespeitaMaxResults
    cli_subcommands_test.go:197: results = 5 com --max-results 2: a flag e lida e descartada
--- FAIL: TestSearchCLIRespeitaMaxResults (0.07s)
=== RUN   TestIndexEInspectNaoAceitamFlagsQueIgnoram
    cli_subcommands_test.go:212: index declara --read-only e nao a usa
    cli_subcommands_test.go:212: index declara --debounce-ms e nao a usa
    cli_subcommands_test.go:212: index declara --max-results e nao a usa
    cli_subcommands_test.go:212: inspect declara --read-only e nao a usa
    cli_subcommands_test.go:212: inspect declara --debounce-ms e nao a usa
    cli_subcommands_test.go:212: inspect declara --max-results e nao a usa
    cli_subcommands_test.go:218: search declara --read-only e nao a usa
    cli_subcommands_test.go:218: search declara --debounce-ms e nao a usa
--- FAIL: TestIndexEInspectNaoAceitamFlagsQueIgnoram (0.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	1.855s
FAIL
```

Arquivos restaurados a partir de cópia local (não `git checkout`); `git diff --stat`
sobre os três arquivos, imediatamente após a restauração, vazio.

GREEN:

```
$ go test ./cmd/gobsidian/ -run 'TestSearchCLIRespeitaMaxResults|TestIndexEInspectNaoAceitamFlagsQueIgnoram' -v
=== RUN   TestSearchCLIRespeitaMaxResults
--- PASS: TestSearchCLIRespeitaMaxResults (0.06s)
=== RUN   TestIndexEInspectNaoAceitamFlagsQueIgnoram
--- PASS: TestIndexEInspectNaoAceitamFlagsQueIgnoram (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	(cached)
```

## Prova de mutação

Comando do brief, rodado sobre a árvore no estado do commit b8ed7f6:

```
$ pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/search.go `
    -Anchor 'MaxResults: cfg.MaxResults' -Replacement 'MaxResults: 0' `
    -Test TestSearchCLIRespeitaMaxResults -Package ./cmd/gobsidian/
[...] Mutando cmd/gobsidian/search.go
      - MaxResults: cfg.MaxResults
      + MaxResults: 0

[...] go test -race -run TestSearchCLIRespeitaMaxResults ./cmd/gobsidian/
----------------------------------------------------------------------
--- FAIL: TestSearchCLIRespeitaMaxResults (0.06s)
    cli_subcommands_test.go:197: results = 5 com --max-results 2: a flag e lida e descartada
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	2.665s
FAIL
----------------------------------------------------------------------
[OK] cmd/gobsidian/search.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

`git diff --stat cmd/gobsidian/search.go` depois: vazio.

A segunda regra (`index`/`inspect` não declaram as três flags; `search` não
declara `--read-only`/`--debounce-ms`) é uma ausência de `Flags().Lookup`, não
um cálculo — não há âncora de mutação sensata para "flag não declarada". A
prova é a própria RED acima: com as flags ainda declaradas (código pré-fix),
`TestIndexEInspectNaoAceitamFlagsQueIgnoram` reprova nomeando as oito
combinações; com as flags removidas, passa. Cobertura da regra fica no
RED/GREEN, não num `mutate.ps1` separado.

## As verificações do brief

1. `--help` dos três subcomandos, binário buildado a partir da árvore commitada:

```
=== search --help ===
Uso:
  gobsidian search <consulta> [flags]

Flags:
      --follow-symlinks   segue symlink dentro do cofre; o padrao recusa, porque o confinamento nao alcanca o alvo
  -h, --help              help for search
      --json              saida estruturada em formato JSON
      --limit int         limite maximo de resultados (default 20)
      --max-results int   teto de resultados por consulta
      --vault string      caminho da raiz do cofre (obrigatorio)

=== index --help ===
Uso:
  gobsidian index [flags]

Flags:
      --follow-symlinks   segue symlink dentro do cofre; o padrao recusa, porque o confinamento nao alcanca o alvo
  -h, --help              help for index
      --json              saida estruturada em formato JSON
      --vault string      caminho da raiz do cofre (obrigatorio)

=== inspect --help ===
Uso:
  gobsidian inspect <nota> [flags]

Flags:
      --follow-symlinks   segue symlink dentro do cofre; o padrao recusa, porque o confinamento nao alcanca o alvo
  -h, --help              help for inspect
      --json              saida estruturada em formato JSON
      --vault string      caminho da raiz do cofre (obrigatorio)
```

`search` mostra `--max-results`, não mostra `--read-only`/`--debounce-ms`;
`index` e `inspect` não mostram nenhuma das três. Confirmado.

2. README — tabela agora tem coluna **Subcommands**, cada flag conferida contra
   `grep -n 'Flags()' cmd/gobsidian/*.go` antes de escrever a linha
   (`--vault`, `--follow-symlinks`: all; `--read-only`, `--cache-dir`,
   `--debounce-ms`, `--log-level`, `--eager-search`: `serve`; `--max-results`:
   `serve`, `search`; `--json`: `search`, `index`, `inspect`; `--limit`:
   `search`). `daemon`/`doctor` fora de escopo, deixados de fora da tabela
   por ruling do orquestrador.

```
$ pwsh -File scripts/check_readme_anchors.ps1
[i] 11 heading(s), 11 link(s) interno(s).
[OK] toda ancora resolve e toda secao H2 e alcancavel pela navegacao.
```

3. `cfg.MaxResults` chega a `service.Options` — provado pelo teste
   `TestSearchCLIRespeitaMaxResults` (GREEN acima) e pela mutação (EXIT=0
   acima). `--limit` continua funcionando como antes:

```
$ gobsidian search --vault <5 notas com "palavra"> --json --limit 3 palavra
{
  "hits": [ ... 3 itens ... ],
  "results": [ ... 3 itens ... ],
  "total": 5,
  "truncated": true,
  "effective_snippet_chars": 240,
  "effective_limit": 3
}
```

`results` tem 3 itens, `total: 5`, `truncated: true` — `--limit` intacto.

## `verify.ps1`

Rodado uma vez antes do primeiro commit deste lote (154) e uma vez depois do
último commit (o doc-only de encerramento.md, b354dcf), ambas as vezes
completas, ambas com:

```
[OK] Bateria completa. Pode commitar.
```

13 etapas, todas `[OK]` (build, test -race, test tetos de latencia sem -race,
vet windows/linux/darwin, gofmt, golangci-lint x2, check_net, check_tool_params,
check_doc_refs, check_readme_anchors) nas duas rodadas.

## O que ficou de fora

`daemon` e `doctor` continuam declarando flags — nenhuma delas foi auditada
nesta tarefa (fora de escopo por ruling do orquestrador). Não investiguei se
`daemon`/`doctor` têm o mesmo problema de flag declarada e ignorada.

## `git status --porcelain` (arquivos deste commit, no momento do commit)

```
M  README.md
M  cmd/gobsidian/cli_subcommands_test.go
M  cmd/gobsidian/index.go
M  cmd/gobsidian/inspect.go
M  cmd/gobsidian/search.go
```

Nenhum arquivo do usuário (`test-vault/`, `.claude/skills/`, `Resume-Claude.ps1`,
`.superpowers/`) tocado; `git add` foi por caminho explícito, nunca `-A`.

## Fix round 1

Achado bloqueante da revisão: a coluna **Subcommands** da tabela de flags
listava só `serve` para seis flags que `daemon` e `doctor` também declaram e
lêem. Ruling do orquestrador: opção (a) — escrever os subcomandos que o
código de fato declara, conferido linha a linha contra o grep abaixo.

```
$ grep -n "Flags()" cmd/gobsidian/*.go
cmd/gobsidian/daemon.go:33:			flags.ReadOnlySet = cmd.Flags().Changed("read-only")
cmd/gobsidian/daemon.go:34:			flags.DebounceMSSet = cmd.Flags().Changed("debounce-ms")
cmd/gobsidian/daemon.go:35:			flags.MaxResultsSet = cmd.Flags().Changed("max-results")
cmd/gobsidian/daemon.go:55:	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
cmd/gobsidian/daemon.go:56:	cmd.Flags().StringVar(&flags.LogLevel, "log-level", "", "debug, info, warn ou error")
cmd/gobsidian/daemon.go:57:	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "desabilita toda a superficie de escrita")
cmd/gobsidian/daemon.go:58:	cmd.Flags().IntVar(&flags.DebounceMS, "debounce-ms", 0, "janela de coalescencia de eventos do watcher")
cmd/gobsidian/daemon.go:59:	cmd.Flags().IntVar(&flags.MaxResults, "max-results", 0, "teto de resultados por consulta")
cmd/gobsidian/daemon.go:60:	cmd.Flags().StringVar(&flags.CacheDir, "cache-dir", "", "diretorio do cache de indice")
cmd/gobsidian/daemon.go:61:	cmd.Flags().BoolVar(&flags.FollowSymlinks, "follow-symlinks", false,
cmd/gobsidian/daemon.go:63:	cmd.Flags().BoolVar(&flags.EagerSearch, "eager-search", false,
cmd/gobsidian/daemon.go:65:	cmd.Flags().IntVar(&idleSeconds, "idle-seconds", daemon.DefaultIdleSeconds,
cmd/gobsidian/doctor.go:23:			flags.ReadOnlySet = cmd.Flags().Changed("read-only")
cmd/gobsidian/doctor.go:24:			flags.DebounceMSSet = cmd.Flags().Changed("debounce-ms")
cmd/gobsidian/doctor.go:25:			flags.MaxResultsSet = cmd.Flags().Changed("max-results")
cmd/gobsidian/doctor.go:67:	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
cmd/gobsidian/doctor.go:68:	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "nao verifica permissao de escrita")
cmd/gobsidian/doctor.go:69:	cmd.Flags().IntVar(&flags.DebounceMS, "debounce-ms", 0, "janela de coalescencia de eventos do watcher")
cmd/gobsidian/doctor.go:70:	cmd.Flags().IntVar(&flags.MaxResults, "max-results", 0, "teto de resultados por consulta")
cmd/gobsidian/doctor.go:71:	cmd.Flags().BoolVar(&flags.FollowSymlinks, "follow-symlinks", false,
cmd/gobsidian/index.go:85:	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
cmd/gobsidian/index.go:86:	cmd.Flags().BoolVar(&jsonOutput, "json", false, "saida estruturada em formato JSON")
cmd/gobsidian/index.go:87:	cmd.Flags().BoolVar(&flags.FollowSymlinks, "follow-symlinks", false,
cmd/gobsidian/inspect.go:123:	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
cmd/gobsidian/inspect.go:124:	cmd.Flags().BoolVar(&jsonOutput, "json", false, "saida estruturada em formato JSON")
cmd/gobsidian/inspect.go:125:	cmd.Flags().BoolVar(&flags.FollowSymlinks, "follow-symlinks", false,
cmd/gobsidian/search.go:26:			flags.MaxResultsSet = cmd.Flags().Changed("max-results")
cmd/gobsidian/search.go:88:	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
cmd/gobsidian/search.go:89:	cmd.Flags().BoolVar(&jsonOutput, "json", false, "saida estruturada em formato JSON")
cmd/gobsidian/search.go:90:	cmd.Flags().IntVar(&limit, "limit", 20, "limite maximo de resultados")
cmd/gobsidian/search.go:91:	cmd.Flags().BoolVar(&flags.FollowSymlinks, "follow-symlinks", false,
cmd/gobsidian/search.go:93:	cmd.Flags().IntVar(&flags.MaxResults, "max-results", 0, "teto de resultados por consulta")
cmd/gobsidian/serve.go:35:			flags.ReadOnlySet = cmd.Flags().Changed("read-only")
cmd/gobsidian/serve.go:36:			flags.DebounceMSSet = cmd.Flags().Changed("debounce-ms")
cmd/gobsidian/serve.go:37:			flags.MaxResultsSet = cmd.Flags().Changed("max-results")
cmd/gobsidian/serve.go:47:	cmd.Flags().StringVar(&flags.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
cmd/gobsidian/serve.go:48:	cmd.Flags().StringVar(&flags.LogLevel, "log-level", "", "debug, info, warn ou error")
cmd/gobsidian/serve.go:49:	cmd.Flags().BoolVar(&flags.ReadOnly, "read-only", false, "desabilita toda a superficie de escrita")
cmd/gobsidian/serve.go:50:	cmd.Flags().IntVar(&flags.DebounceMS, "debounce-ms", 0, "janela de coalescencia de eventos do watcher")
cmd/gobsidian/serve.go:51:	cmd.Flags().IntVar(&flags.MaxResults, "max-results", 0, "teto de resultados por consulta")
cmd/gobsidian/serve.go:52:	cmd.Flags().StringVar(&flags.CacheDir, "cache-dir", "", "diretorio do cache de indice")
cmd/gobsidian/serve.go:53:	cmd.Flags().BoolVar(&flags.FollowSymlinks, "follow-symlinks", false,
cmd/gobsidian/serve.go:55:	cmd.Flags().BoolVar(&flags.EagerSearch, "eager-search", false,
```

Diff aplicado (`README.md`, commit `8bb4769`):

```diff
-| `--read-only` | Removes the entire write surface. | `serve` |
-| `--cache-dir <path>` | Cache directory. Default: a hash of the vault path, always **outside** it. | `serve` |
-| `--debounce-ms <n>` | Watcher coalescing window. | `serve` |
-| `--log-level <level>` | `debug`, `info`, `warn` or `error`. | `serve` |
-| `--eager-search` | Loads the search index at boot. Default: lazy — most sessions read and write without ever searching. | `serve` |
-| `--max-results <n>` | Caps results per query. | `serve`, `search` |
+| `--read-only` | Removes the entire write surface. | `serve`, `daemon`, `doctor` |
+| `--cache-dir <path>` | Cache directory. Default: a hash of the vault path, always **outside** it. | `serve`, `daemon` |
+| `--debounce-ms <n>` | Watcher coalescing window. | `serve`, `daemon`, `doctor` |
+| `--log-level <level>` | `debug`, `info`, `warn` or `error`. | `serve`, `daemon` |
+| `--eager-search` | Loads the search index at boot. Default: lazy — most sessions read and write without ever searching. | `serve`, `daemon` |
+| `--max-results <n>` | Caps results per query. | `serve`, `search`, `daemon`, `doctor` |
```

daemon.go/doctor.go não foram tocados (fora de escopo por ruling).

```
$ pwsh -File scripts/check_readme_anchors.ps1
[i] 11 heading(s), 11 link(s) interno(s).
[OK] toda ancora resolve e toda secao H2 e alcancavel pela navegacao.

$ pwsh -File scripts/check_doc_refs.ps1
[...]
[OK] nenhum token entre crases parece citar artefato ausente do codigo.
```

Commit: `8bb4769` `docs(readme): the flag table names every subcommand that declares each flag`.

- 01:45 revisao recebida: item 1 (README) bloqueante, item 2 (wiki cache) nao-bloqueante
- 01:46 grep Flags() colado no relatorio; seis linhas da tabela corrigidas
- 01:48 check_readme_anchors e check_doc_refs verdes; commit 8bb4769
- 01:49 relatorio atualizado com Fix round 1
