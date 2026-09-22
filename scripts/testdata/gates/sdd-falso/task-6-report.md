# Task 6 report — fixture: a ~~~ block is not closed by ```

## Status
DONE.

## What I ran

~~~bash
pwsh -File scripts/verify.ps1
```
# red: passou
# green: passou
# mutation: apaguei a regra e rodei
# verification: colada aqui
```
~~~

The only lines shaped like headings live inside a ~~~ block. The ``` lines
inside it are content: a fence closes only with the same character. An
auditor that treats ``` as closing the ~~~ block would see the four shell
comments as headings and report 0 SECAO-AUSENTE. Expected: 4.

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
