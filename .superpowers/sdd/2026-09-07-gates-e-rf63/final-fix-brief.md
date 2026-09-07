# Final-fix brief — review-final.md findings F1–F6, F8, F10

Read `review-final.md` in this directory for the full text of each finding (locals, measured scenarios, suggested fixes). This brief says WHAT to change; the review says WHY. F7 and F9 are out of scope (ruled: F7 pre-existing, F9 cosmetic).

## A. `scripts/pre_commit_docs.ps1` — F1, F2, F3, F10

A1. Before extracting the message, cut the command line down to the segment of the **last** `git commit` on it: start at the last match of `git\s+commit`, end at the first `#`, `&&`, `||`, `;` or `|` that is **outside** single/double quotes (walk the string character by character tracking quote state; a `#` inside quotes is message text). Put this in a function `Segmento-Commit([string]$linha)` returning the segment (or the whole line if no `git commit` is found — the non-simulated path already returned `allow` for that). Then `Extrair-Mensagem` runs on the segment. Comment it with the two measured bypasses from F1/F2 (shell comment containing `-m "... [sem-doc]"`; two chained commits).

A2. Accept grouped short flags (F3): in `$reM` replace the leading `(?:-m|--message)` with `(?:(?<![\w-])-[a-zA-Z]*m|--message)`. Keep `$reF` as is (`-F` is uppercase and is not grouped in practice).

A3. Deny text (the `-F` sentence region): add one line after the `-F` sentence: `"O hook so le -m/--message= e o arquivo de -F/--file=; -C, --fixup, heredoc e"` + `"caminho sem aspas nao sao lidos e caem aqui."` (ASCII, same style as neighbours).

A4. `docs/papeis/documentador.md:139-141` (F10): replace "nunca na linha de comando fora dela" with "só no trecho do último `git commit` da linha, e só onde o hook consegue ler a mensagem: `-m`/`--message=` (inclusive `-am`) e o arquivo de `-F`/`--file=`; comentário de shell e comandos encadeados antes dele não contam". UTF-8 check after.

## B. `scripts/check_gates.ps1` — new hook cases (F1, F2, F3, F6)

Add after the existing 9 hook cases:

```powershell
    # F1 da revisao final: -m dentro de um comentario de shell nao e a mensagem.
    Caso -Nome '-m com escotilha dentro de comentario de shell, -F sem ela -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook "git commit -F $msgSem # was: -m `"wip [sem-doc]`"" $go)

    # F2: dois commits na linha — vale o ultimo, que nao tem escotilha.
    Caso -Nome 'dois git commit encadeados, escotilha so no primeiro -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook 'git commit -m "docs: a [sem-doc]" && git commit -m "feat: b"' $go)

    Caso -Nome 'dois git commit encadeados, escotilha so no ultimo -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook 'git commit -m "feat: a" && git commit -m "docs: b [sem-doc]"' $go)

    # F3: flags curtas agrupadas.
    Caso -Nome '-am com escotilha -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook 'git commit -am "fix: x [sem-doc]"' $go)

    Caso -Nome 'escotilha dentro de aspas com # no texto -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook 'git commit -m "fix: issue #12 [sem-doc]"' $go)
```

All five use `$go` (a `.go` staged, no doc): the `allow` can only come from the hatch in the message the hook actually read. Check the variable names against the existing cases (`$go`, `$msgSem` or whatever the script calls them) and adapt.

F6: add a function `Motivo-Hook` returning `permissionDecisionReason`, and one case:

```powershell
    Caso -Nome 'motivo do allow e a escotilha, nao o catch do hook' `
        -Esperado 'escotilha [sem-doc] na mensagem do commit' -Obtido (Motivo-Hook 'git commit -m "fix: x [sem-doc]"' $go)
```

To avoid duplicating the EncodedCommand plumbing, refactor `Decisao-Hook` into `Invocar-Hook` returning the parsed JSON's `hookSpecificOutput` object — or `$null` on error — and have `Decisao-Hook`/`Motivo-Hook` read one property each, returning `'erro'` on `$null`.

## C. `scripts/audit_reports.ps1` — F5

Before the `foreach ($Sec in $Required)` loop, build `$BodySemCercas` from `$Lines`: walk the lines, toggle a flag on every line matching `^\s*(```|~~~)`, drop lines while the flag is on (and the fence lines themselves), join with "`n". Match `$Required` against `$BodySemCercas`. Everything else (HEDGE, NAO-RESPOSTA, CURTO...) keeps using `$Body` — a hedge inside a pasted output is still worth flagging. Comment: pasted shell output uses `# comment` lines, which look like headings to `^#{1,6}\s`; measured in review-final F5 (four sections "present" in a report that had none).

Fixture `scripts/testdata/gates/sdd-falso/task-3-report.md` (ASCII, >2000 bytes with the same neutral filler approach as task-1): a report whose ONLY `#`-lines other than the `# Task 3 report` title are shell comments inside a ```bash fence, naming red, green, mutation and verification. Expected: 4 SECAO-AUSENTE.

## D. `scripts/check_gates.ps1` — F4 + new audit cases

D1. In `Secoes-Ausentes`, after the call: `if ($LASTEXITCODE -eq 2) { return 'erro' }` before counting (exit 2 = root missing or no report matched; exit 1 = findings, the normal path for fixtures 1 and 3; exit 0 = no findings, normal for fixture 2).

D2. Add:
```powershell
    Caso -Nome 'secoes so em comentario de shell dentro de cerca -> 4 SECAO-AUSENTE' `
        -Esperado '4' -Obtido (Secoes-Ausentes '3')

    Caso -Nome 'fixture inexistente -> erro, nao 0' `
        -Esperado 'erro' -Obtido (Secoes-Ausentes '9999')
```

## E. `docs/ESTADO.md` — F8

Around lines 620-621: change "Efeito medido nos relatórios reais: `79` → `203` `SECAO-AUSENTE`" to "Efeito medido em 2026-09-07 sobre 150 relatórios: `79` → `203` `SECAO-AUSENTE` (o número cresce com o corpus; não o corrija sem re-medir)". UTF-8 check after.

## Evidence required in the report (headings, not prose)

- `### RED` — check_gates output with the new cases added BEFORE the code changes (the new cases must fail; paste which).
- `### GREEN` — after: all cases `[OK]` (count them — expect 11 + 6 hook + 2 audit = 19).
- `## Mutation proofs` — two, past tense, output pasted: (1) make `Segmento-Commit` return `$linha` unchanged → the F1/F2 cases fail; restore. (2) make `$BodySemCercas = $Body` → the fixture-3 case fails; restore. Confirm `git diff --stat` on the two scripts shows only the intended change after restoring.
- `## Verification` — `pwsh -File scripts/verify.ps1 -SkipCross -SkipNet` tail + EXIT (full gate ran in Task 188; the cross/net steps do not touch these files); UTF-8 lines for the two .md; `audit_reports.ps1 -Task 187` and `-Task 188` still 0 SECAO-AUSENTE (they have real headings plus fenced output — this proves fence stripping does not eat real headings).

## Commit

Message file `final-fix-commit.txt` in this directory:

```
fix(gates): read only the last git commit segment, and ignore fenced code in reports

Final review of the gates plan found the two checkers still matching text
they should not see. pre_commit_docs now cuts the command line to the
last `git commit` segment before the first unquoted `#`, `&&`, `;` or
`|`, so a shell comment or an earlier chained commit cannot supply the
hatch; it also reads grouped short flags (`-am`). audit_reports drops
fenced code blocks before looking for section headings, so a pasted
`# comment` line no longer counts as a section. check_gates carries a
case for each, and its audit helper no longer reads an early exit as
"zero sections missing".

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01P5wkw6PAdBFzF3uB1w1jNj
```

`git add scripts/pre_commit_docs.ps1 scripts/check_gates.ps1 scripts/audit_reports.ps1 scripts/testdata/gates/sdd-falso/task-3-report.md docs/ESTADO.md docs/papeis/documentador.md` then `git commit -F .superpowers/sdd/2026-09-07-gates-e-rf63/final-fix-commit.txt`. The hook allows this: no `.go` staged.
