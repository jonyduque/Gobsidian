# Task 5 report — fixture: a ```` block with a ``` context line inside it

## Status
DONE.

## Diff (`git show HEAD -- CLAUDE.md`)

````diff
@@ -108,9 +108,10 @@ scripts/           gates e utilitarios
 As justificativas estao logo abaixo do bloco:
 
 ```
 text  vault  config  lifecycle      folhas
-service  → index, parser, search, vault, writer
+service  → index, parser, search, text, vault, writer
````

### RED — before the fix
(saida colada)

### GREEN — after the fix
(saida colada)

## Mutation proofs
(saida colada)

## Verification

```
pwsh -File scripts/verify.ps1
[OK] Bateria completa. Pode commitar.
```

One ```` (four-backtick) block containing ONE ``` line copied from the diffed
file, then the four real headings, then an ordinary ``` block. Under CommonMark
a shorter fence does not close a longer one, so the headings are real. An
auditor that toggles on every fence line closes the ```` block at the inner
```, reopens it at the real closing ````, and hides everything up to the next
fence — the four headings included. That is the shape of the real
final-fix-report.md of the broken-links plan, whose diff block carried a
" ```" context line from CLAUDE.md.

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
