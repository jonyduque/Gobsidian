---
name: verifying-the-verifier
description: Rules for building or trusting a gate, checker, hook, harness or measurement script. Use when writing or changing a script that reports pass/fail or allow/deny (pre_commit_docs.ps1, audit_reports.ps1, check_gates.ps1, verify.ps1 steps), when a gate goes green or red unexpectedly, when triaging findings from a checker, when a plan or brief prescribes harness code, or before believing any [OK] on screen.
---

# Verifying the verifier

The existing skills cover tests that cannot fail (`preventing-false-pass-and-offset-bugs`) and proof by mutation (`mutation-proof-discipline`). This one covers the layer above: **the gate, the checker and the measurement script themselves**.

Every defect below happened in this repository. None was in the product. All of them were in the thing that was supposed to tell us whether the product was fine.

## The rule

**A gate reports what it measured, not what it appears to cover.** Before trusting a green, answer: *what, exactly, did this run observe?* The gap between that answer and what the command's name suggests is where these defects live.

## Checklist before trusting a gate

Run through this whenever a gate goes green, and whenever you write one.

- [ ] **What artifact did it exercise?** A harness that runs a prebuilt binary can validate code from days ago. `test_orphans.ps1` gave three `[OK]` against a binary predating the subcommand a fourth scenario needed — and the failure message blamed the daemon's startup timing. A gate that does not build must refuse an artifact older than the source.
- [ ] **Does every dispatch path reach the guard?** A check placed after an early `exit` covers the paths that reach it and no others. The stale-binary guard above was first written *below* the block that resolves its own paths and exits — it covered three scenarios and not the fourth, which is precisely the defect it existed to prevent.
- [ ] **Can it hang?** `StreamReader.Peek()` blocks on an empty pipe despite the name, so the deadline test one line above was never reached: one cycle sat for 15h44m. Every wait needs a bound that is actually reachable.
- [ ] **Does a failure mean a defect, or an unobserved run?** A cycle that never launched observed nothing — neither success nor leak. Counting it as failure measures machine load. Distinguish, tolerate a small reported fraction, and keep "measured nothing at all" fatal at any tolerance.
- [ ] **Does the error path work?** A directive parser called a reporting function whose output silently joined the return value, so the map came back as an array and the next index threw. It only happened on malformed input — the path nobody exercises until they need it.
- [ ] **Does it see every input it claims to?** `audit_reports.ps1` globbed `task-*-report.md`; the two real `final-fix-report.md` were never audited, and one had none of the four required sections. Measured on 2026-09-07 when the glob became `*-report.md`: 150 → 152 reports, 199 → 203 `SECAO-AUSENTE`. A filter that excludes silently is a gate with a hole nobody can see.
- [ ] **Does it read the same input the real caller gives it?** The pre-commit hook looked for the `[sem-doc]` hatch anywhere on the command line, so a shell comment (`git commit -F msg # was: -m "wip [sem-doc]"`) and an earlier chained commit both supplied it. The hook now cuts the line to the last `git commit` segment before the first unquoted `#`, `&&`, `||`, `;`, `|` and reads only `-m`/`--message=`/`-am` or the `-F` file.
- [ ] **Is every branch reachable from the simulated path?** The hook's `--amend` exception lived outside `-Simular`; `check_gates` never exercised it, and it was an unconditional `allow` with a `.go` staged and no doc. The reviewer called it cosmetic; running the exact command showed the `allow`. A branch the harness cannot reach is a branch without a test.
- [ ] **Is the signal drowning?** Ten permanent benign findings teach people to ignore the output. Either fix them or dispense them **individually, with a stated reason, still printed**. Never with a global list: it suppresses the token everywhere, including in a document that later makes a false claim about it.

## A gate rule ships with three cases

Gates that decide on text lie the way a test that cannot fail lies. Each new rule in `pre_commit_docs.ps1` or `audit_reports.ps1` lands in the same commit as three cases in `scripts/check_gates.ps1`:

1. **What it must refuse** — the literal input that motivated the rule.
2. **What it must accept** — the closest legitimate input (`git commit -m "fix: issue #12 [sem-doc]"`: the `#` is inside quotes).
3. **The inverse mechanism** — what a naive implementation of the rule breaks. Stripping fenced code so a pasted `# comment` is not a heading is the rule; a fence that is never closed swallowing every real heading after it is the inverse, and it was found by a re-review (fixture `task-4-report.md`).

All three use the same minimal fixture so the `allow` can only come from the rule under test (for the hook: one `.go` staged, no doc). Then the mutation proof, same as for Go: put the old line back, run `check_gates.ps1`, paste the cases that failed by name, restore. If the harness cannot express the case — `pwsh -File x.ps1 -Param @('a','b')` never binds an array; use `-EncodedCommand` — fix the harness, not the case.

`Secoes-Ausentes` once counted matches in the auditor's output: a fixture that did not exist made the auditor exit 2 with empty output, and the case passed as "0 missing". Exit 2 is `erro`, never `0`. Likewise `Decisao-Hook` read only the decision, so a hook that threw returned `allow` from its `catch` and passed — the case now also checks `permissionDecisionReason` names the hatch.

## Harness code in a plan is run before it is dispatched

The gates plan carried a `check_gates.ps1 -EmStage @('a','b')` invocation that cannot bind, and RED/GREEN regexes that did not match the real output. Nobody had executed a line; the implementer found out mid-task and the round became plan debugging. Any script, command or regex a brief prescribes is executed once by the orchestrator, against the repository, before the dispatch.

## Never trust an exit code through a pipe

`cmd | tail` returns `tail`'s status. This produced two wrong reports in one session: a CI run declared green when it had failed, and a mutation proof declared verified when it was inconclusive.

```bash
cmd > /tmp/out.log 2>&1; echo "exit=$?"   # correct
cmd | tail -5; echo "EXIT=$?"             # tail's status, not cmd's
```

Same trap in the background: a command piped to `tail` writes nothing to its output file until it terminates, so progress is invisible for the whole run.

## Prove the checker bites

A checker that has never failed is indistinguishable from one that cannot. Before committing one, plant a defect it must catch, run it, paste the output, remove the probe, confirm the tree is clean.

```bash
# plant, run, restore -- three lines, and they are the difference between
# "I wrote a checker" and "I have a checker"
cp target /tmp/bak && printf '<defeito>' >> target
pwsh -File scripts/check_x.ps1; echo "exit=$? (quer 1)"
cp /tmp/bak target
```

Both failure modes, not one. The README anchor checker has two — broken link and section with no link — and only planting both proved both branches live.

## A proof can be over-determined

Exit 0 from `mutate.ps1` means the test failed under mutation. It does **not** mean the assertion you care about is what failed.

One mutation here killed the test with a nil-pointer panic, because the mutated branch left a variable nil — the count assertion the test message described never ran. The rule held anyway, but the report would have claimed evidence it did not have. When the failure output does not name the rule, find a mutation that compiles cleanly and trips the assertion:

```
FAIL: iniciar foi chamado 10 vez(es), esperado exatamente 1   <- proves the rule
FAIL: panic: runtime error: invalid memory address            <- proves something broke
```

`mutate.ps1` exit 2 (inconclusive, usually a broken build) is not coverage either. Report which of the three you got.

## Measurement scripts are gates too

- **Metric noisier than the effect measures nothing.** `FreePhysicalMemory` varied 93 MB across repeats of the same configuration while the effect under test was 16 MB. Switching to per-process Working Set made the cell decidable — and reversed the recommendation.
- **Compare like with like.** Numbers taken from an earlier ledger were measured on a 4,490-note vault; the new ones on 5,619. Mixing them would have presented vault growth as an optimization. Re-measure all cells in one session or state the difference in the cell.
- **Kill by PID you launched, never by name.** `Stop-Process -Name gobsidian -Force` killed the user's live editor session and the concurrent orphan gate. The gate then found no survivors — a **false green**, which is the dangerous direction.

## Concurrency in a shared checkout

Two agents in one worktree is a hazard, not a speedup:

- `git add <explicit-path>` still stages another process's uncommitted work in that same file. Run `git diff <path>` first.
- A measurement and a gate running together contaminate each other. Serialize.
- Whoever cleans up must scope the cleanup to what it started.

## Red flags

| Thought | Reality |
|---|---|
| "It passed, so we're covered" | Ask which artifact, which paths, how many cycles measured. |
| "The check is obviously right" | The stale-binary guard was obviously right and covered 3 of 4 paths. |
| "These findings are all noise" | Then dispense them individually with reasons, so a real one stands out. |
| "The exit code was 0" | Through a pipe, that is the last command's code. |
| "Mutation exited 0, rule verified" | Only if the failure output names the rule. |
| "That branch is pre-existing and cosmetic" | Run the exact command through it. `--amend` was an unconditional allow. |
| "The glob covers the reports" | List what it does not match. Two real reports were never audited. |
| "I'll re-run, it's flaky" | Flaky gate = no gate. Find whether it failed to measure or found a defect. |
