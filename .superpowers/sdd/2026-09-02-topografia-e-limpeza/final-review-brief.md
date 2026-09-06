# Final whole-branch review — plan 2026-09-02-topografia-e-limpeza

Range: `6c5d1f1..5c613d5` (98 commits, 185 files, +12 653/−2 501). Plan:
`docs/superpowers/plans/2026-09-02-topografia-e-limpeza.md` (Tasks 146–181;
read only `## Global Constraints`, lines 53–74, and the task titles).

## Inputs (all under `.superpowers/sdd/2026-09-02-topografia-e-limpeza/`)

| File | What |
|---|---|
| `final-log-stat.txt` | commit list + `--stat`. Read first. |
| `final-diff-prod.diff` | production code: `internal/`, `cmd/`, `tools/`, go.mod/go.sum, no `_test.go`. 5 858 lines. Read in full. |
| `final-diff-tests.diff` | `_test.go` files. 9 069 lines. Read in full — a test that cannot fail is a defect here. |
| `final-diff-docs.diff` | `docs/`, `CLAUDE.md`, `README.md`, `scripts/`, `.claude/` (excluding OPERACAO.md, wiki). 6 651 lines. Read selectively: CLAUDE.md, ESTRUTURA.md, ARCHITECTURE.md, TOOLS.md, ESTADO.md hunks. |
| `final-diff-excluded-stat.txt` | what was left out (OPERACAO.md, wiki, testdata) — only stat. |
| `final-parked-items.txt` | ledger lines already parked for the final doc pass. Known — do not re-report. |
| `progress.md` | the ledger. Every `Ruling:` line is a decision already taken — do not re-litigate; you may flag a ruling as wrong only with a concrete failure it causes. |

Per-task reviews (`review-*.md`) already ran on each task. Your job is what
a task-scoped review cannot see: cross-task interactions, contracts that
drifted between tasks, duplicated accounts, the import graph, docs that
describe code that no longer exists, tests that assert nothing.

## Standing rules (copy of the binding constraints — the plan wording wins)

- Never `git checkout`, `git restore`, `git stash`, `git clean`, `git reset`. Uncommitted owner work sits in the tree (`test-vault/`, `.claude/skills/`, `Resume-Claude.ps1`). You do NOT commit anything.
- Never `go mod tidy`.
- stdout belongs to JSON-RPC; slog to stderr. `doctor`/`version`/`search`/`index`/`inspect` print to stdout on purpose, with a comment.
- No `net/*` under `internal/` or `cmd/` except `net.Dial`/`net.Listen` with literal `"unix"` in `internal/ipc`.
- No SDK MCP types outside `internal/mcpsrv`.
- Import graph in `CLAUDE.md` must match `go list -f '{{.Imports}}' ./internal/...` for production imports (test-only edges are listed separately there). New edge needs written justification; leaves gain no import.
- Uma conta por regra: every derived key/path/format decision through one function.
- Platform code behind build tags in separate files; never `if runtime.GOOS ==`.
- Console ASCII only: `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`.
- No `helpers.go`, `utils.go`, `common.go`.
- Conventional Commits, English, one category per commit.
- No unmeasured number; "não medido" is the honest value. Mutation proof = pasted output, past tense.
- A test that cannot fail is worse than no test.
- Schema that promises and code that ignores is a defect; a field with a fixed value lies.
- Do NOT run `scripts/test_orphans.ps1`. Run everything in the FOREGROUND (no run_in_background, no Monitor). Do not spawn subagents.

## Known / already-parked (do not re-report)

The seven lines in `final-parked-items.txt`, plus: B19 (heading_level unenforced), B20 (parser NFD inline-tag truncation, `ext_tag.go:36`), B21 (dead `default` in `leitor.value`), `parser.dedupeTags` second "same tag" account, TOOLS.md:331 `vault_stats` counts, 171 N1/N3, 172 N6, 174 stale comments (incl. `internal/daemon/daemon.go:40`, wiki pages, test names TestInvertedCacheState / TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem), 176 N1/N2, 177 N2 (wiki encerramento.md source_paths), 178 N2, 180 N7/N8, "CLI subcommands still assemble on their own" sentences post-176. These are scheduled for one fix dispatch after your review — if you find MORE of the same class (stale docs naming pre-plan files/functions), list them under a single "stale references" finding with file:line so that dispatch can fold them.

## Verify with your own commands (paste output in the review)

```
go build ./...
go vet ./...
go test -race -count=1 ./...
go list -f '{{.ImportPath}} {{.Imports}}' ./internal/... ./cmd/...   # compare against the CLAUDE.md graph
pwsh -File scripts/verify.ps1 -SkipCross -SkipNet
```

If `verify.ps1` fails, that is a finding by itself, with the output.

Rubric: `docs/papeis/revisor.md` (read in full, ~120 lines) and
`docs/ARMADILHAS.md` (skim headings). Findings follow the contract in
`revisor.md` §"O contrato do achado": file:line, mechanism, concrete failure,
what fixes it. No praise. No restating the diff.

## Output

Write `.superpowers/sdd/2026-09-02-topografia-e-limpeza/review-final.md`:

1. `## Verification` — commands run, exit codes, pasted tails.
2. `## Import graph` — the `go list` result vs CLAUDE.md, line by line if they differ.
3. `## Findings` — grouped `### Critical` / `### Important` / `### Nits`, each `**F<n>** file:line — …`. Critical = wrong behavior, data loss, broken contract, race, security. Important = defect a user or the next session pays for. Nit = everything else.
4. `## Stale references` — one list, file:line → what it should say.
5. `## Verdict` — one of `APPROVED`, `APPROVED_WITH_NITS`, `CHANGES_REQUIRED`, with one sentence why.

Keep a `## Progresso` section at the TOP of the file from the first minute:
one line per step (`HH:MM — read final-log-stat`, `HH:MM — prod diff 1/3`,
`HH:MM — go test -race started`…), timestamps from real `date +%H:%M`. Update
it as you go — if you die, the next reader must know where you stopped.
Never idle silently; if something blocks you, write `BLOCKED: <exact error>`
in the file and return.

Return to the orchestrator ONLY: verdict, finding counts per severity, and
the path of the review file. Validate the .md as UTF-8:
`python -c "open('<path>',encoding='utf-8').read()"`.
