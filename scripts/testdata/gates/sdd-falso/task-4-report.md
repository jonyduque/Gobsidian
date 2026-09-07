# Task 4 report — fixture: an unclosed fence must not hide real headings

## Status
DONE.

## What I ran

```bash
pwsh -File scripts/verify.ps1
# red passou, green passou
# mutacao: apaguei a regra e rodei de novo
# verificacao: saida colada abaixo
this fence is opened above and never closed, by accident

### RED — before the fix
(saida colada)

### GREEN — after the fix
(saida colada)

## Mutation proofs
(saida colada)

## Verification
(saida colada)

This report has one ```bash fence that opens and is never closed. The four
required headings above (RED, GREEN, Mutation proofs, Verification) are real
Markdown headings that appear textually AFTER the unclosed opener, exactly as
a report author would write them if they forgot the closing fence while
pasting long command output. An auditor whose fence toggle has no
unmatched-fence handling treats everything from the opener to EOF as fenced
and strips these four real headings along with it, producing four false
SECAO-AUSENTE findings on a report that genuinely has all four sections.

## Filler

This paragraph is neutral padding, written only so the fixture file crosses
the two thousand byte floor that the auditor uses to flag a report as too
short to hold pasted command output. Its content carries no signal at all;
only its length matters here, and the length is measured and pasted into
the task report rather than assumed.

This paragraph is neutral padding, written only so the fixture file crosses
the two thousand byte floor that the auditor uses to flag a report as too
short to hold pasted command output. Its content carries no signal at all;
only its length matters here, and the length is measured and pasted into
the task report rather than assumed.

This paragraph is neutral padding, written only so the fixture file crosses
the two thousand byte floor that the auditor uses to flag a report as too
short to hold pasted command output. Its content carries no signal at all;
only its length matters here, and the length is measured and pasted into
the task report rather than assumed.
