# Test Coverage Fix - HighDateTime Coverage

## Fix pass 2 — HighDateTime coverage

### Initial test with correct implementation

Command:
```
go test -race ./internal/lifecycle/ -v
```

Output:
```
=== RUN   TestSameProcessRejectsRecycledPID
--- PASS: TestSameProcessRejectsRecycledPID (0.00s)
=== RUN   TestParentGoneCancelsContext
--- PASS: TestParentGoneCancelsContext (0.22s)
=== RUN   TestLiveParentKeepsContextAlive
--- PASS: TestLiveParentKeepsContextAlive (0.50s)
=== RUN   TestSignalCancelsContext
    signals_test.go:28: sinal nao entregavel nesta plataforma: not supported by windows
--- SKIP: TestSignalCancelsContext (0.10s)
=== RUN   TestStdinEOFCancelsContext
--- PASS: TestStdinEOFCancelsContext (0.00s)
=== RUN   TestStdinOpenKeepsContextAlive
--- PASS: TestStdinOpenKeepsContextAlive (0.20s)
=== RUN   TestWaitReturnsWhenShutdownTriggeredWithStdinOpen
--- PASS: TestWaitReturnsWhenShutdownTriggeredWithStdinOpen (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/lifecycle	3.075s
```

### Mutation test (weakened sameProcess)

Temporarily removed HighDateTime comparison from sameProcess function.

Command:
```
go test -race ./internal/lifecycle/ -run TestSameProcessRejectsRecycledPID -v
```

Output (shows failure on the new HighDateTime case specifically):
```
=== RUN   TestSameProcessRejectsRecycledPID
    parent_identity_windows_test.go:33: divergencia apenas em HighDateTime aceita como mesmo processo
--- FAIL: TestSameProcessRejectsRecycledPID (0.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/lifecycle	1.207s
FAIL
```

The mutation test fails **only on the new HighDateTime case** (line 33), proving the new test case catches the regression.

### Restoration verification

Command:
```
git diff internal/lifecycle/parent_windows.go
```

Output:
```
(no output - file is clean)
```

Confirmed: parent_windows.go was restored exactly after mutation testing.

### Post-fix validation

Command:
```
go test -race ./internal/lifecycle/ -run TestSameProcessRejectsRecycledPID -v
```

Output (restored implementation passes):
```
=== RUN   TestSameProcessRejectsRecycledPID
--- PASS: TestSameProcessRejectsRecycledPID (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/lifecycle	2.289s
```

### Code quality checks

Command: `go vet ./...`
Output: `go vet passed`

Command: `gofmt -l .`
Output: (no output - all files properly formatted)

### Commit verification

Command: `git show --stat HEAD`
Output:
```
commit 150fd38a0b8388e87abcdbce6bf88c83091a5855
Author: jonyduque <jonyduque@hotmail.com>
Date:   Sat Jul 25 17:41:28 2026 -0300

    test(lifecycle): add HighDateTime test case to catch recycled PID regression
    
    Add a test case that verifies sameProcess rejects processes differing only
    in HighDateTime. This catches implementations that incorrectly drop the
    HighDateTime conjunct, which would fail to detect recycled PIDs born more
    than seven minutes apart.
    
    Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>

 internal/lifecycle/parent_identity_windows_test.go | 9 +++++++++
 1 file changed, 9 insertions(+)
```

Exactly one file changed as required.
