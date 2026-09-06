# Final doc pass + final-review fixes — report

## Progresso

- 15:50 início; lido `task-final-brief.md` por completo.
- 15:52 lido `review-final.md` (`## Findings` + `## Stale references`) e `task-final-base.txt`; HEAD confere com a base (`5c613d5`).
- 15:56 B-F1 implementado: removida a flag `--debounce-ms` de `doctor` (`cmd/gobsidian/doctor.go:24,69`), `README.md:206` corrigido, teste `TestIndexEInspectNaoAceitamFlagsQueIgnoram` estendido com um caso para `doctor`. RED/GREEN abaixo. `verify.ps1` full gate verde. Commit `cd61624`.
- 16:03 B-F2 implementado: `jsonschema` de `vaultSearchInput.Tags` e `noteListInput.Tags` (`internal/mcpsrv/tools_read.go:327,363`) alinhado à prosa de `TOOLS.md`; `TagMode` sem descrição própria em `TOOLS.md`, deixado sem tag jsonschema (dito no achado B-F2 abaixo). Teste novo `TestSchemaServidoDescreveTagsHierarquicas` em `schema_params_test.go`, com RED/GREEN nos dois campos (abaixo). `verify.ps1` full gate verde. Commit `ada4897`.
- 16:05–16:24 Parte A (A1–A8) e os achados B-F3, B-N1, B-N5 e B-Stale implementados como edição de comentário/documentação — sem mudança de comportamento. Detalhe por item abaixo.
- 16:24 `go build ./...`, `go vet ./...`, `gofmt -l internal/ cmd/` e `go test -count=1 ./...` (suíte inteira) verdes depois de todos os comentários e docs.
- 16:24 `verify.ps1` full gate verde antes do commit de docs.
- 16:31 commit `b3ee480` (docs + comentários), 29 arquivos.
- 16:32 `scripts/audit_reports.ps1 -Task final` rodado; achados do próprio relatório corrigidos nesta revisão do arquivo (ver seção Auditoria).
- 16:37 relatório finalizado; tarefa concluída.

## Verificação de base

```
$ git log -1 --format=%H
5c613d5bb91b1b9a9d44ee793eb52158bf2a1a6e
```

Bate com `task-final-base.txt` (`5c613d5`).

---

## B-F1 — `doctor` não lê `--debounce-ms`

**Done.** `doctor` não tem watcher, então não há nada para coalescer;
`internal/doctor` nunca consultava `cfg.DebounceMS`. Removidas
`cmd/gobsidian/doctor.go:24` (`flags.DebounceMSSet = ...`) e `:69`
(`cmd.Flags().IntVar(&flags.DebounceMS, "debounce-ms", ...)`).
`README.md:206`: `doctor` removido da coluna Subcommands de `--debounce-ms`.
Verificado `docs/OPERACAO.md`/`docs/TOOLS.md`/`docs/ESTRUTURA.md`: nenhum
menciona `doctor` junto de `--debounce-ms` (grep vazio).

Teste estendido em `cmd/gobsidian/cli_subcommands_test.go`, dentro de
`TestIndexEInspectNaoAceitamFlagsQueIgnoram` (nome mantido — o teste já cobria
`search` além de index/inspect antes desta mudança, sem ser renomeado por
isso; decisão: manter, para não invalidar as referências históricas ao nome
em `review-176.md`, `task-156-report.md` etc.):

```go
	// doctor legitimamente mantem --read-only e --max-results (o proprio
	// diagnostico os le); --debounce-ms nao tem watcher para coalescer e
	// nao deve ser declarada (achado F1 da revisao final).
	if newDoctorCmd().Flags().Lookup("debounce-ms") != nil {
		t.Errorf("doctor declara --debounce-ms e nao a usa")
	}
```

**RED** (linha `:69` restaurada manualmente, teste rodado, depois removida de novo):

```
$ go test ./cmd/gobsidian/ -run TestIndexEInspectNaoAceitamFlagsQueIgnoram -v
=== RUN   TestIndexEInspectNaoAceitamFlagsQueIgnoram
    cli_subcommands_test.go:225: doctor declara --debounce-ms e nao a usa
--- FAIL: TestIndexEInspectNaoAceitamFlagsQueIgnoram (0.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	0.229s
FAIL
```

**GREEN** (fix restaurado):

```
$ go test ./cmd/gobsidian/ -run TestIndexEInspectNaoAceitamFlagsQueIgnoram -v
=== RUN   TestIndexEInspectNaoAceitamFlagsQueIgnoram
--- PASS: TestIndexEInspectNaoAceitamFlagsQueIgnoram (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	0.143s
```

Commit: `cd61624a49f5dfccc7134cff22e3e9ed8097fdef`.

---

## B-F2 — schema servido de `tags` não dizia "subtags"

**Done.** Task 180 (`18d9da4`) fez `vault_search.tags` e `note_list.tags`
casarem hierarquicamente (subtags) e dobrarem caixa/NFC; `TOOLS.md` foi
atualizado, a tag `jsonschema` das duas structs não. Como o servidor só
encaminha `type`+`description` ao host (`jsonschema-go v0.4.2`), o schema
servido continuava dizendo "todas as tags", sem mencionar subtags.

- `internal/mcpsrv/tools_read.go:327` (`vaultSearchInput.Tags`): tag
  `jsonschema` estendida com a prosa de `TOOLS.md:52` (bloco JSON de
  `vault_search`).
- `internal/mcpsrv/tools_read.go:363` (`noteListInput.Tags`, sem tag alguma
  antes): recebeu a prosa de `TOOLS.md:203` (bloco JSON de `note_list`).
- `noteListInput.TagMode` (`:364`): continua sem tag `jsonschema`.
  `TOOLS.md` não tem uma frase de descrição para `tag_mode` (só `enum` +
  `default`, que o servidor não encaminha de qualquer forma) — nada para
  copiar, então fica como está. Registrado, não corrigido.
- Teste novo `TestSchemaServidoDescreveTagsHierarquicas` em
  `internal/mcpsrv/schema_params_test.go`: abre uma sessão MCP real
  (`setupMCPServer`, já existente no arquivo), chama `ListTools`, navega
  `InputSchema.(map[string]any)["properties"]["tags"]["description"]` (é
  assim que o SDK entrega o schema no lado do cliente — documentado em
  `mcp.Tool.InputSchema`) e verifica a substring `"subtags"` para
  `vault_search` e `note_list`.

**RED 1/2** (`vaultSearchInput.Tags` com a tag antiga, sem subtags):

```
$ go test ./internal/mcpsrv/... -run TestSchemaServidoDescreveTagsHierarquicas -v
=== RUN   TestSchemaServidoDescreveTagsHierarquicas
    schema_params_test.go:317: vault_search.tags: description servida nao menciona subtags: "Notas que contenham TODAS as tags."
--- FAIL: TestSchemaServidoDescreveTagsHierarquicas (0.03s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv	0.997s
FAIL
```

**RED 2/2** (fix de `vault_search` restaurado; `noteListInput.Tags` sem tag jsonschema nenhuma):

```
$ go test ./internal/mcpsrv/... -run TestSchemaServidoDescreveTagsHierarquicas -v
=== RUN   TestSchemaServidoDescreveTagsHierarquicas
    schema_params_test.go:317: note_list.tags: description servida nao menciona subtags: ""
--- FAIL: TestSchemaServidoDescreveTagsHierarquicas (0.02s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv	0.971s
FAIL
```

**GREEN** (as duas tags restauradas):

```
$ go test ./internal/mcpsrv/... -run TestSchemaServidoDescreveTagsHierarquicas -v -count=1
=== RUN   TestSchemaServidoDescreveTagsHierarquicas
--- PASS: TestSchemaServidoDescreveTagsHierarquicas (0.03s)
PASS
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	0.984s
```

Suíte completa do pacote depois do fix, sem cache (`-count=1`):

```
$ go test ./internal/mcpsrv/... -count=1
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	2.323s
```

Commit: `ada4897d6845e4f6a76ae3a2c43cfa5be6486910`.

---

## A1 — `ARCHITECTURE.md` §5.5 (escrita atômica)

**Done.** Lido `internal/vault/atomic.go:140-222`. O texto antigo dizia
"backoff exponencial, três tentativas, 50 ms iniciais"; o código real é
`maxRetries := 10`, `retryDelay := 10 * time.Millisecond`, **fixo**, dormido
via `select` em `time.After`/`ctx.Done()`. Reescrevi o parágrafo com os
números reais, acrescentei o `Chmod(modo)` do temporário (`atomic.go:157`,
não-fatal, por que existe) e o `sincronizarDiretorio(dir)` depois do rename
(`atomic.go:205`, falha só logada em Debug — achado M12). Também corrigi
`:405-414` (pseudocódigo de `note_patch`): `writer.AtomicWrite(path, bytes)`
→ `vault.WriteAtomic(path, bytes)` — a implementação mora em
`internal/vault/atomic.go` (`ReplaceFile` + `WriteAtomic`), não em
`internal/writer` (confirmado: `internal/writer/` não tem `atomic.go`).

Não mexi em `writer.LockPath`/`writer.UnlockPath` no mesmo bloco de
pseudocódigo (também simplificados/desatualizados — hoje é
`s.locker.Lock(canonical)` devolvendo um `func()`): fora do escopo do achado,
que era especificamente o nome do pacote da escrita atômica.

---

## A2 — `ESTRUTURA.md` phantom/missing files (`internal`/`cmd`)

**Done.** Comparei a árvore documentada com `ls` de cada diretório de
`internal/` e `cmd/gobsidian/` (não-teste). Diff:

Removidos (não existem):
- `internal/vault/ignore.go`
- `internal/writer/writer.go`
- `internal/index/assets.go`

Adicionados (existem, faltavam no documento):
- `cmd/gobsidian/flags.go` (`flagsDeCofre`/`flagsDeCache`)
- `cmd/gobsidian/cli_log.go` (`loggerDeCLI`)
- `internal/ipc/desconexao.go` (`EhDesconexaoLimpa`)

Corrigido: `internal/config/flags.go` era listado como "mapeamento de flags do
cobra", mas esse arquivo não existe em `internal/config` — o mapeamento real
mora em `cmd/gobsidian/flags.go`. Reescrevi a entrada de `config.go` para
apontar para o lugar certo.

Observação fora do escopo do achado (que pedia só `ls -R internal cmd`, e não
mexi nisso): `scripts/` tem 11 scripts a mais do que o documento lista, e
`.github/workflows/` tem `release.yml` além dos dois listados. Não corrigido
— fica registrado para quem cuidar do restante da árvore documentada.

---

## A3 — `segregacao.md` seção "Troca atômica" e B-F3 (candidato 3 fechado)

**Done** (as duas dobram na mesma edição, como o brief antecipa).

- Seção 2 ("Troca atômica — três implementações") retitulada para "…
  (unificadas em `vault.ReplaceFile`, Task 171/172)"; abertura em passado
  ("Havia três", "A do `writer` **era** a cuidadosa… **tratava**"); o
  argumento do meio (regra × custo do cache vs. da nota) ficou como estava,
  como o brief pede.
- Seção 3 ("Chave derivada — a tag escapa da conta única"): acrescentado
  parágrafo "**Fechado pela Task 180** (`18d9da4`)" — confirmado por
  `grep -n func ChaveDeTag internal/index/chave.go` e
  `grep -n strings.ToLower internal/index/query.go` (linhas `:263,424,428`,
  exatamente as três que o achado cita, todas parâmetro de consulta).
- Fechamento do documento ("Se virar tarefa"): "o 3 porque falta a medição"
  virou "o 3 [está fechado] porque o que restou … já era classificado como
  certo".

---

## A4 — Comentários `.go` obsoletos (nomeando código movido/apagado)

**Done**, um a um, com o símbolo novo confirmado por `grep`/leitura direta do
arquivo antes de escrever (não usei LSP interativo aqui — os alvos eram
strings literais em comentário, `grep -n "^func"` bastou para confirmar cada
símbolo):

| Arquivo | Antes | Depois |
|---|---|---|
| `internal/daemon/daemon.go:40` | `via construirServico` | `via boot.Montar` |
| `internal/index/persist.go:65` | `invertedCacheState em cmd/gobsidian/serve.go` | `estadoDoCache em internal/boot/busca.go` |
| `internal/search/inverted.go:86,601,615` | `buildInvertedIndex` / `invertedCacheState` | `construirBusca` / `boot.estadoDoCache` (`internal/boot/busca.go`) |
| `internal/search/mmap.go:158-159` | idem | idem |
| `internal/search/persist_test.go:54,99` | `buildInvertedIndex, em cmd/gobsidian/serve.go` | `boot.construirBusca, em internal/boot/busca.go` |
| `internal/search/update_bench_test.go:16-17` | idem | idem |
| `internal/search/cloudonly_update_windows_test.go:88,146,171,175` | `buildInvertedIndex`/`invertedCacheState` | `boot.construirBusca`/`boot.estadoDoCache` |
| `internal/service/search_lazy.go:10` | `ver cmd/gobsidian/serve.go` | `boot.Montar, envolvendo boot.PrepararBusca, em internal/boot/montar.go` |
| `internal/service/search_lazy_test.go:25` | `prepararIndiceDeBusca` | `boot.PrepararBusca` |
| `internal/watcher/counters.go:6` | `adaptador em cmd/gobsidian/serve.go` | `adaptador watcherStats em internal/boot/montar.go` |

`internal/boot/doc.go:5-6`: a frase "index, search e inspect montam o índice
por conta própria" é falsa desde a Task 176 — confirmado
`grep -n "boot\." cmd/gobsidian/index.go cmd/gobsidian/inspect.go
cmd/gobsidian/search.go`: os três chamam `boot.AbrirIndice`
(`index.go:45`, `inspect.go:48`, `search.go:39`), e `search.go:45` chama
`boot.PrepararBusca`. Reescrito: "abrem o índice por aqui … mas não chegam a
Montar: não constroem watcher nem Service."

Mesma checagem em `CLAUDE.md`/`ESTRUTURA.md`/`ARCHITECTURE.md`
(`grep -n "por conta propria\|por conta própria\|montam o indice\|assemble on
their own"`): vazio nos três — a claim falsa só existia em `boot/doc.go`.
Estendi `CLAUDE.md` (linha do bloco `cmd/gobsidian/`) para mencionar
`boot.AbrirIndice`/`boot.PrepararBusca` para os três subcomandos de CLI, como
o brief permite ("se o estilo de uma linha permitir").

Todos os dez itens acima são comentário-apenas; nenhuma linha de código
executável mudou (conferido no diff antes do commit — colado na íntegra
abaixo, seção "Diff completo dos `.go`").

---

## A5 — `docs/wiki` (páginas derivadas)

Escolha por página, como o brief pede:

- **`docs/wiki/flows/boot.md`** — **fix in place + `status: stale`**. O
  drift aqui é sistêmico: o diagrama de fluxo inteiro (`construirServico`,
  `invertedCacheState`, `buildInvertedIndex`) usa os nomes de antes da Task
  174, não uma linha isolada. Corrigi os nomes (`boot.Montar`,
  `boot.estadoDoCache`, `boot.construirBusca`), `source_paths` (removido
  `cmd/gobsidian/servico.go`, que não existe mais; adicionados
  `internal/boot/montar.go` e `internal/boot/vigia.go`), e marquei
  `status: stale` com uma nota de topo explicando a troca de nomes — porque
  não re-derivei a página inteira contra o `boot` atual linha por linha (só
  os nomes citados pela revisão), e o resto do conteúdo (cache parcial,
  memória transitória, timings) não foi reconferido nesta passada.
- **`docs/wiki/Home.md:39`** — **fix in place, sem flip de status**. Uma
  única referência isolada (`cmd/gobsidian/servico.go` → `internal/boot/montar.go`
  na lista "primeiros arquivos para ler"); o resto da página não depende
  disso.
- **`docs/wiki/decisions/decisoes-fechadas.md:10`** — **fix in place, sem
  flip**. Só o `source_paths` citava o arquivo morto; o corpo da página não
  menciona `servico.go`/`construirServico` (grep vazio).
- **`docs/wiki/_wiki/manifest.json`** — removida a entrada `:29`
  (`cmd/gobsidian/servico.go`); `:27` (`serve.go`) mantida. JSON validado
  (`python -c "import json; json.load(...)"` → `[OK]`).
- **`docs/wiki/flows/encerramento.md`** — `source_paths` ganhou
  `internal/boot/espelho.go` e `internal/boot/vigia.go`; a seção "O espelho
  de stdin" (`:76-88`) agora diz que `mirrorReader` mora em
  `internal/boot/espelho.go` e que `boot.VigiarHost`
  (`internal/boot/vigia.go`) monta o andaime (pipe + mirrorReader +
  `lifecycle.New`) que serve e a ponte compartilham. `:91`
  (`shutdownExitCode`) mantido apontando para `cmd/gobsidian/serve.go` —
  confirmado ainda real (`grep -n "func shutdownExitCode" cmd/gobsidian/serve.go` → `:59`).

---

## B-Stale — comentários/testes/docs restantes

**Done**, além dos já cobertos em A4/A5:

- **Produção**: `internal/index/persist.go:65`, `internal/search/inverted.go:86,601,615`,
  `internal/search/mmap.go:158-159` — ver tabela A4.
- **Testes**: `internal/search/cloudonly_update_windows_test.go:88,146,171,175`,
  `internal/search/persist_test.go:54,99`, `internal/search/update_bench_test.go:16`,
  `internal/service/search_lazy_test.go:25` — ver tabela A4.
- **`docs/wiki/features/escrita.md:26,46`** — `writer.WriteAtomic` →
  `vault.WriteAtomic`; `CleanStaleTempFiles` (que nunca existiu sob esse
  nome em `vault` — a função sempre foi `SweepStaleTempFiles`, só o texto
  citava o nome errado) → `SweepStaleTempFiles`. `source_paths` corrigido
  (`internal/writer/atomic.go` → `internal/vault/atomic.go`).
- **`docs/wiki/overview/onde-ficam-os-dados.md:92,99`** — `writer.WriteAtomic`/
  `writer.SweepStaleTempFiles` → `vault.WriteAtomic`/`vault.SweepStaleTempFiles`;
  `source_paths` idem.
- **`docs/wiki/concepts/camadas-e-fronteiras.md:78`** — `service.Index`
  (interface de 12 métodos) foi removida na Task 173; `Service` guarda hoje
  um `*index.Index` concreto (confirmado `grep -n "index \*index.Index"
  internal/service/service.go:59`). Reescrevi o bullet: o ponto do
  parágrafo original (a interface não isolava nada) fica de pé, mais forte
  ainda, porque agora nem existe a interface nominal.
- **`docs/wiki/decisions/decisoes-fechadas.md:10`** — ver A5.
- **`docs/ESTADO.md:199`** — linha de cobertura por função (2026-09-02,
  histórica): mantida a medição, acrescentada uma frase "Depois da Task 174,
  `construirServico` é `boot.Montar`, `prepararIndiceDeBusca` é
  `boot.PrepararBusca` e `buildInvertedIndex` é `boot.construirBusca`;
  `carregarIndiceDoCache` continua com o mesmo nome, hoje em
  `internal/boot/indice.go`; `CleanStaleTempFiles` já não existe" — no
  mesmo estilo das linhas vizinhas (`:202-203` já tinham nota equivalente).
- **`docs/OPERACAO.md:884,981,1657,1679,1696`** — histórico e datado, fora
  do diff da revisão mas obsoleto por causa desta branch. Acrescentei
  `(hoje boot.PrepararBusca)` / `(hoje boot.construirBusca)` /
  `(hoje boot.estadoDoCache)` inline em cada um dos cinco pontos, sem
  reescrever a medição/narrativa em volta (mesmo padrão do precedente já
  existente em `OPERACAO.md:1525`: "`cmd/gobsidian/servico.go`, hoje
  `internal/boot/montar.go`").
- **`docs/SUGESTOES.md:311,317,413,437,775`** — achados de auditoria
  históricos e datados (M12, M13, P11, item de Performance, recomendação
  estrutural) citando `writer/atomic.go` e `writer.WriteAtomic`. Mesma
  técnica: anotação `(hoje internal/vault/atomic.go, Tasks 171/172)` /
  `(hoje já no mesmo pacote, Tasks 171/172)` inline, sem tocar no texto do
  achado em si.
- **`docs/SUGESTOES.md:924-925`** (M14) — `vault.NormalizeEOL` foi apagada
  na Task 166 (confirmado: `git log --oneline -S"func NormalizeEOL" --
  internal/vault/` aponta para `e46654d`, "delete dead code and the error
  code no path produces"; hoje só `vault.DetectEOL` existe). Acrescentei um
  comentário HTML (mesmo estilo dos `check-doc-refs: ignore` já usados no
  arquivo) dizendo que a recomendação, como escrita, não é mais executável.
  Não reescrevi a recomendação — o achado pede só a nota.

---

## A6 — `OPERACAO.md:2759` (flags que cada subcomando registra)

**Done.** O texto antigo ("As quatro flags que index, inspect, search,
serve, daemon e doctor registravam") deixava o leitor inferir que `doctor`
tinha `--cache-dir`/`--log-level`. Conferido `grep -n
"flagsDeCofre(cmd\|flagsDeCache(cmd" cmd/gobsidian/*.go`: `flagsDeCofre`
(`--vault`, `--follow-symlinks`) roda nos seis subcomandos; `flagsDeCache`
(`--cache-dir`, `--log-level`) roda em `index`, `inspect`, `search`, `serve`,
`daemon` — não em `doctor`. Reescrevi o parágrafo nomeando os dois grupos
separadamente.

---

## A7 — `TOOLS.md:340` (`vault_stats` retorno)

**Done.** Lido `service.StatsResult` (`internal/service/graph.go:714-735`)
por completo. O documento prometia "contagem de links, contagem de tags",
"notas vazias" e "notas somente-nuvem não hidratadas" — nenhum desses quatro
tem campo na struct. Campos reais: `Notes`, `Assets`, `TotalSize`,
`Collisions` (`alias_collisions`), `Generation`; com `include_health`:
`Orphans`, `BrokenLinks`, `BrokenAnchor`, `FrontmatterErrors` (todos
ponteiro — ausente quando não pedido, presente inclusive `0` quando pedido);
com `include_runtime`: `Runtime` (`num_goroutine`, `alloc`, `total_alloc`,
`sys`, `num_gc` — de `runtime.MemStats`, não RSS do processo, correção
também feita) e `Watcher`. Reescrevi a lista de retorno inteira a partir da
struct, e acrescentei uma frase explícita dizendo que link/tag/vazias/nuvem
NÃO têm campo hoje — para não deixar a omissão parecer descuido na próxima
leitura.

---

## A8 — `TOOLS.md` (Task 180, spelling original de tags + contagem da CLI)

**Done.** `note_list.tags` (`:216`) já dizia "grafia original… como
`note_metadata.tags`" antes desta edição — os dois já concordavam.
Estendi o parágrafo de `tag_list` (~`:308-311`) para citar `note_list.tags`
explicitamente (antes só citava `note_metadata.tags`), e acrescentei um
parágrafo novo sobre `cmd/gobsidian/index.go:53`
(`tags := len(idx.Tags("", 1))`): confirmado por leitura de
`internal/index/query.go:179-190` que `Tags` itera `ix.tags`, cujas chaves
já são `ChaveDeTag` (dobradas), então a contagem de tags distintas que
`index --json` reporta conta chaves dobradas — grafias que só diferem em
caixa/NFC/`#` contam uma vez. De quanto isso reduz a contagem num cofre real:
**não medido** — dito explicitamente no texto, como o achado pede.

---

## A9 / A10 — parked

Sem ação, como o brief instrui:
- A9: nomes de teste `TestInvertedCacheState`/`TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem`
  (hoje sob `internal/boot`) mantidos — são a rastreabilidade de provas de
  mutação de relatórios anteriores.
- A10: tabela do relatório da Task 178 faltando linhas — artefato de
  relatório antigo, sem ação.

---

## Divergências

Nenhuma. Concordei com todas as rulings do brief; onde a ruling dizia
"registre e não corrija" (B-N2, B-N3, B-N4, A9, A10), segui à risca.

---

## `verify.ps1` — tail das três corridas (full gate, sem `-SkipCross`/`-SkipNet`)

**Antes do commit 1** (`cd61624`, B-F1):

```
[...] 14. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

**Antes do commit 2** (`ada4897`, B-F2) — corrida completa colada, porque é a
mais representativa das 14 etapas:

```
Carregado em 587ms
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
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.00s)
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

Os seis pulados são os mesmos seis legítimos de plataforma/ambiente que
`review-final.md` já registrou (nenhum é o de paridade).

**Antes do commit 3** (`b3ee480`, docs + comentários) — idêntico, tail:

```
[...] 13. check_doc_refs
[OK] check_doc_refs
[...] 14. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

`go build ./...`, `go vet ./...`, `gofmt -l internal/ cmd/` e
`go test -count=1 ./...` (suíte inteira, sem cache) também rodaram limpos
separadamente antes desse gate — todos os 17 pacotes com teste em `ok`.

---

## Commits

1. `cd61624a49f5dfccc7134cff22e3e9ed8097fdef` — `fix(cli): doctor no longer declares --debounce-ms, which nothing read`
2. `ada4897d6845e4f6a76ae3a2c43cfa5be6486910` — `fix(mcpsrv): the served schema says what tags matches`
3. `b3ee48058bbdeb87dc9228b149fb336e38454a57` — `docs: final pass -- stale references after the boot/vault/tag moves`

---

## Auditoria (`scripts/audit_reports.ps1`)

Rodado duas vezes: uma contra a primeira versão deste relatório (897 bytes,
só o log de progresso), outra contra esta versão reescrita, com RED/GREEN e
o tail de `verify.ps1` colados.

**1ª corrida** (relatório ainda com 897 bytes, antes desta reescrita):

```
=== Relatorios (1) ===
  ...task-final-report.md:1: [SECAO-AUSENTE] sem secao de TDD/GREEN
  ...task-final-report.md:1: [SECAO-AUSENTE] sem secao de Verificacao
  ...task-final-report.md:1: [CURTO] 897 bytes — pequeno demais para conter saida de comando colada

[!] 17 achado(s).
```

**2ª corrida** (depois de reescrever o relatório com as seções RED/GREEN e a
saída de `verify.ps1` coladas, sem alterar nenhum fato relatado):

```
$ pwsh -File scripts/audit_reports.ps1 -Task final
=== Relatorios (1) ===

=== Ledger ===
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:9: [SHA-NAO-CONFERE] Task 4 descreve "review Approved; 1 Important doc-only fechado depois" mas 0534b5f..6a5c268 contem "docs(lifecycle): document watchSignals exit path and WaitGroup registration | feat(lifecycle): cancel root context on interrupt and SIGTERM"
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:11: [SHA-NAO-CONFERE] Task 6 descreve "review Approved; 2 Important plan-mandated fechados" mas 1a7786e..3453ddc contem "fix(lifecycle): strengthen test assertion and complete docstring | docs: tighten shutdown log assertion and document ctx exception | feat(lifecycle): shutdown sequence with per-step budget and hard limit"
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:284: [RELATORIO-AUSENTE] Task 94 marcada completa e sem task-94-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:287: [RELATORIO-AUSENTE] Task 95 marcada completa e sem task-95-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:290: [RELATORIO-AUSENTE] Task 96 marcada completa e sem task-96-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:315: [RELATORIO-AUSENTE] Task 103 marcada completa e sem task-103-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:318: [RELATORIO-AUSENTE] Task 102 marcada completa e sem task-102-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:321: [RELATORIO-AUSENTE] Task 101 marcada completa e sem task-101-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:324: [RELATORIO-AUSENTE] Task 100 marcada completa e sem task-100-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:327: [RELATORIO-AUSENTE] Task 99 marcada completa e sem task-99-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:330: [RELATORIO-AUSENTE] Task 98 marcada completa e sem task-98-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:333: [RELATORIO-AUSENTE] Task 97 marcada completa e sem task-97-report.md
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:988: [SHA-FANTASMA] deadbee nao existe no repositorio
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:5158: [SHA-NAO-CONFERE] Task 153 descreve "report task-153-report.md; review pending — batched with 155" mas 5360a3b..5360a3b contem "fix(doctor): list the listen lock too, deriving both lock suffixes and the log path from daemon [sem-doc]"

[!] 14 achado(s). Nenhum e automaticamente um defeito — cada um e
    uma frase ou um SHA que precisa de uma pessoa confirmando.
[i] Onde nao houve medicao, 'nao medido' e a resposta certa e nao e sinalizada.
```

Esta segunda corrida está limpa quanto a `task-final-report.md` (zero achados
contra este arquivo) e sobram exatamente os 14 achados de ledger antigo
(`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`, outro marco) que o
brief já cita como conhecidos.
