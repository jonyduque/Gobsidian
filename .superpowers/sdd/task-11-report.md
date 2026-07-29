# Task 11 report: orphan-process test

Status: **BLOCKED** — the test is built correctly and fails for real. The
product does not currently meet RNF-10 (zero orphans across abrupt host
death). Per the task brief and top-level instructions, I stopped at Step 5
(small-cycle local run) instead of proceeding to the CI job, the 100-cycle
run, and the `m0-lifecycle` commit/tag, because doing so would either wire a
permanently-red CI job into the tree or falsely tag M0 as release-ready.

Repo left at commit `2281284` (unchanged). Working tree has three new,
**uncommitted** paths: `scripts/build.ps1`, `scripts/test_orphans.ps1`,
`testdata/vault_small/`. No commit or tag was created.

## What I implemented

1. **`scripts/build.ps1`** — copied verbatim from `docs/WINDOWS.md` §7, with
   one required correction: the doc's build target is `.\cmd\cofre`, but the
   actual module layout is `cmd/gobsidian` (there is no `cmd/cofre` anywhere
   in the tree). Fixed the `go build` invocation to target `.\cmd\gobsidian`.
   Verified the ldflags variables (`main.version`, `main.commit`,
   `main.buildDate`) match the `var (...)` block in `cmd/gobsidian/main.go`.
   Ran it: builds `bin\gobsidian.exe` (6.49 MB) successfully under
   `CGO_ENABLED=0`.

2. **`scripts/test_orphans.ps1`** — based on the brief, with `param()` as the
   first statement and `$HostProc` (not `$Host`) as required. I found and
   fixed one bug in the brief's own script (see "Trap #3" below): the
   `-ArgumentList "/c", "\"$BinaryPath\" serve --vault \"$VaultPath\""`
   construction never actually launches the server under PowerShell 7.
   Replaced it with separate, unquoted tokens
   (`"/c", $BinaryPath, "serve", "--vault", $VaultPath`) so .NET's own
   per-argument escaping does the quoting instead of a hand-built string
   colliding with it.

3. **Test vault** — `testdata/vault_small/` did not exist. Created a minimal
   real vault: `.obsidian/config` (marks it as an Obsidian vault),
   `note-one.md` and `note-two.md` at the root with a wikilink between them
   and a `#tag`, and `folder/note-three.md` for a nested directory. This is
   a created fixture, not a pointer to a temp path — it's small enough to
   commit and gives the server real content to open on every cycle.

4. **CI job** — **not added**. Step 6 in my job list is contingent on Step 5
   passing; it didn't, so `.github/workflows/ci.yml` is untouched. Adding a
   job that runs a test known to fail would either turn CI permanently red
   or invite someone to quietly weaken the test to go green — both worse
   than not adding it yet.

5. **100-cycle run** — not performed, for the same reason. I ran 5 cycles
   (see below) as the smallest sample that establishes the failure is not
   noise, then stopped.

6. **Commit / tag** — not created. `m0-lifecycle` asserts the lifecycle
   requirement is satisfied; it is not.

## The literal output of the run

Small-cycle run with the corrected script, `-Cycles 5`:

```
[...] 5 ciclos de encerramento abrupto
WARNING: [!] Ciclo 1: PID 30276 sobreviveu
WARNING: [!] Ciclo 2: PID 30860 sobreviveu
WARNING: [!] Ciclo 3: PID 52124 sobreviveu
WARNING: [!] Ciclo 4: PID 16688 sobreviveu
WARNING: [!] Ciclo 5: PID 47096 sobreviveu
WARNING: [!] FALHA: 5 orfao(s) em 5 ciclos
```
Exit code 1. 5/5 orphans survived the 2-second grace window. I did not run
the 100-cycle version — 5/5 already demonstrates the failure is deterministic
in this environment, and running 100 more cycles of a known-broken server
would only produce 100 more stray processes to clean up for no new
information.

## Answers to the four questions

**Is the server actually running when you kill the parent?**
Yes, verified directly, not assumed. I polled the child process list every
50ms after `Start-Process` and confirmed `gobsidian.exe` appears as a child
of the `cmd.exe` host well before the 700ms mark, and I captured
`--log-level debug` stderr showing `msg="servidor pronto"` logged before the
kill in every manual run. 700ms is comfortably sufficient on this machine.
(This check almost gave a false "it's fine" reading — see Trap #3 below:
with the brief's original argument list, the child *never* started, and
polling for it would have quietly returned nothing every cycle.)

**Are you killing the right thing?**
`taskkill /F /PID $HostProc.Id` (no `/T`) confirmed to kill only `cmd.exe`,
never the tree. Evidence: after `taskkill /F` on the host, the previously
recorded `gobsidian.exe` PID is still present in `Get-Process` for the
entire observation window (up to 25s in manual tests), i.e. `taskkill`
demonstrably did not touch it — exactly the intended scenario (a host dies,
its child is orphaned, and the child's own shutdown mechanisms are what's
being tested, not `taskkill`'s tree behavior).

**Which mechanism is doing the work?**
**None of them, in this scenario, within any observed window (up to 25s).**
I captured `--log-level debug` stderr across multiple manual cycles and the
lifecycle's `"encerramento solicitado" reason=...` line never appears after
the host is killed — the process just keeps running. I traced this to a
concrete, reproducible bug (not a timing fluke):

- **stdin-eof does not fire**: the child inherits the same console as the
  `cmd.exe` host (there's a `conhost.exe` sibling in the process tree). It
  is not a pipe, so there is no "last writer closes -> EOF" event when the
  host dies; the console object persists as long as any process remains
  attached to it, which the child still is.
- **signal does not fire**: `taskkill /F` calls `TerminateProcess` on the
  target only; it does not raise `CTRL_C_EVENT`/`CTRL_BREAK_EVENT` for other
  processes sharing the console, so `watchSignals` never sees anything.
- **parent-gone does not fire, and here's why**: I wrote a throwaway Go
  probe (kept outside the repo, in `%TEMP%`, never touched `internal/` or
  `cmd/`) that calls the exact same `windows.OpenProcess` +
  `windows.GetProcessTimes` pair used by
  `internal/lifecycle/parent_windows.go:parentIdentity`, polling once a
  second against a `cmd.exe` PID I killed independently (confirmed via a
  *different*, unrelated probe process, after explicitly disposing every
  .NET handle I held to it and forcing GC — ruling out "my own test
  harness kept a handle open"). Result: `OpenProcess` kept succeeding and
  `GetProcessTimes` kept succeeding for the full 20-second observation
  window, returning the **same, unchanged creation time** every tick.
  Windows does not destroy/recycle a terminated process's object or its PID
  immediately; both stay queryable for a long time after death.
  `parentIdentity` in `internal/lifecycle/parent_windows.go` reads
  `GetProcessTimes`' four out-parameters (`creation, exit, kernel, user`)
  but **only ever compares `pid` and `creation`** (see `sameProcess`,
  lines 34-38) — it discards `exit` entirely. Since `exit` is exactly the
  field that distinguishes "still running" from "terminated," and the code
  never inspects it, `sameProcess(initial, current)` keeps returning `true`
  for a process that has been dead for tens of seconds, and `watchParent`
  never trips. This is not a rare edge case gated behind handle-retention;
  it reproduced identically with zero references to the dead process held
  anywhere in my process, confirmed by an independent probe process.

  I did not modify `internal/lifecycle/parent_windows.go` — that package is
  closed for this task, and the top-level instructions are explicit that a
  genuine defect found here should be reported, not fixed in-place.

**Does the vault matter?**
Yes for realism (the server does real vault setup, watcher startup, etc. on
every cycle, not a no-op path) but not for this failure — the failure is in
the shutdown path, independent of vault content. I created the fixture
vault described above rather than pointing at a temp directory, since it's
small enough to keep as a committed test asset (once this task un-blocks).

## Would the test catch a regression?

Yes, decisively, for the parent-watch mechanism specifically — it caught a
live regression by design, without me touching a single line of product
code. It is meaningful, not decorative: I proved this by finding and fixing
a bug in the *test itself* first (see Trap #3), which would otherwise have
made the test pass trivially and silently. Once fixed, the test failed
5-for-5, immediately and reproducibly.

Whether it would catch a regression in the *other two* mechanisms
specifically is moot right now, because in this exact scenario (`cmd.exe`
host, console-inherited stdin, `taskkill /F` without `/T`) neither
stdin-eof nor signal ever fires today regardless of whether they're
implemented correctly or deleted — the scenario doesn't exercise them. If
someone "fixed" this test by deleting the stdin monitor or the signal
handler, this specific test would show no change, because neither is
currently what saves the process in this path (nothing currently is). That
is itself worth flagging: the redundancy claimed in `lifecycle.go`'s package
doc ("tres mecanismos independentes... a redundancia e deliberada") is not
actually exercised end-to-end by this scenario the way stdin/signal-based
hosts (e.g. a real MCP client using a pipe, or Ctrl+C) would exercise it.
This orphan test as specified only really stresses the parent-watch path,
and that path is broken.

## Trap #3 (found empirically, not in the brief's list of two)

`Start-Process -ArgumentList "/c", "\"$BinaryPath\" serve --vault \"$VaultPath\""`
(the brief's literal two-element array, second element pre-quoted) fails
silently under PowerShell 7.6.4: `Start-Process` builds the child's real
Win32 command line via `ProcessStartInfo.ArgumentList`, which escapes
embedded `"` characters in each array element (turning them into literal
`\"` sequences) rather than treating them as argument delimiters. `cmd.exe`
then receives a mangled path and fails immediately with "A sintaxe do nome
do arquivo, do nome do diretorio ou do rotulo do volume esta incorreta" —
`gobsidian.exe` never starts, `$ServerPids` is always empty, and the test
prints `[OK] Nenhum orfao em N ciclos` every time, having measured nothing.
I found this by directly polling the child process list and reading
redirected stderr instead of trusting the survivor count, exactly the kind
of check this task asked me to do before believing a green (or in this
case, a suspiciously-green would-be) result. Fixed by passing tokens
unquoted/separately in `-ArgumentList` and letting .NET quote each one
itself.

## Files changed

- `scripts/build.ps1` (new) — build script, `cmd/cofre` -> `cmd/gobsidian` fix applied
- `scripts/test_orphans.ps1` (new) — orphan test, argument-list quoting bug fixed
- `testdata/vault_small/` (new) — `.obsidian/config`, `note-one.md`, `note-two.md`, `folder/note-three.md`
- `.github/workflows/ci.yml` — **not modified** (blocked)
- No files under `internal/` or `cmd/` were modified.

## Self-review

- **Would this test fail if the product regressed?** For the parent-watch
  mechanism: yes, proven — it's currently failing against the *unregressed*
  code, which is the strongest possible evidence it's not decorative.
  For stdin-eof/signal specifically: this particular scenario doesn't
  exercise them either way (see above), which is a genuine gap in scenario
  coverage worth a follow-up (e.g. a variant that redirects stdin through
  an actual pipe host, and a variant that sends Ctrl+Break to a process
  group, to isolate each mechanism the way Task 9's unit tests did in
  isolation but this end-to-end test does not yet).
- **Does the script leave anything behind?** Verified: no stray
  `gobsidian.exe`/probe processes remain (`tasklist` checked clean after
  every experiment), no temp vaults (the vault is a committed fixture, not
  a temp dir), no build artifacts outside `bin/` (which is gitignored).
  I also removed all scratch investigation files (`manual-*.log`,
  throwaway probe binaries in `%TEMP%` and `/tmp`) — none of that is part
  of the deliverable.
- **Is the exit code right?** Yes: `test_orphans.ps1` exits 1 on any
  survivor (verified: it did, in the run above), 0 only if none survive
  (not yet observed to be reachable in this environment).
- **Would the CI job work on a GitHub Windows runner?** Not evaluated in
  depth since the job wasn't added, but worth flagging for whoever resumes
  this: `windows-latest` runners are non-interactive, so `-WindowStyle
  Hidden` should behave the same as here (no console session), but the
  underlying bug (OpenProcess/GetProcessTimes on a dead PID) is a Windows
  kernel behavior, not a console/session artifact, so I'd expect the same
  failure to reproduce on the runner rather than being masked by it.

## Concerns

1. **This blocks M0.** RNF-10 ("zero orphans") is not met. The bug is
   narrow and specific: `sameProcess`/`parentIdentity` in
   `internal/lifecycle/parent_windows.go` need to check whether the target
   process has actually exited (e.g. via the `exit` `Filetime` returned by
   `GetProcessTimes`, or `GetExitCodeProcess` != `STILL_ACTIVE`), not just
   compare `(pid, creation)`.
2. **Scenario coverage gap, independent of the bug above**: even with the
   parent-watch fix, this specific test only ever exercises that one
   mechanism, because a console-inherited `cmd.exe /c` host defeats
   stdin-eof and `taskkill /F` (no `/T`) defeats signal delivery by
   construction. The "three redundant mechanisms" claim needs a test that
   can actually falsify each of the other two, or the redundancy claim is
   partially untested even after this bug is fixed.
3. **Default `ParentCheckInterval` (5s) plus the required 3 consecutive
   failures before `parent-gone` fires means the earliest correct detection
   would be ~15s** even after the exit-time bug is fixed, which is far
   longer than this script's 2-second wait. Whoever fixes the bug should
   also revisit whether 2 seconds is long enough once the mechanism starts
   working, or whether the interval/threshold needs to shrink for the
   default served-locally case.

## Fix pass -- parent exit detection and faithful harness

Status: DONE. Commit 67d7d69 on master, tag m0-lifecycle created (100-cycle
run passed). Built on the predecessor's uncommitted work
(scripts/build.ps1, scripts/test_orphans.ps1, testdata/vault_small/) rather
than replacing it.

### Fix 1 -- parent_windows.go

Transcribed the corrected code from
docs/superpowers/plans/2026-07-25-gobsidian-v01.md ("Task 5" / the
exit-time appendix committed at 11352c5) into
internal/lifecycle/parent_windows.go: identity gained an `exited bool`, set
from a non-zero `exit` Filetime returned by GetProcessTimes; sameProcess
now returns false immediately whenever `b.exited` is true, regardless of
whether pid/created still match. This routes into watchParent's
identity-divergence branch, which triggers immediately rather than waiting
out maxConsecutiveFailures.

Added TestSameProcessRejectsExited to parent_identity_windows_test.go: an
identity with exited=true must not sameProcess()-match an otherwise
identical one. Mutation evidence: reverted the `if b.exited { return false
}` block, ran `go test -race ./internal/lifecycle/... -run
TestSameProcessRejectsExited -v` -- it failed exactly as expected
("identidade com exited=true aceita como mesmo processo -- pai morto nunca
seria detectado"). Restored the fix; re-ran the same test and the full
internal/lifecycle suite -- all green.

### Fix 2 -- faithful harness

Replaced the `cmd /c` + inherited-console launch with a real synthetic
host: scripts/orphan_host.ps1 is a separate pwsh.exe process that builds
the child via System.Diagnostics.ProcessStartInfo with
RedirectStandardInput = $true (UseShellExecute = $false, ArgumentList built
token-by-token per the predecessor's Trap #3 fix, not a hand-quoted
string), starts it, writes the child's PID to a file the outer script
polls, then blocks in an infinite sleep loop holding the pipe's write end
open. scripts/test_orphans.ps1 launches this host per cycle, polls for the
PID file (with a hard timeout, see Fix 3), taskkill /F's only the host (no
/T), waits SettleMs (default 2000ms), and checks survival. Killing the
host (not the child) and not using /T matters: that's what proves the
child's own shutdown mechanisms work, rather than taskkill's tree-kill
behavior.

The child's stderr is inherited from the host process, and the host
process itself is launched by the outer script via `Start-Process
-RedirectStandardError $LogFile`, so each cycle's --log-level debug output
(including reason=...) lands in a per-cycle log file the outer script greps
at the end.

Mechanism confirmed firing: stdin-eof, in every single cycle observed (5/5,
10/10, and 100/100 across multiple runs). This matches the plan's
documented primary mechanism and is the direct consequence of giving the
child a real pipe instead of an inherited console.

### Fix 3 -- no vacuous pass

- Server never started / PID indeterminate: the outer script polls the PID
  file for up to PidTimeoutMs (default 5000ms); if the PID never appears,
  or appears but the process is already gone by the time it's checked, the
  cycle increments $LaunchFailures, prints a loud Write-Warning, and the
  run exits 1 at the end. It does not silently continue to the next cycle
  as a pass.
- Host could not be killed: taskkill /F /PID $HostProc.Id's $LASTEXITCODE
  is checked; non-zero increments $KillFailures and warns loudly.
- Final sweep: after all cycles, any remaining gobsidian.exe (by name, not
  just tracked PIDs) is counted as a survivor and force-killed, so a host
  that died between the PID-poll and the taskkill can't hide a process from
  the per-cycle count.
- Exit code is 1 if Survivors > 0 OR LaunchFailures > 0 OR KillFailures >
  0; 0 only if all three are zero.
- Found this matters in practice: while regression-testing (see below), a
  run with the mechanism broken hit a second, real bug in the harness
  itself -- $Reasons.Count on a $null pipeline result (Select-String found
  nothing) threw under Set-StrictMode, which would have masked the "0
  orphans" vs "5 orphans" distinction behind a crash. Fixed by wrapping the
  pipeline in @(...) so it's always an array. Verified the fix by
  re-running the broken-mechanism case again: clean exit 1 with "FALHA: 5
  orfao(s) em 5 ciclos" and "nenhum reason= encontrado nos logs de debug",
  no crash.

### Verification

1. `go test -race ./... -v` -- all green (config, doctor, lifecycle
   including TestSameProcessRejectsExited, mcpsrv, service, vault,
   tools/netcheck). TestSignalCancelsContext still SKIPs in this sandboxed
   shell ("sinal nao entregavel nesta plataforma: not supported by
   windows") -- pre-existing, unrelated to this pass.
2. Mutation evidence for Fix 1: see above (revert / fail / restore / pass).
3. Short run: -Cycles 5 and -Cycles 10, both "[OK] Nenhum orfao",
   reason=stdin-eof every time.
4. Full 100-cycle run, literal final output:

```
[...] 100 ciclos de encerramento abrupto (host com pipe real em stdin)
[i] logs e PIDs em C:\Users\jonyd\AppData\Local\Temp\gobsidian_orphan_e4a959ae0a744df782582bb41bf1399e
[i] 10/100
[i] 20/100
[i] 30/100
[i] 40/100
[i] 50/100
[i] 60/100
[i] 70/100
[i] 80/100
[i] 90/100
[i] 100/100
[i] motivos observados nos logs de debug:
    stdin-eof: 100x
[OK] Nenhum orfao em 100 ciclos
```

Exit code 0. Verified with Get-Process -Name gobsidian immediately after:
no results. Work dir was auto-removed by the script (only kept on
failure).

5. Mechanism that fires: stdin-eof, 100/100 in the final run (and in every
   prior run at every cycle count). Parent-watch (Fix 1) and signal never
   got a chance to fire in this harness because stdin-eof -- now that the
   host holds a real pipe -- wins the race by a wide margin (well under
   SettleMs = 2000ms, versus parent-watch's default 5s check interval and
   3-failure threshold).
6. Regression testing -- what was actually broken and re-run end-to-end:
   - stdin-eof: temporarily changed internal/lifecycle/stdin.go's
     `l.trigger("stdin-eof"); return` on io.EOF to a no-op
     `time.Sleep(10ms); continue` (busy-loop guard against a hot spin on a
     closed pipe). Rebuilt, ran -Cycles 5: all 5 survived ("FALHA: 5
     orfao(s) em 5 ciclos"), and no reason= lines appeared in any log --
     exactly the expected signature of "the mechanism that actually
     protects this scenario is gone." This is also what surfaced the
     $Reasons.Count bug described above. Restored stdin.go exactly (git
     diff empty afterward), rebuilt, re-ran -Cycles 10: back to clean,
     stdin-eof 10/10.
   - Fix 1 (parent-watch/sameProcess): regression-tested at the unit level
     only (revert/fail/restore, described above), not end-to-end in the
     orphan harness. Re-running the harness with only Fix 1 reverted would
     not currently show a different result, because stdin-eof (not
     parent-watch) is what saves every cycle in this harness --
     parent-watch never gets to matter within the 2-second settle window
     regardless of whether it's correct or broken. Proving Fix 1
     end-to-end would require also disabling stdin-eof and lengthening
     SettleMs past ~5-6s to give parent-watch's default interval and
     failure threshold room to act; that combined experiment was not run,
     given time constraints. The unit-level mutation is airtight for the
     code-level defect, but the harness's coverage of parent-watch
     specifically remains indirect.
   - signal: not regression-tested at all in this pass, at either level --
     out of scope per the brief's "touch only parent_windows.go and its
     tests" for Fix 1, and the harness doesn't exercise it (no
     GenerateConsoleCtrlEvent, and TestSignalCancelsContext already SKIPs
     unconditionally in this sandbox).
7. `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go
   vet ./...` -- all clean, no output.
8. Added the orphans job to .github/workflows/ci.yml: runs-on:
   windows-latest, builds via scripts/build.ps1, runs
   scripts/test_orphans.ps1 -Cycles 100. Expected to work on a GitHub
   Windows runner. The mechanism this harness actually exercises (stdin-eof
   via a real anonymous pipe created by System.Diagnostics.ProcessStartInfo)
   has no dependency on an interactive console or desktop session -- that's
   precisely why the previous cmd /c-based harness was unfaithful (it
   depended on console inheritance, which behaves differently in
   non-interactive contexts) and why this one should be more portable, not
   less. Spawning redirected-I/O child processes from a script step is a
   completely standard CI pattern. Two residual risks worth flagging for
   whoever watches the first run: (a) first-run antivirus/Defender scanning
   of a freshly-built gobsidian.exe could slow individual cycles without
   affecting correctness; (b) GitHub Actions Windows runners may enclose job
   steps in a Win32 Job Object with kill-on-close semantics -- this
   shouldn't affect the mid-run behavior being tested (job-object cleanup
   fires at job/step teardown, not mid-script), but if the step itself were
   ever torn down mid-run it could kill the "orphan" out from under the test
   in a way that looks like a false pass. No access to an actual
   GitHub-hosted Windows runner to confirm directly.
9. `git status --porcelain` after the final 100-cycle run: clean except the
   intended staged changes (verified before commit). Get-Process -Name
   gobsidian returned no results after every run in this pass, including
   after the two regression-test runs that intentionally left survivors --
   the script's own per-cycle Stop-Process cleanup handled those. Two
   %TEMP%\gobsidian_orphan_* work directories were left behind by the two
   failed regression-test runs (the script preserves the work dir on
   failure, by design, for diagnosis); removed manually after inspection
   since they're outside the repo and outside the scope of anything that
   needs preserving.

### Concerns

- Fix 1 is proven correct at the unit level (mutation-tested) and is
  necessary for the redundancy claim in lifecycle.go's package doc to
  actually hold, but it is not exercised end-to-end by this specific
  harness because stdin-eof wins the race first. A future pass that wants
  full end-to-end coverage of all three mechanisms would need per-mechanism
  harness variants (e.g., disable stdin-eof and extend the settle window to
  isolate parent-watch; use a real console + GenerateConsoleCtrlEvent to
  isolate signal) -- the predecessor already flagged the same gap in their
  report's "Concerns" #2, and it still stands after this pass, just with
  stdin-eof now genuinely covered instead of vacuously "passing" by
  accident.
- CI orphans job is added but unverified against an actual GitHub-hosted
  Windows runner (no access to one from this environment).
