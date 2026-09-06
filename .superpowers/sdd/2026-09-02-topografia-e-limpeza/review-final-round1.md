# Re-review — final fix round (5c613d5..b3ee480)

## Progresso

- 16:36 início; lidos `review-final.md`, `task-final-brief.md`, `task-final-report.md`.
- 16:37 lido o diff completo `review-5c613d5..b3ee480.diff` (1 839 linhas, 34 arquivos) — inteiro, três partes.
- 16:37 `go build ./...` e `go vet ./...`: exit 0.
- 16:38 `go test -race -count=1 ./cmd/... ./internal/mcpsrv/...`: exit 0.
- 16:38 F1: grep de `debounce` em `doctor.go`/README/OPERACAO/TOOLS/ESTRUTURA — nenhuma menção de `doctor` junto de `--debounce-ms`.
- 16:39 F1: mutação — restaurada a flag em `doctor.go`, teste RED confirmado, restaurado exatamente, `git diff` limpo.
- 16:39 F2: teste novo passa; mutação — `noteListInput.Tags` sem tag `jsonschema`, RED confirmado, restaurado exatamente, `git diff` limpo.
- 16:39 Stale references: `grep -rn "func "` para os 8 símbolos citados — todos existem no arquivo citado.
- 16:39 A1: leitura de `internal/vault/atomic.go:128-222` linha a linha contra o parágrafo reescrito de ARCHITECTURE.md §5.5.
- 16:40 A2: `ls` de cada diretório não-teste de `internal/` e `cmd/gobsidian`, comparado item a item com a árvore de ESTRUTURA.md.
- 16:40 Wiki: `head` das sete páginas tocadas — `status:` confere com o que o relatório diz ter escolhido.
- 16:40 Commit hygiene: `git show --stat` nos três commits — uma categoria cada, nenhum arquivo de `.superpowers`/`test-vault`/`.claude/skills`.
- 16:40 `go test -count=1 ./...` (suíte inteira, sem race): 17 pacotes `ok`, 2 sem teste.
- 16:40 A7/A8: leitura direta de `StatsResult`/`RuntimeStats` (`internal/service/graph.go:695-735`) e de `docs/TOOLS.md:204` (tag_mode sem description) contra o texto reescrito.
- 16:40 revisão fechada.

---

## Verification

```
$ go build ./...
$ go vet ./...
(ambos exit 0, saída vazia)

$ go test -race -count=1 ./cmd/... ./internal/mcpsrv/...
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	4.999s
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	6.201s

$ go test -count=1 ./...   (suíte inteira, sem -race, para tempo)
ok  	.../cmd/gobsidian	2.362s
ok  	.../internal/boot	2.081s
ok  	.../internal/config	0.718s
ok  	.../internal/console	0.581s
ok  	.../internal/daemon	6.899s
ok  	.../internal/doctor	31.186s
ok  	.../internal/index	17.006s
ok  	.../internal/ipc	0.767s
ok  	.../internal/lifecycle	1.986s
ok  	.../internal/mcpsrv	11.814s
ok  	.../internal/parser	0.739s
ok  	.../internal/search	20.788s
ok  	.../internal/service	39.584s
?   	.../internal/text	[no test files]
ok  	.../internal/vault	28.816s
ok  	.../internal/vaulttest	0.608s
ok  	.../internal/watcher	19.183s
ok  	.../internal/writer	3.699s
ok  	.../tools/netcheck	12.980s
?   	.../tools/netcheck/cmd/netcheck	[no test files]
```

### F1 — mutação reproduzida

RED (linha `cmd.Flags().IntVar(&flags.DebounceMS, "debounce-ms", ...)` restaurada em `cmd/gobsidian/doctor.go:69`):

```
$ go test ./cmd/gobsidian/ -run TestIndexEInspectNaoAceitamFlagsQueIgnoram -v
=== RUN   TestIndexEInspectNaoAceitamFlagsQueIgnoram
    cli_subcommands_test.go:225: doctor declara --debounce-ms e nao a usa
--- FAIL: TestIndexEInspectNaoAceitamFlagsQueIgnoram (0.18s)
FAIL
```

GREEN (linha removida de novo):

```
--- PASS: TestIndexEInspectNaoAceitamFlagsQueIgnoram (0.00s)
```

`git diff --stat cmd/gobsidian/doctor.go` depois de restaurar: vazio.

### F2 — mutação reproduzida

RED (`noteListInput.Tags` sem tag `jsonschema`, em `internal/mcpsrv/tools_read.go:363`):

```
$ go test ./internal/mcpsrv/... -run TestSchemaServidoDescreveTagsHierarquicas -v -count=1
    schema_params_test.go:317: note_list.tags: description servida nao menciona subtags: ""
--- FAIL: TestSchemaServidoDescreveTagsHierarquicas (0.03s)
FAIL
```

GREEN (tag restaurada):

```
--- PASS: TestSchemaServidoDescreveTagsHierarquicas (0.03s)
```

`git diff --stat internal/mcpsrv/tools_read.go` depois de restaurar: vazio.

### Stale references — símbolos citados, confirmados

```
internal/boot/busca.go:50   func estadoDoCache(...)
internal/boot/busca.go:169  func construirBusca(...)
internal/boot/busca.go:108  func PrepararBusca(...)
internal/boot/montar.go:65  func Montar(...)
internal/vault/atomic.go:226 func WriteAtomic(...)
internal/vault/atomic.go:51  func SweepStaleTempFiles(...)
internal/vault/atomic.go:128 func ReplaceFile(...)
internal/vault/syncdir_unix.go:15 / syncdir_windows.go:17  func sincronizarDiretorio(...)
```

Todos os oito símbolos que a documentação passou a citar existem exatamente no arquivo apontado.

### A1 — atomic.go vs ARCHITECTURE.md §5.5, linha a linha

Lido `internal/vault/atomic.go:128-222` por completo. Confere item a item com o parágrafo reescrito:
`maxRetries := 10` (não 3), `retryDelay := 10 * time.Millisecond` **fixo** (não backoff exponencial,
não 50 ms), dormido via `select` em `time.After`/`ctx.Done()` (:200-204); `tmpFile.Chmod(modo)` com
`modo` de `os.Stat(targetPath)` ou `0644` se o alvo não existe, falha só logada em `slog.Debug`, não
fatal (:143-158); `sincronizarDiretorio(dir)` chamado **depois** do rename bem-sucedido, falha também
só em `Debug` (:197-206). Nenhuma divergência.

### A2 — árvore de ESTRUTURA.md vs `ls` real

`ls` de todos os 16 diretórios não-teste de `internal/` e de `cmd/gobsidian`, comparado arquivo a
arquivo com o bloco da árvore: **zero divergência** — nem arquivo fantasma sobrando, nem arquivo real
faltando, em nenhum dos 16 diretórios (contagens batem: cmd/gobsidian 10, config 2, lifecycle 7,
vault 13, vaulttest 6, parser 11, index 14, search 11, watcher 7, writer 5, service 8, mcpsrv 8,
text 1, console 4, boot 6, ipc 4, daemon 9, doctor 5).

### Wiki — status consistente com a escolha relatada

| Página | `status:` atual | Relatado |
|---|---|---|
| `flows/boot.md` | `stale` | fix in place + stale — confere |
| `Home.md` | `active` | fix in place, sem flip — confere |
| `decisions/decisoes-fechadas.md` | `active` | fix in place, sem flip — confere |
| `features/escrita.md` | `active` | fix in place, sem flip — confere |
| `overview/onde-ficam-os-dados.md` | `active` | fix in place, sem flip — confere |
| `concepts/camadas-e-fronteiras.md` | `active` | fix in place, sem flip — confere |
| `flows/encerramento.md` | `active` | source_paths só ganhou entradas — confere |

### Commit hygiene

```
cd61624 fix(cli): doctor no longer declares --debounce-ms, which nothing read
  README.md, cmd/gobsidian/cli_subcommands_test.go, cmd/gobsidian/doctor.go
ada4897 fix(mcpsrv): the served schema says what tags matches
  internal/mcpsrv/schema_params_test.go, internal/mcpsrv/tools_read.go
b3ee480 docs: final pass -- stale references after the boot/vault/tag moves
  29 arquivos, todos docs/ ou comentário .go — nenhuma linha executável
```

Nenhum commit toca `.superpowers/`, `test-vault/`, `.claude/skills/` ou `Resume-Claude.ps1`. Uma
categoria por commit, exatamente como o brief ordenou.

### A7/A8 — TOOLS.md vs struct real

`service.StatsResult` (`internal/service/graph.go:714-735`): `Notes`, `Assets`, `TotalSize`,
`Collisions`(`alias_collisions`), `Generation`, ponteiros `Orphans`/`BrokenLinks`/`BrokenAnchor`/
`FrontmatterErrors`, `Runtime`(`*RuntimeStats`), `Watcher`. `RuntimeStats` (:695-701):
`NumGoroutine`/`Alloc`/`TotalAlloc`/`Sys`/`NumGC` — bate exatamente com a lista reescrita em
`docs/TOOLS.md`, campo a campo. `docs/TOOLS.md:204` confirma que o bloco JSON de `tag_mode` só tem
`enum`+`default`, sem `description` — não havia prosa para copiar, como o relatório diz.

---

## Tabela — item por achado

| Item | Endereçado? | Evidência |
|---|---|---|
| F1 (`doctor --debounce-ms`) | Sim | flag removida, README corrigido, teste estendido, RED/GREEN reproduzidos por mim |
| F2 (schema servido de `tags`) | Sim | duas tags `jsonschema` alinhadas a TOOLS.md, teste novo, RED/GREEN reproduzidos por mim; `TagMode` deixado sem tag por falta de prosa em TOOLS.md (correto) |
| F3 (`segregacao.md` candidato 3) | Sim | nota "Fechado pela Task 180" adicionada, três `strings.ToLower` restantes confirmados como parâmetro de consulta |
| N1 (`EhArquivoDeTrava` doc comment) | Sim (comentário) | comentário agora diz "qualquer `.lock`"; predicado intocado, como a ruling pediu |
| N2 | Parked (ruling) | sem ação, como instruído |
| N3 | Parked (ruling) | sem ação, como instruído |
| N4 | Parked (ruling) | sem ação, como instruído |
| N5 (`montar.go` comentário) | Sim | frase sobre `--cache-dir` dentro do cofre adicionada, sem mudança de código |
| A1 (ARCHITECTURE.md §5.5) | Sim | confere linha a linha com `atomic.go` |
| A2 (ESTRUTURA.md árvore) | Sim | zero divergência contra `ls` real em 16 diretórios |
| A3 (segregacao.md seção 2) | Sim | retitulada, passado, "unificadas em vault.ReplaceFile" |
| A4 (comentários `.go` obsoletos) | Sim | 10 pontos corrigidos, símbolos confirmados por grep, nenhuma linha executável tocada |
| A5 (wiki) | Sim | escolha por página documentada e consistente com o `status:` real |
| A6 (OPERACAO.md flags por subcomando) | Sim | reescrito nomeando os dois grupos (`flagsDeCofre`/`flagsDeCache`) separadamente |
| A7 (TOOLS.md vault_stats) | Sim | lista reescrita bate campo a campo com `StatsResult`/`RuntimeStats` |
| A8 (TOOLS.md tags/contagem CLI) | Sim | `note_list.tags` citado, contagem de `index --json` explicada, "não medido" dito explicitamente |
| A9/A10 | Parked (ruling) | sem ação, como instruído |
| Stale references (produção/teste/docs) | Sim | todos os 8 símbolos novos confirmados existentes no arquivo citado |
| Commit hygiene | Sim | 3 commits, 1 categoria cada, sem arquivo de owner/ledger dentro |

---

## Findings

Nenhum. Não encontrei defeito novo introduzido por esta leva — todas as mudanças são comentário/doc
ou comportamento estritamente reduzido (remoção de flag morta, extensão de tag de schema), com prova
de mutação reproduzida por mim de forma independente para as duas mudanças de comportamento (F1, F2).
A verificação do grafo de arquivos (A2) e dos 8 símbolos citados nas referências obsoletas não achou
nenhuma divergência residual.

---

## Verdict

**APPROVED**
