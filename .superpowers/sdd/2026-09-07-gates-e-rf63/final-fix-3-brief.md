# Final-fix brief, round 3 — the three parked findings: F7, N1, F9

F7 and F9 are in `review-final.md`; N1 is in `rereview-final.md`. The owner asked for all three to be closed. Read those sections first.

## A. F7 — `--amend` is an unconditional allow (`scripts/pre_commit_docs.ps1:136-138`)

Today, in the non-simulated path only: `--amend` without `--no-edit` -> `allow "amend de mensagem"` BEFORE the message or the stage is looked at. When nothing is staged the normal path already answers `allow "nada em stage"`, so the exception only acts when something IS staged — and then it is a bypass: `git commit --amend -m "fix: x"` with a `.go` staged and no doc passes.

Fix: delete the three lines of the exception. An amend follows the normal rule (message read, stage checked). Replace them with a comment saying why the exception is gone (the measured reasoning above: with an empty stage the normal path already allows; with a non-empty stage the exception was the hole; and the exception lived outside `-Simular`, so `check_gates` could not see it). Do not add any amend special case anywhere.

Cases in `scripts/check_gates.ps1` (after the F6 case):

```powershell
    # F7 da revisao final: --amend nao e mais allow incondicional. Com .go em
    # stage e sem doc, amend e commit igual; com nada em stage o caminho normal
    # ja responde allow.
    Caso -Nome '--amend -m sem escotilha, .go sem doc -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook 'git commit --amend -m "fix: x"' $go)

    Caso -Nome '--amend -m com escotilha, .go sem doc -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook 'git commit --amend -m "fix: x [sem-doc]"' $go)

    Caso -Nome '--amend --no-edit, nada em stage -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook 'git commit --amend --no-edit' @())
```

Check how `Decisao-Hook` passes an empty `-EmStage`; if `@()` does not survive the `-EncodedCommand` plumbing, use whatever the script already does for "nothing staged" or add a `$nada = @()` variable — but the case must exist and must be `allow` with reason `nada em stage` (verify with `Motivo-Hook`, and make THAT the assertion if it is cheap: `-Esperado 'nada em stage' -Obtido (Motivo-Hook ...)`).

RED for F7: the first case must fail (`allow`) before the deletion — but the exception is outside `-Simular`, so `-Simular` never reached it and the case may already pass. If it already passes at RED, say so explicitly in the report: the RED of F7 is that the exception was unreachable by the harness, and the deletion is what makes the case meaningful. Then the mutation proof is what carries the weight: re-add the three deleted lines but OUTSIDE the `if (-not $Simular)` block (i.e. reachable) -> the first case must fail with `allow`; remove them again; paste both outputs.

## B. N1 — `\` escape in `Segmento-Commit` (`scripts/pre_commit_docs.ps1`)

Bash: outside single quotes, `\` escapes the next character (`\"` is a literal quote, not a quote toggle; `\#` is a literal `#`). Inside single quotes nothing escapes. Fix in the walker: when `$c -eq '\'` and `-not $aspaS`, do `$i++; continue` (skip the escaped character). Put it BEFORE the quote checks. One comment line naming N1 and the rule.

Case (after the F7 cases):

```powershell
    # N1 da re-revisao final: \" e aspa literal, nao abre nem fecha aspas; o #
    # depois dela continua sendo comentario de shell.
    Caso -Nome 'aspa escapada antes do comentario, -F sem escotilha -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook "git commit -F $msgSem \`" # was: -m `"wip [sem-doc]`"" $go)
```

Mind the PowerShell quoting: the command string handed to the hook must be exactly `git commit -F <path> \" # was: -m "wip [sem-doc]"` (a backslash, a double quote, a space, a hash). Print it once with `Write-Host` while developing if unsure, then remove the print. RED: this case returns `allow` before the fix (the reviewer measured it). Mutation proof: remove the `\` branch -> the case fails; restore; paste.

## C. F9 — `docs/ESTADO.md:637` has 357 columns

Re-wrap that one line at ~80 columns, keeping the two-space list indentation of its neighbours and changing no words. `python -c "open('docs/ESTADO.md',encoding='utf-8').read()" && echo "[OK] UTF-8"` after.

## D. Docs

- `docs/papeis/documentador.md`, the paragraph about the hook (around line 146-152): add one sentence: "`--amend` segue a mesma regra: com `.go` em stage e sem doc, precisa da escotilha ou do doc; com nada em stage, passa."
- `docs/ESTADO.md`: in the `pre_commit_docs` debt item (around lines 606-616, the one marked "corrigido em 2026-09-07 (Task 187)"), add one sentence: "A exceção de `--amend`, que era allow incondicional e vivia fora do alcance de `-Simular`, foi removida na mesma data (rodada 3 da revisão final); amend segue a regra normal."
- UTF-8 check on both.

## Evidence in the report

Append to `final-fix-report.md` under `## Round 3 (F7, N1, F9)` with sub-headings `### RED`, `### GREEN` (24/24 expected: 20 + 3 F7 + 1 N1), `### Mutation proofs` (two, past tense, output pasted: the F7 re-add and the N1 branch removal; `git diff --stat` clean after restoring), `### Verification` (`check_gates` 24/24; `pwsh -NoProfile -File scripts/verify.ps1 -SkipCross -SkipNet` tail + EXIT; UTF-8 lines; `audit_reports.ps1 -Task 187` and `-Task 188` still 0 SECAO-AUSENTE). Keep `## Progresso` lines with real times.

## Commit

Only: `scripts/pre_commit_docs.ps1 scripts/check_gates.ps1 docs/ESTADO.md docs/papeis/documentador.md`. Message file `final-fix-3-commit.txt` in this directory:

```
fix(hook): --amend follows the commit rule, and an escaped quote is a literal

The pre-commit hook allowed any --amend before reading the message or the
stage. With an empty stage the normal path already allows, so the
exception only ever acted when something was staged, which made it a
bypass; it also lived outside the simulated path, where check_gates could
not see it. The exception is gone: an amend reads its message and checks
its stage like any commit. The segment cutter now treats a backslash
outside single quotes as bash does, so an escaped quote no longer flips
the quote state and lets a shell comment reach the message extractor.
ESTADO.md re-wraps one 357-column line.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01P5wkw6PAdBFzF3uB1w1jNj
```

`git commit -F .superpowers/sdd/2026-09-07-gates-e-rf63/final-fix-3-commit.txt`. The hook allows it (no `.go` staged).
