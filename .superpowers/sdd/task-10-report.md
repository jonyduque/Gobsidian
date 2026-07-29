# Task 10 report: `gobsidian doctor`

## What was implemented

- `internal/doctor/doctor.go` — `Status`/`Result` types, `Run`, `ExitCode`, exactly as specified in the brief (no mechanical changes needed).
- `internal/doctor/checks.go` — the eight cross-platform checks: `checkRootExists`, `checkReadable`, `checkWritable`, `checkObsidianDir`, `checkNoteCount`, `checkLongestPath`, `checkCacheDir`, `checkFreeSpace`. Also two unexported shared helpers: `walkVault` (opens a `vault.Vault` and walks it, the single point every vault-walking check goes through) and `longestVaultPath` (shared by `checkLongestPath` and the Windows long-paths check, so both measure the same value instead of duplicating the walk).
- `internal/doctor/checks_windows.go` (`//go:build windows`) — `platformChecks()` returning `checkLongPathsEnabled`, `checkCloudOnlyFiles`, `checkCasingCollisions`; plus `longPathsEnabled()` (reads `HKLM\SYSTEM\CurrentControlSet\Control\FileSystem\LongPathsEnabled` via `golang.org/x/sys/windows/registry`) and `diskFreeBytes()` (via `windows.GetDiskFreeSpaceEx`).
- `internal/doctor/checks_other.go` (`//go:build !windows`) — `platformChecks()` returns `nil`; plus `diskFreeBytes()` via `golang.org/x/sys/unix.Statfs` (compiles on both linux and darwin — `Bsize` differs in type between the two, both cast cleanly to `uint64`).
- `cmd/gobsidian/doctor.go` — replaced the Task 9 placeholder with the brief's cobra wiring verbatim (flags, `ReadOnlySet` propagation, ASCII report to stdout, exit code).
- `internal/doctor/doctor_test.go` — the brief's two tests, unmodified.
- `internal/doctor/doctor_extra_test.go` — four additional tests covering the "checks the brief's tests do not make" list (see below).

`golang.org/x/sys` was already pinned in `go.mod` as `// indirect`; this task makes it a real, direct import (windows and unix subpackages) but I did not run `go mod tidy`, per the constraint. The `// indirect` comment on that line is now stale/cosmetic — it does not affect build, vet, or test correctness, but a future `tidy` run (once all pinned deps have importers) would clean it up.

## TDD evidence

**RED** — before any non-test file existed in the package:

```
$ go test ./internal/doctor/ -v
github.com/jonyd/gobsidian/internal/doctor: no non-test Go files in .../internal/doctor
FAIL	github.com/jonyd/gobsidian/internal/doctor [build failed]
```

(Equivalent to the brief's expected `undefined: doctor.Run` — the package had zero implementation files so the compiler failed one step earlier, at "no Go files", instead of at name resolution. Same root cause: `doctor.Run` did not exist yet.)

**GREEN** — after implementing `doctor.go`, `checks.go`, `checks_windows.go`, `checks_other.go`:

```
$ go test -race ./internal/doctor/ -v
=== RUN   TestRunFlagsMissingVault
--- PASS: TestRunFlagsMissingVault (0.00s)
=== RUN   TestRunWarnsWithoutObsidianDir
--- PASS: TestRunWarnsWithoutObsidianDir (0.01s)
PASS
ok  	github.com/jonyd/gobsidian/internal/doctor	1.731s
```

After adding the four extra tests (`doctor_extra_test.go`):

```
$ go test -race ./internal/doctor/ -v
=== RUN   TestExitCodeEmptyResults
--- PASS: TestExitCodeEmptyResults (0.00s)
=== RUN   TestRunContextCancelledStopsWalk
--- PASS: TestRunContextCancelledStopsWalk (0.01s)
=== RUN   TestCheckWritableNoLeftoverTempFile
--- PASS: TestCheckWritableNoLeftoverTempFile (0.01s)
=== RUN   TestCheckLongestPathMeasuresAbsolutePath
--- PASS: TestCheckLongestPathMeasuresAbsolutePath (0.01s)
=== RUN   TestRunFlagsMissingVault
--- PASS: TestRunFlagsMissingVault (0.00s)
=== RUN   TestRunWarnsWithoutObsidianDir
--- PASS: TestRunWarnsWithoutObsidianDir (0.01s)
PASS
ok  	github.com/jonyd/gobsidian/internal/doctor	1.999s
```

Full suite, vet, gofmt, and cross-platform vet, all clean:

```
$ go test -race ./...
ok  	github.com/jonyd/gobsidian/internal/config
ok  	github.com/jonyd/gobsidian/internal/doctor	2.129s
ok  	github.com/jonyd/gobsidian/internal/lifecycle
ok  	github.com/jonyd/gobsidian/internal/mcpsrv
ok  	github.com/jonyd/gobsidian/internal/service
ok  	github.com/jonyd/gobsidian/internal/vault
ok  	github.com/jonyd/gobsidian/tools/netcheck

$ go vet ./...              # clean
$ gofmt -l .                # clean, no output
$ GOOS=linux go vet ./...   # clean
$ GOOS=darwin go vet ./...  # clean
```

## Manual run against a real vault

Vault: `%TEMP%\gobsidian-demo-vault` — `.obsidian/app.json`, `Welcome.md`, `Daily/2026-07-26.md`, `Projects/Project-A.md`.

```
$ go run ./cmd/gobsidian doctor --vault "C:\Users\jonyd\AppData\Local\Temp\gobsidian-demo-vault"
[OK] raiz do cofre existe
[OK] permissao de leitura
     4 entradas na raiz
[OK] permissao de escrita
[OK] .obsidian presente
[OK] contagem de notas
     3 notas
[OK] comprimento de caminho
     maior caminho: 76 caracteres
[OK] diretorio de cache
     C:\Users\jonyd\AppData\Local\gobsidian\2dfe0d7e7984d168
[OK] espaco em disco
     439228 MB livres
[OK] caminhos longos habilitados
[OK] arquivos somente-nuvem
[OK] colisoes de casing
[OK] Ambiente apto
$ echo $?
0
```

Missing-root case (exit code path):

```
$ go run ./cmd/gobsidian doctor --vault "C:\Users\jonyd\AppData\Local\Temp\gobsidian-demo-vault-does-not-exist"
[!] raiz do cofre existe
     "C:\\Users\\jonyd\\AppData\\Local\\Temp\\gobsidian-demo-vault-does-not-exist": GetFileAttributesEx ...: The system cannot find the file specified.
[!] Ha falhas bloqueantes acima
$ echo $?
1
```

Only the root check ran; every other check was skipped, confirming the "first failure that invalidates everything after it stops the run" behaviour.

`--read-only` flag wiring confirmed against a writable vault (write check still reports `[OK]` since the directory is in fact writable — the Warn-instead-of-Fail branch only shows when the temp-file probe itself fails, which a healthy temp dir won't trigger; verified by code reading instead of by test, since reliably forcing a write failure on a fresh Windows temp dir in CI is not practical).

All demo artifacts (`%TEMP%\gobsidian-demo-vault`, `%TEMP%\gobsidian-ro-vault`, and the cache directories `doctor` created under `%LOCALAPPDATA%\gobsidian`) were deleted after the runs above — nothing left behind.

## The five extra checks

| # | Question | Result |
|---|---|---|
| 1 | Does every check return a stable `Name`? | Yes — every check uses `const name = "..."` literal strings, none built with `fmt.Sprintf`. Verified by code inspection of all 11 check functions. |
| 2 | Does the write-permission check clean up after itself? | Yes. `checkWritable` always attempts `Close` then `Remove` on the probe file it creates, even when `Close` fails. Added `TestCheckWritableNoLeftoverTempFile`, which runs `doctor.Run` against a real directory and lists it afterward — no `gobsidian-doctor` file remains. **PASS.** |
| 3 | Is `ExitCode` right when checks are empty? | Yes — the loop over a nil/empty slice never executes, so it returns 0. Added `TestExitCodeEmptyResults` covering both `nil` and `[]Result{}`. **PASS.** |
| 4 | Does a cancelled context stop the vault-walking checks? | Yes. `vault.Walk` checks `ctx.Err()` on every entry, including the root, so a pre-cancelled context returns `context.Canceled` before visiting any file. Added `TestRunContextCancelledStopsWalk`: 5 note files, context cancelled before `Run`, and `contagem de notas` reports `"varredura interrompida: context canceled"` instead of `"5 notas"`. **PASS — this is the one check that revealed something worth making explicit:** without deliberate handling, a cancelled walk would either hang (if `ctx.Err()` were ignored) or silently report `"0 notas"` (a false "empty vault" warning) instead of correctly flagging that the check didn't finish. I added a distinct `"varredura interrompida"` Warn branch in every vault-walking check specifically so cancellation is never confused with "the vault is actually empty/collision-free/etc." |
| 5 | Does the longest-path check measure the right thing (full absolute path, not vault-relative)? | Yes, by construction: `longestVaultPath` builds `filepath.Join(cfg.VaultPath, relPath)` and measures that. Added `TestCheckLongestPathMeasuresAbsolutePath`: a vault at `t.TempDir()` with one 4-character-named note; the check's `Detail` contains `len(filepath.Join(root, "A.md"))` (tens of characters, from the OS temp path), not `len("A.md")` = 4. **PASS.** |

## Files changed

- `cmd/gobsidian/doctor.go` (replaced placeholder)
- `internal/doctor/doctor.go` (new)
- `internal/doctor/checks.go` (new)
- `internal/doctor/checks_windows.go` (new)
- `internal/doctor/checks_other.go` (new)
- `internal/doctor/doctor_test.go` (new, brief's tests verbatim)
- `internal/doctor/doctor_extra_test.go` (new, four additional tests)

## Self-review findings

- **Completeness:** all 8 common checks + 3 Windows-only checks implemented with the exact `Name` strings from the brief's tables, and the failure/warning conditions match the tables (root: fail only; read: fail only; write: fail unless `--read-only`, then warn; `.obsidian`/note-count/longest-path/cache-dir: warn-only per table; free space: fail <10MB, warn <100MB; the three Windows checks: warn-only per table).
- **Correctness:** no check opens vault file content. `checkReadable` uses `os.ReadDir` (metadata only). `checkObsidianDir`/`checkRootExists` use `os.Stat`. The vault-walking checks (`checkNoteCount`, `checkLongestPath`, `checkCloudOnlyFiles`, `checkCasingCollisions`) all go through `vault.Walk`, which classifies files and calls `vault.IsCloudOnly` (attribute-only, no open) but never opens content. `checkWritable`'s temp file is its own scratch probe, not a vault file being opened for a purpose other than testing writability.
- **Discipline — deviations from the literal brief, and why:**
  1. Every vault-walking check reports a distinct `"varredura interrompida: <err>"` Warn when the walk returns an error (including context cancellation), instead of silently reporting a misleading "zero/none found" result. This doesn't add a new check or change any Fail verdict — it only disambiguates a Warn's meaning. Judged necessary given the brief's own instruction to verify context cancellation propagates correctly.
  2. `checkCacheDir` returns `StatusOK` with a note when `cfg.CacheDir` is empty, instead of attempting `MkdirAll("")`. This only matters for tests that build `config.Config` directly via `config.Defaults()` (bypassing `config.Load`, which always fills a real default); in the real CLI path `config.Load` guarantees a non-empty `CacheDir`, so this branch never fires there. Without the guard, `doctor_test.go`'s `TestRunWarnsWithoutObsidianDir` would have attempted `os.MkdirAll("")` from the package's working directory — a landmine for accidental side effects in the repo, avoided by the guard.
  3. `diskFreeBytes` measurement failures report `StatusWarn` (not `StatusFail`) since the table doesn't define behaviour for "couldn't measure" — treated as informational rather than blocking, consistent with the checks that "never fail" bias in the rest of the table.
  4. Two small shared helpers (`walkVault`, `longestVaultPath`) factor out the vault-opening/walking boilerplate so the Windows long-paths check reuses the exact same measurement as `checkLongestPath` instead of re-walking with different logic. Not requested verbatim by the brief but stays within `checks.go`'s stated scope ("implementa as oito funções acima").
- **Testing:** confirmed pristine after the manual real-vault runs — demo vault directories and the cache directories doctor created under `%LOCALAPPDATA%\gobsidian` were deleted afterward. `go test` leaves nothing behind either (`t.TempDir()` self-cleans, and the added `TestCheckWritableNoLeftoverTempFile` explicitly asserts no doctor-created file remains).

## Mechanical corrections

None — the brief's code for `doctor.go` and `cmd/gobsidian/doctor.go` compiled and matched the described behaviour without modification. The check-table prose (not code) was implemented from scratch as instructed.

## Concerns

- `go.mod`'s `golang.org/x/sys // indirect` comment is now inaccurate (the package is directly imported by `checks_windows.go` and `checks_other.go`), but per the environment constraints I did not run `go mod tidy` or edit `go.mod` by hand. This is cosmetic only — build, vet, and test all pass as-is — but a future task that does run `tidy` (once every pinned dependency has an importer) will pick it up for free.
- The `--read-only`-triggers-a-warning branch (write probe fails **and** `--read-only` is set) is exercised by code reading and by the `TestRunFlagsMissingVault`/`TestRunWarnsWithoutObsidianDir` tests' overall flow, but not by a test that forces the actual write to fail. Forcing a reliable write failure against a real filesystem in a portable way (without relying on OS-specific ACL manipulation) wasn't attempted, since Task 10's scope is the check logic, not filesystem permission fixtures.

## Fix pass — review findings

Four Important findings plus two minor ones, from the Task 10 review. All addressed in `internal/doctor/doctor.go`, `checks.go`, `checks_windows.go`, `checks_other.go`, `doctor_test.go`, and `doctor_extra_test.go`. `internal/vault`, `internal/lifecycle`, `internal/config`, `internal/service`, `internal/mcpsrv` untouched, per the constraint.

### Finding 1 — Fail/Warn markers collapsed to the same string

`Status.Marker()` transcribed verbatim from the plan's fixed section (`docs/superpowers/plans/2026-07-25-gobsidian-v01.md`, "Task 10"): `StatusOK` maps to `[OK]`, `StatusWarn` maps to `[*]`, `StatusFail` maps to `[!]`.

Added `TestStatusMarkerDistinctPerStatus` (`doctor_extra_test.go`), which compares all three `Marker()` values pairwise and pins the exact strings. Proved it bites: reverted the fix locally so `StatusWarn` returned `"[!]"` again (colliding with `StatusFail`), reran the test, it failed with `marcadores devem ser distintos: OK="[OK]" Warn="[!]" Fail="[!]"`. Restored the fix, reran, PASS. No other test in the package previously compared the three marker values against each other; every existing assertion only checked substring/status combos in isolation, which is why the collision was invisible before.

### Finding 2 — Run halted on check name, not on a halting property

Replaced the `res.Name == "raiz do cofre existe"` string comparison in `Run` with a `halting bool` carried per check entry, transcribed from the plan's fixed design. `checkRootExists` and `checkReadable` are the two halting checks; everything else is not. `Run` now returns immediately on the first `StatusFail` from a halting check, restructured into two phases: the halting checks first, then (only if both pass) everything else including the single vault scan from Finding 4.

Regression test: could not make a directory exist-but-be-unreadable in this environment. Tried an explicit deny ACE (`icacls dir /deny "USERNAME:(RD)"`) against a fresh temp directory; `os.ReadDir` and `os.Stat` both still succeeded afterward, since the account carries an inherited Full-Control ACE from the Administrators group/owner that outranks the explicit deny in practice on this machine. Cleaned up the ACL and temp directory afterward. Same limitation a previous task hit with both chmod and icacls.

Fallback per the brief: extended `TestRunFlagsMissingVault` (`doctor_test.go`) to assert the results slice has exactly one entry when the root doesn't exist, proving the halt mechanism stops the run after exactly the one failing check rather than merely asserting a failure is present somewhere in a longer slice.

### Finding 3 — six checks accepted but discarded ctx

`checkRootExists`, `checkReadable`, `checkWritable`, `checkObsidianDir`, `checkCacheDir`, `checkFreeSpace` now all check `ctx.Err()` as their first statement and, if non-nil, return a Warn result with detail `"varredura interrompida: <err>"` without touching the filesystem. Same vocabulary the vault-walking checks already used for cancellation, so a cancelled doctor run reads consistently across every check instead of grinding through blocking syscalls it can no longer act on.

### Finding 4 — the vault was walked up to five times

Added a `vaultScan` struct in `checks.go` holding note count, longest path length and value, cloud-only count, casing collisions, and an error field (non-nil when the scan was interrupted). A new `scanVault(ctx, cfg)` function runs a single walk and populates every field from the same pass over the same entries. `Run` calls `scanVault` exactly once, after the two halting checks pass, and only then.

`checkNoteCount` and `checkLongestPath` (cross-platform) plus `checkLongPathsEnabled`, `checkCloudOnlyFiles`, `checkCasingCollisions` (Windows-only) all changed signature from taking `(context.Context, config.Config)` to taking the shared `vaultScan` value, and now read from it instead of walking independently. `platformChecks` changed signature to accept a `vaultScan` and return `[]Result` directly on both platform files. Checks that need no scan (root exists, writable, free space, cache dir, `.obsidian` presence) were left on their original context/config signature, untouched by the scan.

Measured walk count: added a temporary debug print at the top of the shared `walkVault` helper, built and ran `go run ./cmd/gobsidian doctor` against a demo vault (3 notes plus `.obsidian/`, all 11 checks executing including the 3 Windows-only ones), captured stderr separately from stdout, and counted matching lines: exactly 1. One walk per doctor invocation, confirmed empirically, then the instrumentation was removed before committing (confirmed via `git diff`).

### Minor findings

- `checkCacheDir`: added `TestCheckCacheDirCreatable` (non-empty CacheDir pointing at a creatable nested path, asserts OK and that the directory now exists) and `TestCheckCacheDirUncreatable` (CacheDir nested under a path whose ancestor is a regular file, so MkdirAll fails on every OS; asserts Warn and that it is not a blocking failure). Both in `doctor_extra_test.go`.
- `checkObsidianDir`: now distinguishes `errors.Is(err, fs.ErrNotExist)` (reported as "pasta .obsidian ausente...") from any other stat error (reported as "nao foi possivel verificar ...") and from existing-but-not-a-directory. All three remain Warn (the check never fails), but the detail text no longer conflates "never existed" with "exists but could not be checked."

### Verification

`go test -race -count=1 ./...` is green across all seven packages (config, doctor, lifecycle, mcpsrv, service, vault, tools/netcheck). `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, and `GOOS=darwin go vet ./...` are all clean.

Manual run against a real demo vault (`.obsidian/app.json`, `Welcome.md`, `Daily/2026-07-26.md`, `Projects/Project-A.md`) produced all-OK output with exit 0. A second demo vault without `.obsidian/` showed the `[*]` warn marker rendering distinctly from `[OK]` (detail: "pasta .obsidian ausente..."). A missing-root run showed `[!]` distinctly from both, followed by "Ha falhas bloqueantes acima" and exit 1, with exactly one check result before the run stopped.

All demo vaults and the cache directories doctor created under the local app-data gobsidian folder were deleted after these runs. `git status --porcelain` shows only the six modified `internal/doctor` files, nothing stray. Also found and removed four leftover temp directories matching this same finding's naming pattern from a prior, unrelated attempt at this review item — not created by this pass, but cleaned up since they cluttered the same temp namespace.

### Concerns

None new. The two pre-existing concerns from the original Task 10 report (the stale `// indirect` comment on `golang.org/x/sys` in `go.mod`, and no automated test forcing a real write-permission failure) still stand, unchanged by this pass.
