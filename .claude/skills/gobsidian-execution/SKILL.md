---
name: gobsidian-execution
description: Resume and continue executing the gobsidian implementation plan task by task, with the ledger, task briefs, review packages, and milestone gates this project uses. Use whenever asked to continue the plan, work on the next task, pick up where the last session stopped, check what is done, or execute any numbered task from docs/superpowers/plans/. Also use before dispatching an implementer or reviewer for this repository, since the per-task loop and the accumulated decisions here are what keep reviews from re-litigating settled ground. Read the ledger before assuming anything about progress — context does not survive between sessions but the ledger does.
---

# Executing the gobsidian plan

The plan lives in `docs/superpowers/plans/2026-07-25-gobsidian-v01.md` and is the source implementations are transcribed from. Tasks are numbered; M0 is Tasks 1–11, M1 is Tasks 12–26.

## Before anything: read the ledger

```bash
cat .superpowers/sdd/progress.md
```

Tasks marked complete there are done. Do not re-dispatch them — re-running a finished task sequence is the most expensive failure in this workflow. The ledger records the commit range per task, so `git log` corroborates it. Trust the ledger and git over recollection.

The ledger also carries the accumulated decisions: why `ctx` is scoped the way it is, why `go mod tidy` is banned, why the stdin watcher sits outside the WaitGroup. Handing those to a reviewer prevents it re-opening settled questions and spending its attention on ground already covered.

## The per-task loop

Everything moves as files, not pasted text — what you paste into a prompt stays in your context for the rest of the session.

```bash
SK="$HOME/.claude/plugins/cache/claude-plugins-official/superpowers/6.1.1/skills/subagent-driven-development/scripts"

"$SK/task-brief" docs/superpowers/plans/2026-07-25-gobsidian-v01.md <N>   # → .superpowers/sdd/task-N-brief.md
"$SK/review-package" <BASE> HEAD                                          # → .superpowers/sdd/review-*.diff
```

`BASE` is the commit recorded **before** dispatching the implementer. Never `HEAD~1` — it silently drops all but the last commit of a multi-commit task.

Then: dispatch implementer → package the diff → dispatch reviewer → fix pass for Critical and Important findings → re-review → record in the ledger.

Model choice has mattered here. Transcription from complete plan code runs fine on the cheapest tier. Anything touching concurrency, platform behaviour, or the path-confinement boundary earns a stronger reviewer — the two Opus reviews in this project each found defects that lighter passes had missed, including one that only surfaced by simulating Linux against the standard library.

## Fix the plan, not just the code

Nearly every real defect found so far originated in the plan's own snippets, not in the transcription. When a review finds one, **update the plan and commit that before dispatching the fix**, then point the fixer at the plan section.

Two reasons. The next implementer transcribes from the plan, so an uncorrected plan re-injects the bug. And a subagent may wipe your uncommitted edit while cleaning up its own work — commit first, then verify the text is actually on disk.

## Gates that block a milestone

`pwsh -File scripts/test_orphans.ps1 -Cycles 100` must report zero orphans. This is the release-blocking criterion the product exists for, and it failed genuinely on its first honest run — five of five cycles left an orphan against code that had already passed task-level review. If it fails, that is the most valuable result available; diagnose with `--log-level debug` and report which mechanism did or did not fire. Never weaken the test to make it pass.

The full check before tagging a milestone:

```bash
go vet ./... && go test -race ./... && gofmt -l .
GOOS=linux go vet ./... && GOOS=darwin go vet ./...
pwsh -File scripts/check_net.ps1
pwsh -File scripts/build.ps1
pwsh -File scripts/test_orphans.ps1 -Cycles 100
```

## What is done

M0 is complete and tagged `m0-lifecycle`: lifecycle with three shutdown mechanisms, `internal/vault`, the minimal MCP server, `doctor`, and the orphan gate.

M1 starts at Task 12 — parser types, frontmatter, slug — and runs through Task 26, which cuts v0.1. The plan's Task 18 builds the golden-file corpus; Task 25 needs a one-time Obsidian plugin run to produce the parity reference, so it cannot be fully automated.

M2–M6 are specified at task altitude only. Each gets its own detailed plan written against the code that exists then, rather than in advance where the detail would go stale.

## Carried gaps

Recorded rather than hidden, and the final whole-branch review should cover them:

- Task 9's last fix pass and Task 10's fix pass closed on direct evidence — mutation proofs, twenty clean exit-code runs, a measured walk count — without a fresh review round. `cmd/gobsidian/serve.go`, `internal/mcpsrv/convert.go` and `internal/doctor/` need that attention.
- The orphan harness is always won by `stdin-eof`, so the parent watcher and signal handling remain unverified end to end. M6 needs a scenario where stdin stays open and the parent dies — precisely what the parent watcher exists for.
