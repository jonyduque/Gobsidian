# Task 3 Report — Ciclo de vida: EOF em stdin

## What was implemented

- `internal/lifecycle/lifecycle.go` — `Options`, `Lifecycle`, `New`, `trigger`, `Reason`, `Wait`, `ParentPID`, package doc comment. Verbatim from the brief.
- `internal/lifecycle/stdin.go` — `watchStdin`, the goroutine that reads and discards from an `io.Reader` until any read error, then calls `trigger("stdin-eof")` for `io.EOF` or `trigger("stdin-error")` for anything else. Verbatim from the brief.
- `internal/lifecycle/signals.go` — no-op stub for `watchSignals`, per Step 4, to be replaced in Task 4.
- `internal/lifecycle/parent.go` — no-op stub for `watchParent`, per Step 4, marked for removal in Task 5.
- `internal/lifecycle/stdin_test.go` — the two tests from the brief, verbatim.

No mechanical corrections were needed — the brief's code compiled and passed as written.

## TDD Evidence

### RED

Command:
```
go test ./internal/lifecycle/ -v
```

Output (before `lifecycle.go`/`stdin.go`/stubs existed, only `stdin_test.go` present):
```
github.com/jonyd/gobsidian/internal/lifecycle: no non-test Go files in C:\Users\jonyd\Projetos\Gobsidian\internal\lifecycle
FAIL	github.com/jonyd/gobsidian/internal/lifecycle [build failed]
FAIL
```

Expected failure: the brief predicted `undefined: lifecycle.New`; the actual message differs slightly (`no non-test Go files`) because at this point the package had zero non-test files at all — `lifecycle.New` couldn't even be referenced yet. This is the same underlying cause (the implementation doesn't exist), so the RED state is confirmed for the expected reason. Had I stubbed an empty `lifecycle.go` with just `package lifecycle` before writing the implementation, the error would have matched `undefined: lifecycle.New` exactly. I judged this an immaterial variance in a build-failure message, not a correction to the brief's code.

### GREEN

Command:
```
go test -race ./internal/lifecycle/ -v
```

Output:
```
=== RUN   TestStdinEOFCancelsContext
--- PASS: TestStdinEOFCancelsContext (0.03s)
=== RUN   TestStdinOpenKeepsContextAlive
--- PASS: TestStdinOpenKeepsContextAlive (0.20s)
PASS
ok  	github.com/jonyd/gobsidian/internal/lifecycle	2.209s
```

Ran three additional times with `-count=1` to check for flakiness — all three passed cleanly (2.184s, 2.234s, 2.160s), no race warnings.

`go vet ./...` — clean, no output.
`gofmt -l .` — clean, no output (no files need formatting).
`go build ./...` — succeeds.

## Goroutine-exit verification (per task instructions)

- **`lc.Wait()` promptness:** confirmed in both tests — `TestStdinEOFCancelsContext` completes in 0.03s, `TestStdinOpenKeepsContextAlive` in 0.20s (dominated by the intentional 200ms open-stdin wait), well under the 2s failure timeouts. Neither test hangs on `Wait()`.
- **`TestStdinOpenKeepsContextAlive` pipe unblocking:** `io.Pipe`'s `Read` on the reader side is specifically designed to unblock and return `io.EOF` once the writer is closed (`pw.Close()`). This is exactly what happens here: the `watchStdin` goroutine is parked in `pr.Read(buf)` during the first 200ms window (stdin "open"), and when the test closes `pw`, that blocked `Read` returns immediately with `io.EOF`, the goroutine calls `trigger("stdin-eof")` and returns, `wg.Done()` fires, and `lc.Wait()` returns promptly. No leak in this test. `-race` across four runs found nothing.

## Concern to flag: blocking `Read` is not unwound by context cancellation

`watchStdin`'s goroutine never selects on `ctx.Done()` — the `ctx` parameter is accepted but unused inside the loop. Its only exit path is a read error on the given `io.Reader`. This is correct and sufficient for Task 3 in isolation, because stdin is the only mechanism that can trigger `l.cancel()` right now (signals/parent are no-op stubs).

But it is a real forward-looking risk once Task 4 (signals) and Task 5 (parent-watch) are wired in: if either of those fires first and cancels the root context while stdin never produces EOF/error (e.g., the host sends SIGTERM but leaves the child's stdin handle open, or the parent-watch detects parent death first), `watchStdin`'s goroutine stays parked in `r.Read(buf)` indefinitely — a blocking OS read is not interruptible by `context.Context` cancellation in the general case. `lc.Wait()` would then block forever in that scenario, which is exactly the leak this package exists to prevent.

This isn't a bug in Task 3's code — it matches the brief exactly and both tests pass because stdin is the only active trigger. I'm flagging it per the task instructions rather than papering over it, since it's a genuine open design question for Task 4/5 integration: either `serve` must guarantee stdin is always eventually closed/erroring when the process should exit (which the brief's own note about `io.TeeReader` + `IOTransport` in Task 9 suggests is the intended answer — the host closing stdin remains the terminal event either way), or `Wait()`'s contract needs to explicitly document that it does not guarantee prompt return once other trigger mechanisms exist, or `watchStdin` needs a bounded/cancelable read strategy. Not addressed here — out of scope for Task 3 per the brief, but worth surfacing before Task 4/5 land.

## Files changed

- `C:\Users\jonyd\Projetos\Gobsidian\internal\lifecycle\lifecycle.go` (new)
- `C:\Users\jonyd\Projetos\Gobsidian\internal\lifecycle\stdin.go` (new)
- `C:\Users\jonyd\Projetos\Gobsidian\internal\lifecycle\signals.go` (new, stub)
- `C:\Users\jonyd\Projetos\Gobsidian\internal\lifecycle\parent.go` (new, stub)
- `C:\Users\jonyd\Projetos\Gobsidian\internal\lifecycle\stdin_test.go` (new)

## Self-review

- **Completeness:** every symbol from the brief's interface list is present: `Options{Stdin, ParentPID, ParentCheckInterval, Logger}`, `New(ctx, opts) (context.Context, *Lifecycle)`, `(*Lifecycle).Wait()`, `(*Lifecycle).Reason() string`. `trigger` and `ParentPID()` (package-level) also present per Code Organization list.
- **Concurrency:** only one goroutine is started in this task's active code path (`watchStdin`'s). `l.wg.Add(1)` is called synchronously before `go func()`, not inside it. The goroutine has a single exit path: any read error triggers cancellation and returns. The two stubs (`watchSignals`, `watchParent`) start no goroutines, so no leak surface from them yet.
- **Discipline:** implemented only what the brief specified — no extra helpers, no speculative Task 4/5 logic, no files beyond the five named in Code Organization. Package doc comment matches the brief.
- **Testing:** output is pristine — no stray warnings from `go test`, `go vet`, or `gofmt`. The only warnings seen anywhere were Git's own LF→CRLF line-ending notices during `git add`/`git commit`, which are unrelated to test output and harmless (this repo has no `.gitattributes` forcing LF; not addressed since it's outside this task's scope and doesn't affect correctness).

## Mechanical corrections to the brief

None. The brief's code compiled and ran as written, byte-for-byte.

## Commit

`930eaa1` — `feat(lifecycle): cancel root context on stdin EOF`
5 files changed, 184 insertions(+)

## Fix pass — stdin goroutine not waited on

### The defect (recap)

`watchStdin` registered its goroutine with `l.wg.Add(1)` / `defer l.wg.Done()`. Its only exit path is a read error on stdin, and a blocking `Read` on an arbitrary `io.Reader` cannot be interrupted by context cancellation. Today stdin is the only trigger, so it always exits and the concern flagged at the end of Task 3 was latent. The moment Task 4 (signals) or Task 5 (parent-watch) lands, a shutdown triggered by either of them would leave stdin open and this goroutine parked in `Read` forever — and because it was in the `WaitGroup`, `Wait()` would hang, which is functionally the same orphaned-process failure this package exists to prevent.

### The fix

- `internal/lifecycle/stdin.go` — removed `l.wg.Add(1)` and `defer l.wg.Done()` from `watchStdin`. The goroutine is no longer registered with the `WaitGroup`. Added a comment on `watchStdin` explaining it is deliberately not waited on, that a blocking `Read` cannot be interrupted by context cancellation, and that including it would make `Wait()` hang whenever another mechanism triggers shutdown first. Everything else about `watchStdin` (loop, discard, `trigger("stdin-eof")` on `io.EOF`, `trigger("stdin-error")` on any other error) is unchanged.
- `internal/lifecycle/lifecycle.go` — added a comment on `Wait` stating what it does and does not cover: it waits for the goroutines that can be unwound (signal handler, parent watcher), and deliberately not for the stdin reader.
- `internal/lifecycle/stdin_test.go` — added `TestWaitReturnsWhenShutdownTriggeredWithStdinOpen`, which captures the scenario Task 4 would otherwise break: shutdown triggered while stdin stays open, and `Wait()` must still return. It opens an `io.Pipe` that is never closed during the test, creates the `Lifecycle` with a cancelable parent context, calls `cancelParent()` to simulate an external trigger (standing in for signals/parent-watch, which are still stubs), waits for `ctx.Done()`, then runs `lc.Wait()` in a goroutine and selects on a `done` channel against `time.After(2*time.Second)`, calling `t.Fatal` if the timeout wins. No `io.Closer` was added to `Options`, no `select` on `ctx.Done()` was added around the `Read`, and `signals.go`/`parent.go` were not touched.

### Verification

Command:
```
go test -race -count=1 ./internal/lifecycle/ -v
```

Output (after the fix):
```
=== RUN   TestStdinEOFCancelsContext
--- PASS: TestStdinEOFCancelsContext (0.03s)
=== RUN   TestStdinOpenKeepsContextAlive
--- PASS: TestStdinOpenKeepsContextAlive (0.20s)
=== RUN   TestWaitReturnsWhenShutdownTriggeredWithStdinOpen
--- PASS: TestWaitReturnsWhenShutdownTriggeredWithStdinOpen (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/lifecycle	2.232s
```

### Mutation evidence — the new test actually bites

Temporarily restored `l.wg.Add(1)` / `defer l.wg.Done()` in `watchStdin` (everything else unchanged) and re-ran just the new test with an explicit outer timeout:

Command:
```
go test -race ./internal/lifecycle/ -v -run TestWaitReturnsWhenShutdownTriggeredWithStdinOpen -timeout 15s
```

Output (with `wg.Add`/`wg.Done` restored — regression state):
```
=== RUN   TestWaitReturnsWhenShutdownTriggeredWithStdinOpen
    stdin_test.go:87: Wait() nao retornou com stdin ainda aberto apos desligamento externo; a goroutine de watchStdin nao deveria estar no WaitGroup
--- FAIL: TestWaitReturnsWhenShutdownTriggeredWithStdinOpen (2.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/lifecycle	3.218s
FAIL
```

The test fails via the explicit `t.Fatal` on the 2s `time.After` path (not a binary-wide hang/timeout), confirming the assertion bites correctly on regression.

Reverted `watchStdin` back to the fixed version (no `wg.Add`/`wg.Done`) and re-ran the full suite — green again:

Command:
```
go test -race -count=1 ./internal/lifecycle/ -v
```

Output:
```
=== RUN   TestStdinEOFCancelsContext
--- PASS: TestStdinEOFCancelsContext (0.03s)
=== RUN   TestStdinOpenKeepsContextAlive
--- PASS: TestStdinOpenKeepsContextAlive (0.20s)
=== RUN   TestWaitReturnsWhenShutdownTriggeredWithStdinOpen
--- PASS: TestWaitReturnsWhenShutdownTriggeredWithStdinOpen (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/lifecycle	2.232s
```

### Static checks

Command:
```
go vet ./...
```
Output: clean, no output.

Command:
```
gofmt -l .
```
Output: clean, no output (no files need formatting).

Command:
```
git status --porcelain
```
Output (before commit):
```
 M internal/lifecycle/lifecycle.go
 M internal/lifecycle/stdin.go
 M internal/lifecycle/stdin_test.go
```
Only the three intended files touched — no stray files.

### Files changed

- `C:\Users\jonyd\Projetos\Gobsidian\internal\lifecycle\stdin.go` — removed `wg.Add`/`wg.Done`, added explanatory comment.
- `C:\Users\jonyd\Projetos\Gobsidian\internal\lifecycle\lifecycle.go` — added explanatory comment on `Wait`.
- `C:\Users\jonyd\Projetos\Gobsidian\internal\lifecycle\stdin_test.go` — added `TestWaitReturnsWhenShutdownTriggeredWithStdinOpen`.

### Commit

`7882017` — `fix(lifecycle): don't wait on the unkillable stdin goroutine`
3 files changed, 57 insertions(+), 4 deletions(-)
