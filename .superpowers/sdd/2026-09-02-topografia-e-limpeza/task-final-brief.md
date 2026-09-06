# Final doc pass + final-review fixes — brief

One dispatch, one or more commits by category (`docs:` for docs/comments,
`test:` for tests, `fix:` only for behavior). Base: see `task-final-base.txt`.

## Part A — parked doc items (all verified against the tree on 2026-09-06)

Every fact below was checked by the orchestrator; do NOT re-verify by guessing —
if the code disagrees with the line, the code wins and you say so in the report.

A1. `docs/ARCHITECTURE.md:433` (§5.5) says «Backoff exponencial, três
    tentativas, 50 ms iniciais». Code (`internal/vault/atomic.go:186-222`):
    `maxRetries := 10`, `retryDelay := 10 * time.Millisecond`, **fixed** delay,
    sleep via `select` on `time.After`/`ctx.Done()`. Also missing from §5.5: the
    `Chmod(modo)` on the temp file (`atomic.go:157`, non-fatal) and the
    directory sync after the rename (`sincronizarDiretorio(dir)`, `atomic.go:205`,
    failure logged at Debug, not returned — achado M12). Rewrite the paragraph
    with those facts. Also check `:405-414` of the same section for the
    package name: the implementation lives in `internal/vault/atomic.go`
    (`ReplaceFile` + `WriteAtomic`), not `internal/writer`.

A2. `docs/ESTRUTURA.md:45` lists `vault/ignore.go` and `:116` lists
    `writer/writer.go`. Neither exists (`ls internal/vault internal/writer`).
    Delete the two lines. Then check the whole tree block against `ls -R
    internal cmd` and fix any other phantom or missing file — report the diff.

A3. `docs/segregacao.md:85-87`, section «### 2. Troca atômica — três
    implementações» still opens with `writer/atomic.go:141`,
    `index/persist.go:124`, `search/persist.go:87` in the present tense.
    Retitle to mark it historical (e.g. «— três implementações (unificadas em
    `vault.ReplaceFile`, Task 171/172)») and make the opening sentence past
    tense; keep the argument as history. Do NOT delete the section.

A4. Stale `.go` comments naming deleted/moved code:
    - `internal/daemon/daemon.go:40` — «via construirServico» → the function is
      `boot.Montar` (`internal/boot/montar.go`). Fix the sentence.
    - `internal/index/persist.go:65`, `internal/search/mmap.go:159`,
      `internal/search/persist_test.go:54`, `internal/search/update_bench_test.go:17`,
      `internal/service/search_lazy.go:10`, `internal/watcher/counters.go:6` —
      all cite `cmd/gobsidian/serve.go` as the home of `invertedCacheState` /
      `buildInvertedIndex` / the Service assembly. Find the real symbol with
      gopls (`invertedCacheState` and `buildInvertedIndex` now live under
      `internal/boot/` — confirm the file with grep/LSP) and point each comment
      there. Keep the comments' meaning; change only the location.
    - `internal/boot/doc.go:5-6` — «Os subcomandos de CLI nao passam por aqui:
      index, search e inspect montam o indice por conta propria» is FALSE
      since Task 176: `cmd/gobsidian/index.go:45`, `inspect.go:48`,
      `search.go:39` call `boot.AbrirIndice`, and `search.go:45` calls
      `boot.PrepararBusca`. Rewrite: they open the index through `AbrirIndice`
      (and `search` through `PrepararBusca`) but do not build watcher/Service.
    - Same claim anywhere in `CLAUDE.md` / `docs/ESTRUTURA.md` /
      `docs/ARCHITECTURE.md` («por conta própria», «still assemble on their
      own») — grep and fix. `CLAUDE.md:63` currently reads «a montagem que
      serve e daemon compartilham»; extend to mention `AbrirIndice` for the CLI
      subcommands only if the tree block's one-line style allows it.

A5. `docs/wiki` (derived docs, `status:` field is the mechanism):
    - `docs/wiki/flows/encerramento.md`: add `internal/boot/espelho.go` and
      `internal/boot/vigia.go` to `source_paths`; in the «O espelho de stdin»
      section (~:79-88) say the mirror is assembled by `boot.VigiarHost`;
      `:91` `shutdownExitCode` is still in `cmd/gobsidian/serve.go:59` — keep.
    - `docs/wiki/flows/boot.md:7-8,25,29`, `docs/wiki/Home.md:39`,
      `docs/wiki/decisions/decisoes-fechadas.md:10`: `cmd/gobsidian/servico.go`
      and `construirServico` no longer exist → `internal/boot/montar.go`,
      `boot.Montar`. Either fix the references in place AND set
      `status: stale` on pages you did not fully re-derive, or only flip
      `status: stale` with a one-line note — choose per page, state the choice
      in the report. `docs/wiki/_wiki/manifest.json:29` (`servico.go` hash):
      remove the entry; `:27` (`serve.go`) leave.

A6. `docs/OPERACAO.md:2756` «As quatro flags que `index`, `inspect`, `search`,
    `serve`, `daemon` e `doctor` registravam» — reword so a reader cannot
    infer `doctor` had `--cache-dir` (e.g. name which subcommands had which).

A7. `docs/TOOLS.md:340` `vault_stats` promises «contagem de links, contagem de
    tags». `service.StatsResult` (`internal/service/graph.go`, ~:714-735) has
    no such fields. Doc follows code: list the fields the struct really has
    (read the struct; do not paraphrase from memory).

A8. `docs/TOOLS.md` near `:308-311` (Task 180 paragraph on `note_metadata.tags`
    keeping original spelling): add that `note_list.tags` does the same
    (already said at `:216` — cross-check both say the same thing) and that the
    CLI `index` subcommand's distinct-tag count (`cmd/gobsidian/index.go:53`,
    `len(idx.Tags("", 1))`) counts folded keys, so spellings that differ only
    in case/NFC/`#` count once. Number NOT measured — say so if you mention a
    delta.

A9. Test names `TestInvertedCacheState` / `TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem`
    (now under `internal/boot/`) — PARKED, do NOT rename (they are the
    traceability of mutation proofs in earlier reports). Skip.

A10. Task 178 report table missing `note_read.max_bytes` / `note_list.offset`
     rows — report artifact only, NO action. Skip.

## Part B — findings from `review-final.md`

See the section appended below by the orchestrator. Each finding says what to
do; if you disagree, write it in the report under `## Divergências` and do it
anyway unless it would introduce a defect — in that case BLOCKED with the
reason.

## Process

- FOREGROUND only. No subagents. No `scripts/test_orphans.ps1`.
- Use gopls (LSP) for symbol locations; grep for prose.
- After code-comment edits: `gofmt -l ./...` must print nothing.
- Every `.md` touched: `python -c "open('<f>',encoding='utf-8').read()"`.
- `pwsh -File scripts/verify.ps1` (FULL gate) before each commit if any `.go`
  file changed; `-SkipCross -SkipNet` allowed for docs-only commits. Paste the
  tail of the output in the report.
- `pwsh -File scripts/check_doc_refs.ps1` if it exists standalone — otherwise
  verify.ps1 covers it.
- Commit by explicit path (`git add <file>`), message written to
  `.superpowers/sdd/2026-09-02-topografia-e-limpeza/commit-final-<n>.txt`,
  `git commit -F <that file>`. Conventional Commits, English. Trailers:
  `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>` and
  `Claude-Session: https://claude.ai/code/session_01P5wkw6PAdBFzF3uB1w1jNj`.
  Never `git add -A`/`.`; never checkout/restore/stash/clean/reset; never
  `go mod tidy`.
- Report: `.superpowers/sdd/2026-09-02-topografia-e-limpeza/task-final-report.md`
  with `## Progresso` (one line per step, real `date +%H:%M`), one subsection
  per item A1…A8 + each B finding with «done / skipped (why) / BLOCKED», the
  verify tail, the commit SHAs (only after they exist), and
  `scripts/audit_reports.ps1` output (14 old-ledger findings are known).

---

## Part B — findings from `review-final.md` (read that file's `## Findings` and `## Stale references` in full; rulings below)

B-F1 (`fix(cli)`, own commit). `cmd/gobsidian/doctor.go:69` declares
`--debounce-ms`; nothing in `internal/doctor` reads `cfg.DebounceMS` (verified:
`grep -rn DebounceMS internal/doctor` is empty; `ReadOnly` and `MaxResults` ARE
read — `checks.go:126`, `daemon.go:110` — keep those). Do: delete `:69` and
`:24` (`flags.DebounceMSSet = …`); `README.md:206` — remove `doctor` from the
Subcommands column of `--debounce-ms`; in
`cmd/gobsidian/cli_subcommands_test.go:205` (`TestIndexEInspectNaoAceitamFlagsQueIgnoram`)
add a case so `doctor` is checked for `debounce-ms` only (it legitimately keeps
`read-only` and `max-results`) — rename the test if the name becomes false, and
note the rename in the report. Mutation proof: restore line `:69`, run the test,
paste the failure, remove again. Check `docs/OPERACAO.md` / `docs/TOOLS.md` /
`docs/ESTRUTURA.md` for any sentence listing `doctor` among `--debounce-ms`
users and fix it. Ruling: the orphans gate is NOT required — `doctor` is not on
the serve/daemon shutdown path; say "not run, ruling" in the report.

B-F2 (`fix(mcpsrv)`, own commit). `internal/mcpsrv/tools_read.go:327`
(`vaultSearchInput.Tags`) and `:363` (`noteListInput.Tags`, no `jsonschema` tag
at all): put the exact `description` prose shown in `docs/TOOLS.md` (~`:818`
for vault_search, ~`:911` for note_list) into the `jsonschema` tags. Then check
whether an existing test renders the served schema against TOOLS.md
(`internal/mcpsrv/schema_params_test.go`, `tags_contrato_test.go`,
`metadata_include_schema_test.go`) — if one compares descriptions, extend it;
if none does, add ONE small test in `internal/mcpsrv` asserting the served
schema's `description` for `vault_search.tags` and `note_list.tags` contains
the substring `subtags` (the hierarchical claim) — with mutation proof (blank
the tag, run, paste, restore). Also `noteListInput.TagMode` (`:364`) has no
`jsonschema` — give it the TOOLS.md description for `tag_mode` if TOOLS.md has
one; if not, leave and say so.

B-F3 (`docs:`). `docs/segregacao.md:101-112` candidate 3 + `:134`: add the
note that Task 180 (`18d9da4`) closed it with `index.ChaveDeTag`; the three
remaining `strings.ToLower` in `query.go` (`:263`, `:424`, `:428`) are query
parameters, which the document already classifies as fine. Folds with A3.

B-N1 (`docs:` — comment only). `internal/daemon/lock.go:121`
`EhArquivoDeTrava` doc comment says "uma das duas travas"; body is
`HasSuffix(nome, ".lock")`. Ruling: comment follows code — say "qualquer
`.lock` no diretório de runtime (que é só do produto)". Do not change the
predicate.

B-N2: PARKED (no observable difference; ruling ledgered). Skip.
B-N3: PARKED (wall-clock assertion, ~4x margin, noted for the owner). Skip.
B-N4: PARKED (`var cacheMagic` pinned by test; keep). Skip.

B-N5 (`docs:` — comment only). `internal/boot/montar.go:85-87`: add one
sentence: the claim holds because the default `CacheDir` is outside the vault;
`--cache-dir` inside the vault would put `AbrirIndice`'s temp file within the
sweep's reach. No code change.

B-Stale (`docs:`; comment-only `.go` edits and `_test.go` comment edits may go
in the same docs commit — no logic changes). Every row of the review's
`## Stale references` tables — production comments (`internal/index/persist.go:65`,
`internal/search/inverted.go:86,601,615`, `internal/search/mmap.go:158-159`),
test comments (`internal/search/cloudonly_update_windows_test.go:88,146,171,175`,
`internal/search/persist_test.go:54,99`, `internal/search/update_bench_test.go:16`,
`internal/service/search_lazy_test.go:25`), and docs (`docs/SUGESTOES.md:311,317,413,437,775,924-925`,
`docs/ESTADO.md:199`, `docs/OPERACAO.md:884,981,1657,1679,1696`,
`docs/wiki/flows/boot.md:8,25,29,83`, `docs/wiki/features/escrita.md:26,46`,
`docs/wiki/overview/onde-ficam-os-dados.md:92,99`,
`docs/wiki/concepts/camadas-e-fronteiras.md:78`,
`docs/wiki/decisions/decisoes-fechadas.md:10`). Confirm each new symbol with
gopls/grep before writing it (`boot.estadoDoCache`, `boot.construirBusca`,
`boot.PrepararBusca`, `vault.WriteAtomic`, `vault.SweepStaleTempFiles`). For
`docs/ESTADO.md:199` and `docs/OPERACAO.md` (historical, dated): add the same
kind of "hoje `internal/boot/...`" note the neighbouring lines already carry —
do not rewrite the measurement. For `docs/SUGESTOES.md:924-925` (M14 names a
deleted `vault.NormalizeEOL`): add a note that the function was removed in
Task 166 and the recommendation needs re-stating, no more. Wiki pages: same
rule as A5 (fix + keep `status`, or flip `status: stale` — state the choice).

## Commits (order)

1. `fix(cli): doctor no longer declares --debounce-ms, which nothing read`
2. `fix(mcpsrv): the served schema says what tags matches` (+ test)
3. `docs: final pass — stale references after the boot/vault/tag moves`
   (everything else: A1–A8, B-F3, B-N1, B-N5, B-Stale)

Full `verify.ps1` before commits 1 and 2 (code). Commit 3 has `.go`
comment-only edits: run the full gate once more before it (gofmt/vet/lint see
comments too).
