---
name: gobsidian-execution
description: Resume and continue executing the gobsidian implementation plan task by task, with the ledger, task briefs, review packages, and milestone gates this project uses. Use whenever asked to continue the plan, work on the next task, pick up where the last session stopped, check what is done, or execute any numbered task from docs/superpowers/plans/. Also use before dispatching an implementer or reviewing someone else's task, since the per-task loop and the accumulated decisions here are what keep reviews from re-litigating settled ground. Read the ledger before assuming anything about progress — context does not survive between sessions but the ledger does.
---

# Executing the gobsidian plan

The plan lives in `docs/superpowers/plans/2026-07-25-gobsidian-v01.md` and is the source implementations are transcribed from. Tasks are numbered; M0 is Tasks 1–11, M1 is Tasks 12–26.

## Before anything: read the ledger

```bash
pwsh -File scripts/sdd.ps1 status
```

Tasks marked complete there are done. Do not re-dispatch them — re-running a finished task sequence is the most expensive failure in this workflow. The ledger records the commit range per task, so `git log` corroborates it. Trust the ledger and git over recollection.

The ledger also carries the accumulated decisions: why `ctx` is scoped the way it is, why `go mod tidy` is banned, why the stdin watcher sits outside the WaitGroup, why the plain Dataview field is a line-level construct. Those exist so a reviewer does not spend its attention re-opening settled ground.

**The ledger lives at `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`.** The flat path `.superpowers/sdd/progress.md` is a pointer and must not be read as state. Both existed for a while and drifted — one had sixteen tasks recorded, the other six. Consolidating them was not cosmetic: a session reading the smaller one would have re-dispatched finished work.

## The division of labour

Execution goes to the cheapest model that can do it. Review and planning stay with the strongest one — the reviews are where this project's real defects were caught, and every one of them originated in the plan's own snippets rather than in the transcription.

**Tasks 19–26 of the plan are self-contained.** Each carries, inside its own section: where it fits, the closed decisions that bind it, the traps already paid for that apply to it, verifications beyond the numbered steps, execution rules, and the report contract. The extracted brief is enough to execute — do **not** paste accumulated context into the dispatch prompt. That was necessary before and is now duplication.

## The per-task loop

Everything moves as files, not pasted text — what you paste into a prompt stays in your context for the rest of the session.

```bash
pwsh -File scripts/sdd.ps1 base 19     # BEFORE the implementer starts
pwsh -File scripts/sdd.ps1 brief 19
pwsh -File scripts/sdd.ps1 review 19   # packages the diff since the recorded base
```

`base` exists because `review-package` needs the commit from **before** the task began. `HEAD~1` silently drops all but the last commit of a multi-commit task, and the review then examines half a diff without saying so.

Use `sdd.ps1` rather than calling the plugin scripts directly. Their path embeds the plugin version, which went from 6.1.1 to 6.2.0 mid-project — changing `review-package`'s signature and moving artefacts into a per-plan subdirectory. A literal path breaks on the next update, in the middle of a delegation, with a message that does not say what happened.

Then: dispatch implementer → package the diff → review → fix pass for Critical and Important findings → re-review → record in the ledger.

## Fix the plan, not just the code

Nearly every real defect found so far originated in the plan's own snippets, not in the transcription. When a review finds one, **update the plan and commit that before dispatching the fix**, then point the fixer at the plan section.

Two reasons. The next implementer transcribes from the plan, so an uncorrected plan re-injects the bug. And a subagent may wipe your uncommitted edit while cleaning up its own work — commit first, then verify the text is actually on disk.

## Reviewing

Two sibling skills carry most of the technique and are worth invoking rather than reconstructing: `proving-tests-can-fail` for whether coverage is real, and `subagent-session-hygiene` for the coordination failures that show up across a long session.

What has actually found defects here, in order of yield:

**Mutation, not reading.** Delete the rule, confirm a named test fails, restore. In Task 13 seven of the module's rules survived mutation with the suite green — including the one the fix's own comment defended. Reading the tests found none of them.

**A/B against the previous commit.** The Dataview extension deleted wikilinks, embeds and Markdown links from the graph that the commit before it already collected. Only comparing the two revealed it as a regression rather than a pre-existing gap.

**Checking whether a fixture is inert.** Task 8's exclusion fixtures used extensions the extension filter would drop anyway, so deleting the exclusion rule changed no count. Coverage was reported that did not exist.

**Asking what a value of zero means.** Zero is a legitimate byte offset, a legitimate debounce, a legitimate note count. Wherever it can also mean "unknown" or "omitted", the two must be distinguishable — `offsetUnknown = -1`, `ReadOnlySet`, an error rather than an empty walk.

Model choice has mattered. Transcription from complete plan code runs fine on the cheapest tier. The path-confinement boundary, concurrency, and platform behaviour earned the strongest reviewer — one of those reviews found a portability regression only by transcribing the standard library's Unix code paths and simulating a Linux run.

## Gates that block a milestone

`pwsh -File scripts/test_orphans.ps1 -Cycles 100` must report zero orphans. This is the release-blocking criterion the product exists for, and it failed genuinely on its first honest run — five of five cycles left an orphan against code that had already passed task-level review. If it fails, that is the most valuable result available; diagnose with `--log-level debug` and report which mechanism did or did not fire. Never weaken the test to make it pass.

The full check:

```bash
pwsh -File scripts/verify.ps1          # build, -race, vet ×3, gofmt, check_net
pwsh -File scripts/build.ps1
pwsh -File scripts/test_orphans.ps1 -Cycles 100
```

## What is done

M0 is complete and tagged `m0-lifecycle`: lifecycle with three shutdown mechanisms, `internal/vault`, the minimal MCP server, `doctor`, and the orphan gate.

M1 is in progress. **Tasks 12–18 are closed and reviewed:** parser types, frontmatter with a body offset, slugs, headings with section offsets, the four goldmark inline extensions, and the golden-file corpus that freezes them — 48 pairs, no orphan in either direction, harness proven to fail by tampering.

Pending, strictly sequential: 19 (index), 20 (resolution), 21 (backlinks), 22 (queries), 23 (service), 24 (tools and resources), 25 (parity), 26 (v0.1 closure).

Task 25 needs a one-time Obsidian plugin run to produce the parity reference, so it cannot be fully automated. Its brief instructs the test to **skip** when the artefact is absent rather than fail — a test that fails for want of a human artefact is one people learn to ignore.

M2–M6 are specified at task altitude only. Each gets its own detailed plan written against the code that exists then, rather than in advance where the detail would go stale.

## Carried gaps

Recorded rather than hidden, and the final whole-branch review should cover them:

- Task 9's last fix pass, Task 10's fix pass, and Task 14's fix pass closed on direct evidence — mutation proofs, twenty clean exit-code runs, a measured walk count — without a fresh review round. `cmd/gobsidian/serve.go`, `internal/mcpsrv/convert.go`, `internal/parser/ext_wikilink.go`, `internal/parser/ast.go` and `internal/doctor/` need that attention.
- The orphan harness is always won by `stdin-eof`, so the parent watcher and signal handling remain unverified end to end. M6 needs a scenario where stdin stays open and the parent dies — precisely what the parent watcher exists for.
- The `vault.StripBOM` → `parser.Parse` seam is tested at neither end together. The golden `edge/bom.md` pins that a BOM'd note parsed directly yields `{}` — no headings, no title. Routed into Task 19's verifications, since that is where the composition is built.
- A triage list of deferred Minor findings from M1 sits in the ledger under `## Minor diferidos de M1`. The frontmatter one is the sharpest: a closing delimiter with a trailing space makes the whole YAML block become body, losing metadata in silence.
