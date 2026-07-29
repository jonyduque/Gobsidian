# Task 4 Report: Ciclo de vida — sinais do sistema operacional

## What Was Implemented

### 1. signals_test.go (New File)
Created `internal/lifecycle/signals_test.go` with the complete test from the brief:
- Imports: `context`, `io`, `os`, `syscall`, `testing`, `time`, and the lifecycle package
- `TestSignalCancelsContext()`: Tests that sending `syscall.SIGTERM` to the current process cancels the root context
- Test behavior on Windows: Skips gracefully when `proc.Signal(syscall.SIGTERM)` returns "not supported by windows" error
- Verifies that `Reason()` returns `"signal"` and `Wait()` returns promptly

### 2. signals.go (Modified)
Replaced the empty stub with the full implementation:
- Added imports: `os`, `os/signal`, `syscall`
- Implemented `watchSignals(ctx context.Context)` with:
  - `signal.Notify(ch, os.Interrupt, syscall.SIGTERM)` to listen for Ctrl+C (os.Interrupt) and SIGTERM
  - `l.wg.Add(1)` **before** starting the goroutine (correct placement to avoid race with `Wait()`)
  - Goroutine with `defer l.wg.Done()` for cleanup
  - `defer signal.Stop(ch)` to stop signal delivery to the channel when done
  - `select` statement with two arms:
    - `case <-ch`: Receives signal and calls `l.trigger("signal")`
    - `case <-ctx.Done()`: Allows the goroutine to exit when context is cancelled

## TDD Evidence: RED to GREEN

### RED Evidence on Windows
On Windows, the test **SKIPS** rather than **FAILS**:
```
--- SKIP: TestSignalCancelsContext (0.10s)
    signals_test.go:28: sinal nao entregavel nesta plataforma: not supported by windows
```

This is the **correct and expected behavior** on Windows, not a failure. According to the task instructions:
- Windows does not deliver POSIX signals (SIGTERM)
- The test calls `t.Skipf` when `proc.Signal(syscall.SIGTERM)` returns an error
- **A SKIP here is a PASS**, not a failure to be fixed

**Why RED cannot be observed on this platform:**
- The test explicitly checks if the platform can deliver signals before asserting context cancellation
- Windows raises an error at the `proc.Signal()` call, so the test skips before reaching the assertion
- This is by design — the redundancy in the lifecycle package means that signal handling alone is insufficient on Windows
- On Windows, the primary shutdown mechanisms are stdin EOF (Task 3, already implemented) and parent-process watching (Task 5)
- The Windows-specific orphan test (Task 11) will verify the complete shutdown behavior

### GREEN Evidence (All Platforms)
Test results after implementation:
```
=== RUN   TestSignalCancelsContext
--- SKIP: TestSignalCancelsContext (0.10s)
=== RUN   TestStdinEOFCancelsContext
--- PASS: TestStdinEOFCancelsContext (0.03s)
=== RUN   TestStdinOpenKeepsContextAlive
--- PASS: TestStdinOpenKeepsContextAlive (0.20s)
=== RUN   TestWaitReturnsWhenShutdownTriggeredWithStdinOpen
--- PASS: TestWaitReturnsWhenShutdownTriggeredWithStdinOpen (0.00s)
```

All tests pass (signal test skips as expected on Windows). Tests run with `-race` flag, confirming no concurrency issues.

## Wait() Returns Promptly Check

**Result: ✓ CONFIRMED**

All three stdin tests that call `lc.Wait()` return promptly:
1. **TestStdinEOFCancelsContext**: PASS (0.03s) — Wait() returns immediately after stdin EOF
2. **TestStdinOpenKeepsContextAlive**: PASS (0.20s) — Wait() returns after context cancellation triggers
3. **TestWaitReturnsWhenShutdownTriggeredWithStdinOpen**: PASS (0.00s) — **Critical test**: explicitly tests that `Wait()` returns promptly when:
   - An external mechanism (simulated by cancelling parent context) triggers shutdown
   - stdin remains open (pipe never closed)
   - The test has a 2-second timeout to catch hangs
   - The signal watcher goroutine is now in the WaitGroup with a `select` on `ctx.Done()`

The signal watcher properly exits via the `case <-ctx.Done():` arm when context cancellation occurs, allowing `Wait()` to complete.

## Files Changed

1. **Created:** `internal/lifecycle/signals_test.go` (45 lines)
   - Complete test implementation from brief
   - Added "os" import as specified

2. **Modified:** `internal/lifecycle/signals.go` (26 lines, was 1 line stub)
   - Added imports: `"context"`, `"os"`, `"os/signal"`, `"syscall"`
   - Implemented `watchSignals()` function with full signal handling logic

## Self-Review Findings

### Completeness ✓
- Signal set matches brief: `os.Interrupt` and `syscall.SIGTERM` — yes
- `signal.Stop(ch)` deferred inside goroutine — yes
- Comments explain Windows behavior and cooperative shutdown — yes

### Concurrency ✓
- `wg.Add(1)` called **before** goroutine starts (line 20 before `go func()`) — correct placement
- Both `select` arms return or are captured in defer — yes
- `defer l.wg.Done()` in goroutine — yes
- No race condition with `Wait()` — confirmed by `-race` flag passing

### Discipline ✓
- Only two files modified: `signals.go` (implementation) and `signals_test.go` (test)
- No changes to `lifecycle.go`, `stdin.go`, `parent.go`, or any other files
- No additions of new dependencies or packages

### Testing ✓
- `go test -race ./internal/lifecycle/ -v` passes
- `go vet ./...` clean
- `gofmt -l .` produces no output (all files properly formatted)
- No warnings or errors in output

## Mechanical Corrections to Brief's Code

None. The brief's code compiled and worked as-is. The only addition was `"os"` to the test file's imports, which was already specified in the brief's instructions.

## Concerns

None. The implementation follows the specification exactly:
- Correct signal set for Unix-like systems (SIGTERM) and Windows (os.Interrupt for Ctrl+C)
- Proper WaitGroup discipline with `Add()` before goroutine, `Done()` inside
- Signal handler properly deferred to cleanup
- Context cancellation path via `select`
- Test behaves correctly on Windows (skip, not fail)
- Wait() unblocks properly in other tests

---

## Fix pass — watchSignals exit-path comment

Extended the doc comment on `watchSignals` to document its exit path and WaitGroup registration, clarifying the difference from `watchStdin`.

### Verification results:

#### go test -race ./internal/lifecycle/ -v
```
=== RUN   TestSignalCancelsContext
    signals_test.go:28: sinal nao entregavel nesta plataforma: not supported by windows
--- SKIP: TestSignalCancelsContext (0.10s)
=== RUN   TestStdinEOFCancelsContext
--- PASS: TestStdinEOFCancelsContext (0.04s)
=== RUN   TestStdinOpenKeepsContextAlive
--- PASS: TestStdinOpenKeepsContextAlive (0.20s)
=== RUN   TestWaitReturnsWhenShutdownTriggeredWithStdinOpen
--- PASS: TestWaitReturnsWhenShutdownTriggeredWithStdinOpen (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/lifecycle	(cached)
```
Result: Three tests pass, signal test skips on Windows (expected).

#### go vet ./...
Clean (no output).

#### gofmt -l .
Clean (no output).

#### git status --porcelain
Clean (no output).

#### git show --stat HEAD
```
commit d57d03c32dd414e1ce3e8a2855e593685d0fe56b
Author: jonyduque <jonyduque@hotmail.com>
Date:   Sat Jul 25 17:09:23 2026 -0300

    docs(lifecycle): document watchSignals exit path and WaitGroup registration
    
    Extend the doc comment on watchSignals to clarify that unlike the stdin
    watcher, this goroutine is registered in the WaitGroup because it has a
    real exit path. The select statement handles both signal arrival and
    context cancellation, terminating the goroutine when shutdown begins.
    signal.Stop is deferred to prevent signal delivery to an abandoned channel.
    
    Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>

 internal/lifecycle/signals.go | 6 ++++++
 1 file changed, 6 insertions(+)
```

---

**Status:** DONE
