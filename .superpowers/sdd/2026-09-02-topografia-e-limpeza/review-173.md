# Review: Task 173 (drop service.Index)

Commit c1ef235 on base c28bc86.

## Spec compliance

**PASS.** Matches the brief step by step: `service.Index` interface deleted,
`Service.index`/`New` retyped to `*index.Index`, the type assertion in
`search.go` removed and `s.index` passed directly to `search.CalculateBM25`
and `search.GenerateSnippet`. Only the 4 named paths touched
(`internal/service/service.go`, `internal/service/search.go`,
`internal/service/frontmatter_err_test.go`, `docs/ESTADO.md`). The
orchestrator's ruling on `frontmatter_err_test.go:55` (convenience assertion,
not a fake — `newTestService` builds a real index, no `type fakeIndex`) was
applied exactly as ruled: `ix, ok := svc.index.(*index.Index)` + `t.Fatal`
replaced by `ix := svc.index`, unused `internal/index` import dropped.
`docs/ESTADO.md:157` keeps the measured `gopls references` numbers (correctly
labeled "medido antes da Task 173") and appends the Task 173 sentence — a
better-than-literal reading of the brief's "trocar a frase" that avoids
discarding a measured fact.

## Code quality

**PASS.** Purely structural, no behavior change. `Service` doc comment
matches the brief's Step 2 wording, explains why the interface is gone
(one implementation, no fake, both search helpers require the concrete
type anyway) — no deliberation left in comments.

## Findings

**N1 (should-fix)** — `task-173-report.md` Step 5 / brief contract says
"Colar" (paste) the benchstat output; the report only gives a prose summary
("`~` em sec/op, B/op e allocs/op... geomean sec/op +1.40%") with no pasted
table. Raw benchmark output does exist on disk
(`%LOCALAPPDATA%\gobsidian-bench\2026-09-02\{antes,depois}173_service.txt`)
and I re-ran benchstat against it myself — the claim is accurate (all 5
benchmarks `~`, p ranging 0.21–0.81, geomean sec/op +1.40% noise, B/op and
allocs/op geomean ~0%). Not a fabrication, but CLAUDE.md's own rule
("O relatório é o entregável, não o resumo dele") is exactly what this
report violated in this one section — the report should have carried the
benchstat table itself, not a description of it.

No blocking findings.

## Verified myself

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test -race -count=1 ./internal/service/ ./internal/mcpsrv/` — both `ok` (service 40.2s, mcpsrv 8.9s).
- `grep -rn 'service\.Index\b\|idxImpl' --include=*.go .` — empty, matches report.
- `GOOS=windows go list -f '{{.Imports}}' ./internal/service/` — no new edge; `index` was already imported.
- Task 158 nil-index guard: `s.index == nil` checks live in `internal/service/graph.go:99,334,505,606,714`; `TestServicoSemIndiceDevolveVaultUnavailable` (`errors_test.go:30`) calls `New(v, nil, nil, nil, Options{})` with a literal `nil` for `*index.Index` — a plain typed-nil pointer now, simpler than the old interface-nil hazard the task this guard exists for was about. Test still present and asserts `CodeVaultUnavailable` across LinkGraph/TagList/ListNotes/Metadata.
- `search.CalculateBM25(queryTokens, s.inverted, idxImpl)` / `search.GenerateSnippet(..., idxImpl, ...)`: both functions were already declared with `idx *index.Index` (`internal/search/bm25.go:55`, `internal/search/snippet.go:82`), so `idxImpl` was always either the correctly-typed `s.index` or `nil` from a failed assertion that (per Step 1's grep, no fake) could never actually fail — semantics identical, dead branch removed.
- Independently re-ran `benchstat` on the two raw `.txt` files under
  `gobsidian-bench/2026-09-02/` — confirms report's `~` claim on all 5
  benchmarks, sec/op geomean +1.40% (noise), B/op and allocs/op geomean flat.
- `verify173.txt` (in the same bench dir) ends `[OK] Bateria completa. Pode
  commitar.`, mtime 09:01, consistent with report's claimed Step 7 window
  (08:57-09:01) and preceding the 09:02:27 commit — timing is plausible, not
  implausible.
- `git status --short`: no leftover from this task; only the owner's
  pre-existing dirty state (test-vault/, .claude/skills/, Resume-Claude.ps1,
  other plans' ledger files) is untracked.
