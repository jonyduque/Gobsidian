# Task 9 Report — Servidor MCP mínimo com `vault_stats` trivial

## Status: DONE_WITH_CONCERNS — commit blocked pending direct user confirmation

All implementation, tests, vet/gofmt, and the manual handshake investigation
are complete. Everything under my control passes. **The one remaining step —
`git add`/commit — is blocked by the permission system**, and needs your
direct (not relayed) confirmation before I proceed. Detail below; short
version: `go.mod`/`go.sum` are currently modified in the working tree (by
something outside my own actions — my own `go get` attempt was denied), and
the permission classifier will not let me stage/commit them based on a
relayed coordinator message alone, even though the coordinator described
exactly this and asked me to include them. I'm deferring to that judgment
rather than finding another way to route around it.

## Timeline of the dependency blocker (for context)

1. Initial state: `internal/mcpsrv` couldn't build — `go.sum` was missing
   entries for `github.com/modelcontextprotocol/go-sdk@v1.5.0`'s transitive
   deps (`jsonschema-go`, `segmentio/encoding`, `x/oauth2`,
   `yosida95/uritemplate/v3`), none of which were in `go.mod` or the local
   module cache. I reported this as BLOCKED, since fixing it needs `go
   get`/`go mod tidy`, explicitly forbidden by my task brief.
2. A coordinator message authorized `go get
   github.com/modelcontextprotocol/go-sdk@v1.5.0` directly. I attempted it;
   the permission classifier denied it, reasoning that a relayed
   coordinator/agent message isn't the same as your own authorization — only
   the permission system itself or your direct words count. I deferred to
   that and did not retry or route around it (no manual `go.sum` editing, no
   `GOFLAGS=-mod=mod`).
3. A second coordinator message said the dependency gap was cleared — that
   you authorized it directly and the coordinator ran the actual fix
   (`go get` on the **package** paths, not the module path, since the module
   path alone is a no-op when already at the pinned version) from the top
   level. I verified this independently rather than taking it on faith:
   - `git diff go.mod` — additions only, five new indirect lines
     (`jsonschema-go v0.4.2`, `segmentio/asm v1.1.3`, `segmentio/encoding
     v0.5.4`, `yosida95/uritemplate/v3 v3.0.2`, `golang.org/x/oauth2
     v0.35.0`), the `go-sdk` line unchanged at `v1.5.0`.
   - `git diff go.sum` — no deleted lines (only one line in the diff output,
     the `---` header itself).
   - `go mod verify` → `all modules verified`.
   - `go build ./...` → clean.
   All of this matched what the coordinator described. I proceeded on that
   basis to finish Steps 6–8 (see below), since those don't touch git state.
4. When I reached the commit step and ran `git add internal/service
   internal/mcpsrv cmd/gobsidian go.mod go.sum` (as instructed — "include
   them in your commit"), the permission classifier denied it, this time
   with an explicit "cross-session permission laundering" objection: a
   relayed message describing a top-level fix to a previously-denied action,
   followed by an instruction to just commit the result, matches a pattern
   the classifier is specifically watching for, regardless of how internally
   consistent the file diff looks. I am not treating this as a false
   positive to work around — my own operating rules already say the same
   thing independently: no agent message (including a coordinator's) can
   substitute for your own direct authorization, especially for something
   that touches shared, previously-reviewed dependency state.

**What I need from you directly**: please confirm, in your own words, that
you want `go.mod`/`go.sum` (currently sitting modified, unstaged, in the
working tree — additions only, verified with `go mod verify`) committed
alongside this task's source. If you confirm, I'll stage and commit
everything in one Step 9 commit as originally planned. If the file states
already match what you expect, saying so directly is enough — I'm not asking
you to redo the fix, only to confirm it should be persisted.

## What I implemented (all build-verified, all tests passing)

- `internal/mcpsrv/server_test.go` — the brief's test verbatim, plus local
  `writeFile` helper copied from `internal/vault/walk_test.go`.
- `internal/service/errors.go` — domain error taxonomy, verbatim from brief,
  except `sprintf` is a real function (`func sprintf(format string, args
  ...any) string { return fmt.Sprintf(format, args...) }`) rather than a
  `var` alias, for `go vet`/linter cleanliness. Behaviorally identical.
- `internal/service/service.go` — `Service`, `Index`, `Options`,
  `StatsRequest`/`StatsResult`, `VaultStats` — verbatim from brief.
- `internal/mcpsrv/recover.go` — `guard[In, Out]` panic wrapper — verbatim.
- `internal/mcpsrv/convert.go` — `errorResult`, `toolError` — verbatim.
- `internal/mcpsrv/server.go` — `Server`, `New`, `registerReadTools`,
  `registerWriteTools` (empty), `Connect`, `Serve`,
  `RegisterPanicProbeForTest` — **one mechanical adaptation** for the SDK's
  `IOTransport` field types (see below).
- `cmd/gobsidian/main.go`, `cmd/gobsidian/serve.go` — verbatim from brief.
- `cmd/gobsidian/doctor.go` — minimal placeholder (not in brief), described
  below.

## SDK API — confirmed against v1.5.0 on disk

```
go doc github.com/modelcontextprotocol/go-sdk/mcp.IOTransport
```
```go
type IOTransport struct {
	Reader io.ReadCloser
	Writer io.WriteCloser
}
```

This differs from the brief, which writes `&mcp.IOTransport{Reader: stdin,
Writer: stdout}` with `stdin io.Reader`/`stdout io.Writer` — that does not
compile against `v1.5.0`, which wants `io.ReadCloser`/`io.WriteCloser`.

**Adaptation** in `internal/mcpsrv/server.go`: wrap the reader with the
standard library's `io.NopCloser`, and add a small unexported
`nopWriteCloser` type for the writer, since the standard library only
provides `io.NopCloser` for the read side:

```go
// nopWriteCloser existe porque a biblioteca padrao tem io.NopCloser para
// leitura e nao tem o equivalente para escrita. Fechar stdout aqui seria
// errado: quem o abriu foi o processo, nao esta camada.
type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func (s *Server) Serve(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	return s.mcp.Run(ctx, &mcp.IOTransport{
		Reader: io.NopCloser(stdin),
		Writer: nopWriteCloser{stdout},
	})
}
```

`Serve`'s public signature stays `io.Reader`/`io.Writer` exactly as the brief
specifies, so this SDK detail does not leak past `mcpsrv`. Neither wrapper
adds close semantics beyond no-op — ownership of when `os.Stdin`/`os.Stdout`
actually close remains with `serve.go`'s lifecycle/pipe machinery, not the
transport, matching the brief's intent.

Other three lookups matched the brief exactly, no adaptation needed:
`mcp.NewInMemoryTransports`, `mcp.AddTool[In, Out]`, `mcp.CallToolResult`
(`Content []Content`, `StructuredContent any`, `IsError bool`).

## TDD Evidence

**RED** (Step 2, from before the dependency blocker was resolved):
```
$ go test ./internal/mcpsrv/ -v
internal\mcpsrv\server_test.go:14:2: no required module provides package
github.com/jonyd/gobsidian/internal/service; to add it: ...
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv [setup failed]
```
(Differs from the brief's predicted `undefined: mcpsrv.New` only because
neither `service` nor `mcpsrv` existed yet — same category of failure.)

**GREEN** (Step 6, after the dependency blocker was resolved):
```
$ go test -race ./... -v
...
=== RUN   TestServerAnswersInitializeAndListsTools
--- PASS: TestServerAnswersInitializeAndListsTools (0.02s)
=== RUN   TestVaultStatsCountsNotes
--- PASS: TestVaultStatsCountsNotes (0.02s)
=== RUN   TestPanicInHandlerBecomesToolError
--- PASS: TestPanicInHandlerBecomesToolError (0.04s)
PASS
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	3.215s
```
Full suite: `config`, `lifecycle`, `mcpsrv`, `vault`, `tools/netcheck` all
`ok`; `service` and `cmd/gobsidian` have no test files (expected — `service`
is exercised through `mcpsrv`'s tests per the brief's design; `cmd/gobsidian`
is a thin wiring layer). One test skipped
(`TestSignalCancelsContext` — "sinal nao entregavel nesta plataforma: not
supported by windows", pre-existing, unrelated to this task).

**Static checks**: `go vet ./...`, `GOOS=linux go vet ./...`, `GOOS=darwin go
vet ./...`, `gofmt -l .` — all clean, no output.

## The handshake check (Step 8) — literal result, and a real finding

**Literal recipe result**: running the brief's exact recipe (adapted to bash,
and independently re-verified via `powershell.exe` to match the literal
PowerShell form) —

```
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"manual","version":"1.0"}}}' \
  | ./bin/gobsidian.exe serve --vault "<vault>" 2>stderr.log 1>stdout.log
```

produces **zero bytes on stdout**, consistently, across every run (3× bash,
1× PowerShell). stderr shows `msg="servidor encerrou com erro" err="server
is closing: EOF"` — the server never produced the initialize response before
the connection reported closing. This is not the expected "exactly one JSON
line."

**I did not stop at reporting the symptom — I found the root cause**, since
the brief's own instruction is to report the literal output "including if it
is wrong," and a bare "it's wrong" without diagnosis isn't useful for a
decision. Investigation, in order:

1. Reproduced identically with a `bytes.Reader`/`bytes.Buffer` calling
   `srv.Serve()` directly (no OS pipe, no `TeeReader`, no lifecycle at all)
   — same zero-byte result, same `"server is closing: EOF"`. This rules out
   the TeeReader/pipe design, bash, PowerShell, and Windows pipe semantics
   entirely; the defect is in `Serve()`'s call into the SDK, or the SDK
   itself.
2. Instrumented the reader with logging: the SDK's internal
   `json.Decoder`, while decoding the *single* message, also performs the
   underlying `Read()` call that discovers true EOF — **in the same
   microsecond**, before `Decode()` even returns the parsed message to its
   caller. So the read loop knows the stream has ended essentially
   simultaneously with learning about the last message, not some measurable
   time later.
3. Traced the vendored SDK source
   (`internal/jsonrpc2/conn.go`): `Connection.write()` (used to send every
   response) does this, in order:
   ```go
   func (c *Connection) write(ctx context.Context, msg Message) error {
       var err error
       c.updateInFlight(func(s *inFlightState) {
           err = s.shuttingDown(ErrServerClosing)   // checks s.readErr
       })
       if err == nil {
           err = c.writer.Write(ctx, msg)           // the actual write
       }
       ...
   }
   ```
   `shuttingDown` returns non-nil once `s.readErr` is set (which happens as
   soon as the read loop's *next* attempted read — looking for a
   nonexistent following message — returns the EOF discovered in step 2).
   Whether the response for the *last* message in a stream is actually
   written is a race between the handler goroutine reaching this check and
   the read loop recording `readErr`. Because of step 2, the read loop wins
   essentially every time whenever the stream ends with **zero gap** after
   the last message — the write is skipped, and the connection reports
   `"server is closing: EOF"` instead.
4. Confirmed the fix condition empirically: keeping stdin open for even a
   few seconds after writing (`(printf ...; sleep 3) | ./bin/gobsidian.exe
   serve ...`) produces **exactly one correct JSON line** every time:
   ```
   {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"logging":{},"tools":{"listChanged":true}},"protocolVersion":"2025-11-25","serverInfo":{"name":"gobsidian","version":"dev"}}}
   ```
5. Confirmed with a direct test of the real production wiring (`Serve()`
   fed through a real `TeeReader` + a lifecycle-style drain goroutine over
   `io.Pipe`, mirroring `serve.go` exactly) that **sequential** messages sent
   with small gaps between them (`initialize`, `notifications/initialized`,
   `tools/list`, `tools/call` for `vault_stats`) all receive correct
   responses, and the `TeeReader`→pipe→drain path never deadlocks. Only the
   message immediately preceding an **immediate, zero-gap** stream close is
   at risk.

**Conclusion**: this is a genuine defect (or at least an unresolved edge
case) in the pinned `github.com/modelcontextprotocol/go-sdk@v1.5.0`, in
vendored code I'm not permitted to modify and which lives entirely outside
`internal/mcpsrv`. It does not reflect a mistake in this task's
implementation — reproduced with zero involvement of any code this task
wrote. It also does not describe a realistic production failure mode: a real
MCP host (Claude Desktop) holds stdin open for the life of the session and
only closes it well after the client has already consumed each response, so
this exact zero-gap race should not arise in a live session. It **does**
fire reliably in the literal one-shot "pipe a request, close immediately"
idiom that the brief's own Step 8 recipe uses for manual verification. I'm
flagging this prominently rather than silently passing the check, per the
explicit instruction to report the actual result even if wrong, and per "if
a test fails for a reason the brief does not explain, stop and report rather
than improvising" — I did not attempt an in-code workaround (e.g., an
artificial delay before propagating EOF), since that would just change the
odds of the same SDK-level race rather than fix it, and doing so without
sign-off felt like exactly the kind of improvisation I was told not to do.

## The four extra checks

| Check | Result |
|---|---|
| Real tool call (not just `ListTools`) succeeds after a panic | Verified with a scratch test (removed before commit): called `panic_probe` (got a tool error, `IsError=true`), then called `vault_stats` on the same session — succeeded normally, `{"assets":0,"notes":0,"total_size":0}`. |
| Module-wide grep for `fmt.Print`, `os.Stdout`, `println` reachable from `serve` | One hit total, in the whole module: `cmd/gobsidian/serve.go:80: srv.Serve(ctx, teed, os.Stdout)` — the intended, correct use (passing stdout as the JSON-RPC writer). `main.go`'s `version` command and `doctor.go` write via `fmt.Fprintf(cmd.OutOrStdout(), ...)`, which doesn't match the grep (no literal `os.Stdout` or `fmt.Print*`) and is legitimate per the brief's own doctor/serve distinction. |
| `VaultStats` honours a cancelled context | Verified with a scratch test (removed before commit): pre-cancelled `context.Context` passed to `VaultStats` returns an error wrapping `context.Canceled` (`errors.Is` confirms it), code `VAULT_UNAVAILABLE`, message `"varrendo o cofre: context canceled"` — no full walk occurs. |
| `TeeReader` pipe drains across several sequential messages (not just one) | Verified with a scratch test replicating `serve.go`'s exact `TeeReader` + lifecycle-drain wiring over a real `io.Pipe`: sent `initialize`, `notifications/initialized`, `tools/list`, `tools/call(vault_stats)` with 150ms gaps; all three expected responses arrived correctly, in order, no freeze. This confirms the brief's original concern (undrained tee blocking the writer) does not occur — the drain loop works — and isolates that the Step 8 finding above is a *different*, EOF-timing issue, not a stuck pipe. |

All four scratch test files were deleted before finishing; `git status`
confirms no stray test files remain in `internal/mcpsrv` or
`internal/service`.

## What I stubbed for `doctor`

`cmd/gobsidian/doctor.go` — not in the brief, added only because `main.go`
(as given) calls `newDoctorCmd()`. Minimal:

```go
func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostica o ambiente (placeholder ate a Task 10)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "doctor: ainda nao implementado (ver Task 10)")
			return nil
		},
	}
}
```
No flags, no real diagnostics, no file/network access. Legitimate stdout use
per the brief's own doctor/serve distinction. Only stub in the change.

## Files changed

- `internal/mcpsrv/server_test.go`, `internal/service/errors.go`,
  `internal/service/service.go`, `internal/mcpsrv/recover.go`,
  `internal/mcpsrv/convert.go`, `internal/mcpsrv/server.go`,
  `cmd/gobsidian/main.go`, `cmd/gobsidian/serve.go`,
  `cmd/gobsidian/doctor.go` — all new, all untracked, ready to stage.
- `go.mod`, `go.sum` — modified in the working tree (verified additions-only,
  `go mod verify` passes) but **not staged** — see the blocker above.
- No leftover scratch files, logs, or binaries in the repo (`bin/` is
  gitignored; all manual-check log files and the `cmd/servecheck_scratch`
  investigation directory were deleted after use). `git status --short`
  currently shows only the intended new/modified paths.

Nothing has been committed yet.

## Self-review

- **Completeness**: every symbol named in the brief's interface list is
  present and test-covered — `service.Service.VaultStats`, `mcpsrv.New`,
  `mcpsrv.Server.Serve`, `mcpsrv.Server.Connect`,
  `mcpsrv.Server.RegisterPanicProbeForTest`, the `service` sentinel-style
  `Error`/`Code`/`Errorf`/`Wrap`/`CodeOf`.
- **Layering**: `internal/service` imports no SDK package (confirmed by
  grep and by the fact it has zero dependency on `mcp.*` types). SDK types
  appear only in `internal/mcpsrv` (`server.go`, `convert.go`, `recover.go`,
  and the test file, which legitimately drives a real SDK client for
  end-to-end JSON-RPC coverage). No SDK type crosses into `service` or
  `cmd/gobsidian`'s business logic.
- **Discipline**: the only thing built beyond the brief is the `doctor`
  placeholder, required purely for `main.go` to compile. The four scratch
  investigation programs/tests used to chase the Step 8 finding were all
  deleted before finishing; none remain in the tree.
- **Testing**: `go test -race ./... -v` — all green; `go vet`, cross-OS
  `go vet`, `gofmt -l .` — all clean. No stray warnings, no leftover
  binaries or scratch files in the repo.

## Concerns

1. **Primary, blocking**: I need your direct confirmation (not relayed) to
   stage and commit `go.mod`/`go.sum` alongside the source. See the timeline
   section above for exactly what's in them and how I verified it
   independently.
2. **Step 8 SDG-level finding**: flagged in detail above. My assessment is
   that this doesn't block shipping (it's an artifact of one-shot pipe
   testing, not real host behavior), but it's a real, reproducible defect in
   the pinned SDK version worth tracking (possibly upstream) rather than
   silently accepting. Your call on whether anything further is needed here.
3. `internal/service/errors.go`'s `sprintf` is a function, not a `var` alias
   as the brief's literal text says — functionally identical, done for
   `go vet` cleanliness. Flagging in case you'd prefer the literal `var
   sprintf = fmt.Sprintf` form.

## Fix pass — review findings

Base commit `7f997d6`. All four numbered findings and all five minor
findings addressed. `internal/vault`, `internal/lifecycle`, `internal/config`
untouched. `go mod tidy` not run.

### Starting state

`convert.go`, `recover.go`, `internal/service/errors.go` and
`internal/service/service.go` already carried uncommitted edits from a prior
session covering finding 3's `toolErr` rename, the `errors.Is` sentinel, and
the duplicated-cause message fix. However `internal/mcpsrv/server.go` still
called the old `toolError(err) *mcp.CallToolResult` symbol — the tree did
not compile. Fixed as part of finding 3 below.

### Finding 1 — stdin EOF never reaching the lifecycle

Transcribed `mirrorReader` from the plan
(`docs/superpowers/plans/2026-07-25-gobsidian-v01.md`, Task 9, right after
Step 8's IOTransport note) into `cmd/gobsidian/serve.go`, replacing
`io.TeeReader(os.Stdin, pw)`. It reads from `src`, forwards bytes to `dst`
(the `io.PipeWriter`), and on any non-nil read error calls
`dst.CloseWithError(err)` — propagating EOF (or any other read failure) to
the lifecycle's stdin monitor instead of silently discarding it.

### Finding 2 — in-flight step re-reading a drained channel

Added the `serveReturned` bool per the plan: set to `true` in the first
`select`'s `serveErr` branch, checked at the top of the `in-flight` step's
`Fn` to short-circuit to `nil` instead of re-selecting on an
already-drained channel.

### Finding 3 — error results shipping a zero-valued structured payload

Confirmed against the pinned SDK (`go-sdk@v1.5.0`, `mcp/server.go`
`toolForErr`'s inner handler, `mcp/protocol.go`
`CallToolResult.SetError`): when a typed tool handler returns a non-nil Go
`error` that is not a `*jsonrpc.Error`, the SDK builds `CallToolResult` via
`errRes.SetError(err)` and returns immediately — `IsError: true`, `Content`
set to the error text, and no `StructuredContent` is ever assigned (the
marshal-and-attach code is skipped entirely on that return path). It does
not escalate to a JSON-RPC protocol error. This confirms the brief's fix
shape is correct.

Completed the fix (partially already in place):
- `convert.go`: `toolErr(err) error` wraps `service.CodeOf(err)` and
  `err.Error()` into a plain Go error (already done by the prior session).
- `recover.go`: `guard`'s panic recovery sets the named `err` return to a
  wrapped `INTERNAL` error and zeroes `out`/`res` (already done).
- `server.go` (this session's fix — the tree didn't compile without it):
  `vault_stats`'s handler now returns
  `nil, service.StatsResult{}, toolErr(err)` instead of the old
  `toolError(err), service.StatsResult{}, nil`.

Verified empirically (below): a failing `vault_stats` call returns
`StructuredContent == nil` with `IsError: true` and content text
`"VAULT_UNAVAILABLE: varrendo o cofre"`.

### Finding 4 — `TestVaultStatsCountsNotes` asserting nothing

Rewrote the test to unmarshal `res.StructuredContent` into
`service.StatsResult` and assert `Notes == 2, Assets == 0` for the `A.md`
+ `sub/B.md` fixture.

Mutation proof — inverted the `IsNote` branch in
`internal/service/service.go` (`VaultStats`'s `Walk` callback: swapped
`out.Notes++`/`out.Assets++`), ran the test, restored, ran again:

```
$ go test ./internal/mcpsrv/ -run TestVaultStatsCountsNotes -v   # mutated
=== RUN   TestVaultStatsCountsNotes
    server_test.go:117: Notes = 0, want 2
    server_test.go:120: Assets = 2, want 0
--- FAIL: TestVaultStatsCountsNotes (0.01s)
FAIL

$ go test ./internal/mcpsrv/ -run TestVaultStatsCountsNotes -v   # restored
=== RUN   TestVaultStatsCountsNotes
--- PASS: TestVaultStatsCountsNotes (0.01s)
PASS
```

### Minor findings

- **`mcpsrv.Version` wired**: `main()` sets `mcpsrv.Version = version`
  before building the cobra command tree, so the linker-injected version
  reaches the MCP handshake instead of always reporting `"dev"`.
- **`os.Exit(0)` unconditional**: `runServe` now tracks `loopErr` (the
  error observed from `serveErr`, whichever branch captured it — the
  initial `select` or the `in-flight` step) and calls `os.Exit(1)` when it
  is non-nil, `os.Exit(0)` otherwise.
- **`errors.Is`**: already present from the prior session
  (`func (e *Error) Is(target error) bool` comparing `Code`). Confirmed it
  compiles; exercised transitively by `CodeOf`'s `errors.As` use.
- **Duplicated cause in `service.go`**: already fixed by the prior
  session — `Wrap(CodeVaultUnavailable, err, "varrendo o cofre")` no
  longer formats `%v` of `err` into the message.
- **`scripts/build.ps1` reference**: reworded the comment above the
  linker variables in `main.go` to note the script is Task 11 pending
  work rather than implying it exists today. Did not create the script.

### Verification

**1. `go test -race ./... -v`** — full run, all green, no `FAIL`.
`internal/service` and `cmd/gobsidian` correctly report "no test files";
`TestSignalCancelsContext` SKIPs on Windows as documented pre-existing.

```
ok  	github.com/jonyd/gobsidian/internal/config	(cached)
ok  	github.com/jonyd/gobsidian/internal/lifecycle	(cached)
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	(cached)
ok  	github.com/jonyd/gobsidian/internal/vault	(cached)
ok  	github.com/jonyd/gobsidian/tools/netcheck	(cached)
```

**2. Finding 4 mutation evidence** — see above.

**3. Finding 1 closed** — built the binary to a temp path, ran the plan's
Step 8 handshake pattern adapted to bash (`{ printf '%s\n' "$Req"; sleep 1;
} | ./gobsidian.exe serve --vault ... --log-level debug`), piping an
`initialize` request and letting stdin close naturally when the subshell
exits. Representative run:

```
time=...15:25:05.990... level=INFO  msg="servidor pronto" vault=...
time=...15:25:06.931... level=INFO  msg="encerramento solicitado" reason=stdin-eof
time=...15:25:06.931... level=DEBUG msg="etapa de encerramento concluida" step=in-flight duration=0s
time=...15:25:06.931... level=DEBUG msg="etapa de encerramento concluida" step=close-pipe duration=0s
```

`encerramento solicitado reason=stdin-eof` appears before both
shutdown-step lines, and critically it no longer depends on `close-pipe`
running (before the fix, `pw` was only ever closed by that step, so the
trigger could not fire until then, if ever).

Honesty note on ordering: across 5 repeated runs, the relative order of
`encerramento solicitado` vs. the `in-flight` step's own completion log
varied (3 of 5 had `encerramento solicitado` first; 2 of 5 had
`in-flight`'s "concluida" line first, by microseconds). This is a genuine
race, inherent to the design and not a residual bug: the SDK's own
transport reads the same physical stdin EOF that the mirrored copy
carries to the lifecycle monitor, so both "the SDK's `Run()` returns
cleanly" and "the lifecycle cancels ctx via stdin-eof" can independently
and validly unblock the first `select` in `runServe`. What the fix
guarantees, and what held in all 5 runs, is that `encerramento solicitado`
always fires within microseconds of stdin closing and always before
`close-pipe` — the structural defect the finding described (the trigger
previously could only fire *after* `close-pipe`, if at all) is gone.

**4. Finding 2 closed** — none of the runs produced an `etapa de
encerramento abandonada` line for `in-flight`; every run showed
`duration=0s` (once, `754.7µs`). Timed run:

```
$ time ( { printf '%s\n' "$Req"; sleep 1; } | ./gobsidian.exe serve --vault ... --log-level debug )
real	0m1.160s
```

Wall-clock time is dominated entirely by the test harness's fixed
1-second `sleep` (needed to hold stdin open long enough for the response
to be written, per the plan's Step 8 note) — the shutdown itself adds
effectively nothing, versus the ~3.5s it would have added before the fix.

**5. Finding 3 closed** — wrote a temporary test
(`internal/mcpsrv/zzmanual_verify_test.go`, deleted afterward, never
committed) that builds a vault, then calls `os.RemoveAll(root)` before
invoking `vault_stats` over an in-memory transport:

```
IsError: true
Content text: "VAULT_UNAVAILABLE: varrendo o cofre"
StructuredContent (<nil>): <nil>
```

`StructuredContent` is `nil` (absent), not a zeroed `StatsResult`. File
removed before finishing; `git status --porcelain` confirms no stray
files.

**6. Static checks** — all clean: `go vet ./...`, `gofmt -l .`,
`GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.

**7. `git status --porcelain`** — only the eight intentionally modified
files, no stray scratch files:

```
 M cmd/gobsidian/main.go
 M cmd/gobsidian/serve.go
 M internal/mcpsrv/convert.go
 M internal/mcpsrv/recover.go
 M internal/mcpsrv/server.go
 M internal/mcpsrv/server_test.go
 M internal/service/errors.go
 M internal/service/service.go
```

### Files changed this pass

- `cmd/gobsidian/main.go` — wire `mcpsrv.Version`, reword build.ps1 comment.
- `cmd/gobsidian/serve.go` — `mirrorReader`, `serveReturned`, `loopErr`/exit
  code.
- `internal/mcpsrv/server.go` — call `toolErr` instead of the removed
  `toolError`.
- `internal/mcpsrv/server_test.go` — real count assertions in
  `TestVaultStatsCountsNotes`.
- `internal/mcpsrv/convert.go`, `internal/mcpsrv/recover.go`,
  `internal/service/errors.go`, `internal/service/service.go` — carried
  over from the prior session's partial fix pass, verified and completed.

## Fix pass 2 — exit code and error wrapping

Base commit `a86cba9`. Second review round, two Important and three Minor
findings. `internal/vault`, `internal/lifecycle`, `internal/config`
untouched. `go mod tidy` not run.

### Finding 1 (Important) — nondeterministic exit code on a clean session

Root cause confirmed exactly as described: two independent paths detect the
same stdin close (the SDK's own EOF read off `mirrorReader`, and the
lifecycle's stdin monitor observing the mirrored copy and cancelling the
root context). Whichever wins the race decides whether `srv.Serve`'s
`loopErr` comes back `nil` or `context.Canceled`, and that value used to
drive `os.Exit(0)` vs `os.Exit(1)` directly — a client disconnecting cleanly
could exit either 0 or 1 depending on scheduling, indistinguishable at the
process boundary from a real failure.

Fix in `cmd/gobsidian/serve.go`, after `lc.Wait()`:

```go
if loopErr != nil && !errors.Is(loopErr, context.Canceled) {
	os.Exit(1)
}
os.Exit(0)
```

Added `"errors"` to the import block.

### Finding 2 (Important) — `toolErr` dropped `%w`

`internal/mcpsrv/convert.go` used `%s` on `err.Error()`, producing a plain
error with no `Unwrap`. Changed to:

```go
return fmt.Errorf("%s: %w", service.CodeOf(err), err)
```

Added `internal/mcpsrv/convert_test.go` (`TestToolErrPreservesUnwrap`):
builds a `*service.Error` via `service.Errorf`, wraps it with `toolErr`, and
asserts both `errors.Is(wrapped, domainErr)` and `errors.As(wrapped,
&asErr)` succeed, then checks `asErr.Code`.

**Mutation proof** — reverted `%w` back to `%s: %s"/err.Error()`, ran the
new test, restored, ran again:

```
$ go test ./internal/mcpsrv/ -run TestToolErrPreservesUnwrap -v   # mutated (%s)
=== RUN   TestToolErrPreservesUnwrap
    convert_test.go:20: errors.Is nao encontrou o *service.Error original atraves de toolErr; %w foi perdido
--- FAIL: TestToolErrPreservesUnwrap (0.00s)
FAIL

$ go test ./internal/mcpsrv/ -run TestToolErrPreservesUnwrap -v   # restored (%w)
=== RUN   TestToolErrPreservesUnwrap
--- PASS: TestToolErrPreservesUnwrap (0.00s)
PASS
```

### Finding 3 (Minor) — `loopErr` written from a shutdown step's goroutine

`internal/lifecycle/shutdown.go` confirms the mechanism: each `Step.Fn` runs
in its own goroutine (`go func() { done <- step.Fn(ctx) }()`), and if the
step's budget expires before it returns, `Shutdown` moves on without
waiting — the goroutine is explicitly left orphaned ("abandonada"). The
in-flight step in `serve.go` was writing straight into `loopErr`, an
outer-scope variable the main goroutine reads right after `lc.Wait()`: an
unsynchronized write/read pair.

Fix: replaced the direct assignment with a buffered channel.

```go
lateErr := make(chan error, 1)
...
lifecycle.Step{Name: "in-flight", ..., Fn: func(ctx context.Context) error {
	...
	case err := <-serveErr:
		lateErr <- err
	...
}},
...
select {
case err := <-lateErr:
	if loopErr == nil {
		loopErr = err
	}
default:
}
```

The non-blocking receive runs after `lifecycle.Shutdown` returns and before
`lc.Wait()`; if the step was abandoned and never sends, the `default` case
is taken and `loopErr` is left as whatever the first `select` captured.
Channel send/receive gives the required happens-before edge; `-race` stayed
clean across the full suite (see below).

### Finding 4 (Minor) — `*Error.Is` panic on typed-nil target

`internal/service/errors.go`: added the nil guard.

```go
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok || t == nil {
		return false
	}
	return e.Code == t.Code
}
```

Added `internal/service/errors_test.go` (`TestErrorIsTypedNilTarget`):
calls `errors.Is(err, (*Error)(nil))` and asserts it returns `false` instead
of panicking.

### Finding 5 (Minor) — mirror write failure poisoning the live stream

Transcribed the corrected `mirrorReader` from
`docs/superpowers/plans/2026-07-25-gobsidian-v01.md` (Task 9, the commit
`a86cba9` amendment) into `cmd/gobsidian/serve.go`: added a `broken bool`
field, set once when a write to the mirror pipe fails, after which the
mirror is skipped on subsequent reads and only the real read result
(`n, err` from `src.Read`) is returned. `dst.CloseWithError(err)` on a
genuine read error still runs regardless of `broken`, since that's the
actual EOF/error propagation the lifecycle depends on — only the auxiliary
write-to-mirror step is skipped once broken.

### Verification

**1. `go test -race ./... -v`** — full run, all green, no `FAIL`:

```
ok  	github.com/jonyd/gobsidian/internal/config	(cached)
ok  	github.com/jonyd/gobsidian/internal/lifecycle	(cached)
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	3.139s
ok  	github.com/jonyd/gobsidian/internal/service	1.651s
ok  	github.com/jonyd/gobsidian/internal/vault	(cached)
ok  	github.com/jonyd/gobsidian/tools/netcheck	(cached)
```
`cmd/gobsidian` reports "no test files" (expected, thin wiring layer).
`TestSignalCancelsContext` SKIPs on Windows as documented pre-existing.
New tests `TestToolErrPreservesUnwrap` (mcpsrv) and
`TestErrorIsTypedNilTarget` (service) both pass under `-race`.

**2. Exit-code evidence for finding 1** — built the binary to a temp path
and ran the plan's Step 8 handshake pattern 10 times with `--log-level
error`, then 10 more times with `--log-level debug` for visibility into
which race outcome occurred each run:

```
run 1 exit code: 0
run 2 exit code: 0
run 3 exit code: 0
run 4 exit code: 0
run 5 exit code: 0
run 6 exit code: 0
run 7 exit code: 0
run 8 exit code: 0
run 9 exit code: 0
run 10 exit code: 0
```
(identical result across both the `error`-level and `debug`-level batches —
20 runs total, all exit code 0). The debug batch's stderr showed
`msg="encerramento solicitado" reason=stdin-eof"` in every run with no
`"servidor encerrou com erro"` line, and stdout carried the correct single
`initialize` JSON-RPC response each time. No run produced a non-zero exit
code, so there was no error value to report beyond `context.Canceled`
itself (which the fix now classifies as clean).

**3. Finding 2 mutation proof** — see above, both outputs shown, restored,
confirmed green.

**4. Static checks** — all clean, no output:
```
go vet ./...
gofmt -l .
GOOS=linux go vet ./...
GOOS=darwin go vet ./...
```

**5. `git status --porcelain`** — only the intended changes:
```
 M cmd/gobsidian/serve.go
 M internal/mcpsrv/convert.go
 M internal/service/errors.go
?? internal/mcpsrv/convert_test.go
?? internal/service/errors_test.go
```

### Files changed this pass

- `cmd/gobsidian/serve.go` — `context.Canceled` treated as clean shutdown;
  `lateErr` channel replaces the shared `loopErr` write from the in-flight
  shutdown step; `mirrorReader` gets the `broken` guard.
- `internal/mcpsrv/convert.go` — `toolErr` wraps with `%w` instead of
  formatting with `%s`.
- `internal/mcpsrv/convert_test.go` (new) — `TestToolErrPreservesUnwrap`.
- `internal/service/errors.go` — `*Error.Is` guards against a typed-nil
  target.
- `internal/service/errors_test.go` (new) — `TestErrorIsTypedNilTarget`.
