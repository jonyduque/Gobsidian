# Task 3 report — fixture: section names only inside a shell-comment fence

## Status
DONE.

## What I ran

```bash
# rodei red antes do conserto
pwsh -File scripts/verify.ps1
# green depois do conserto
pwsh -File scripts/verify.ps1
# mutacao: apaguei a regra e rodei de novo
# verificacao: saida colada abaixo
```

This report has no `### RED`, `### GREEN`, `## Mutation proofs` or
`## Verification` heading anywhere. The only lines starting with `#` besides
the title and the two headings above are shell comments pasted inside the
fenced block, naming the four required words the way a real report's pasted
command output does. The auditor must strip fenced code before matching
headings, or these four comment lines read as four present sections.

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
