---
name: verifying-the-verifier
description: Rules for building or trusting a gate, checker, harness or measurement script. Use when writing a script that reports pass/fail, when a gate goes green or red unexpectedly, when triaging findings from a checker, or before believing any [OK] on screen.
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
- [ ] **Is the signal drowning?** Ten permanent benign findings teach people to ignore the output. Either fix them or dispense them **individually, with a stated reason, still printed**. Never with a global list: it suppresses the token everywhere, including in a document that later makes a false claim about it.

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
| "I'll re-run, it's flaky" | Flaky gate = no gate. Find whether it failed to measure or found a defect. |
