# Task 7 Report: Caminho canônico e confinamento

## What was implemented

`internal/vault/path.go` and `internal/vault/path_test.go`, exactly as specified in the brief (no deviations):

- `CanonicalPath` — string alias for the internal note-naming form (root-relative, `/`-separated, no leading `./`, exact on-disk casing preserved).
- `ErrOutsideVault`, `ErrAbsolutePath`, `ErrEmptyPath` — sentinel errors, checkable with `errors.Is`.
- `Resolve(root, input string) (string, CanonicalPath, error)` — turns a tool's raw input into an absolute path plus canonical form, rejecting absolute inputs and anything that escapes the vault.
- `Canonicalize(root, abs string) (CanonicalPath, error)` — converts an absolute path to `CanonicalPath`, using `filepath.Rel` (component-wise) rather than string-prefix comparison for the confinement check.
- `hasDriveLetter` — unexported helper, detects `C:/...`-style drive prefixes even on platforms where `filepath.IsAbs` wouldn't catch the Windows form.

No I/O, no `ctx`, no imports of `net`/`net/*`. Package is a leaf, as required.

## TDD Evidence

**RED**

```
$ go test ./internal/vault/ -v
github.com/jonyd/gobsidian/internal/vault: no non-test Go files in C:\Users\jonyd\Projetos\Gobsidian\internal\vault
FAIL	github.com/jonyd/gobsidian/internal/vault [build failed]
FAIL
```

Failed because `path.go` did not exist yet — the package had only the test file, so it couldn't build at all. This is the expected precursor state to "undefined: vault.Resolve"; the brief's exact wording assumes a scaffolded (even if empty) `path.go`, which wasn't present here since this is a from-scratch package. Confirmed for the right reason: no implementation existed.

**GREEN**

```
$ go test -race ./internal/vault/ -v
=== RUN   TestResolveConfinement
=== RUN   TestResolveConfinement/simples
=== RUN   TestResolveConfinement/barra_invertida
=== RUN   TestResolveConfinement/ponto_inicial
=== RUN   TestResolveConfinement/duplo_ponto_interno
=== RUN   TestResolveConfinement/escapa_do_cofre
=== RUN   TestResolveConfinement/escapa_com_muitos_niveis
=== RUN   TestResolveConfinement/absoluto_rejeitado
=== RUN   TestResolveConfinement/absoluto_unix_rejeitado
=== RUN   TestResolveConfinement/vazio_rejeitado
--- PASS: TestResolveConfinement (0.00s)
    (all 9 subcases PASS)
=== RUN   TestResolveRejectsSiblingWithSharedPrefix
--- PASS: TestResolveRejectsSiblingWithSharedPrefix (0.00s)
=== RUN   TestCanonicalizeUsesForwardSlashes
--- PASS: TestCanonicalizeUsesForwardSlashes (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/vault	1.550s
```

11 subcases total (9 + 1 + 1), all PASS, race detector clean.

Additional checks, all clean:
- `gofmt -l .` — no output
- `go vet ./...` — clean
- `GOOS=linux go build ./...` — clean
- `GOOS=linux go vet ./...` — clean

## Adversarial cases (verified against the actual build, root = `C:/cofre`)

All probes were run via a throwaway `main.go` calling `vault.Resolve` directly, then deleted — no test files were added beyond the brief's table (nothing here revealed a gap worth a permanent test).

| Case | Input | Result |
|---|---|---|
| `..` alone | `".."` | Rejected: `caminho fora do cofre: "C:\\"` |
| `../` | `"../"` | Rejected: `caminho fora do cofre: "C:\\"` |
| `a/../../b` | `"a/../../b"` | Rejected: `caminho fora do cofre: "C:\\b"` |
| Trailing separator | `"Civil/"` | Accepted: `canon="Civil"` (resolves to the folder itself; no I/O here, so nothing distinguishes file vs. dir at this layer) |
| Only separators | `"/"` | Rejected: `caminho absoluto nao aceito: "/"` |
| Only separators | `"//"` | Rejected: `caminho absoluto nao aceito: "//"` |
| Only separators | `"\"` (single backslash) | Rejected: `caminho absoluto nao aceito: "\\"` |
| UNC path | `"\\server\share\note.md"` | Rejected: `caminho absoluto nao aceito: "\\server\\share\\note.md"` (Windows `filepath.IsAbs` recognizes UNC form) |
| NUL byte, plain | `"Civil/\x00PONTO.md"` | **Accepted**: `canon="Civil/\x00PONTO.md"` — the NUL byte passes through untouched, no rejection |
| NUL byte + traversal | `"Civil/\x00/../../../etc/passwd"` | Rejected: `caminho fora do cofre: "C:\\etc\\passwd"` — Go's string-based path cleaning treats the NUL as an ordinary byte inside a component, not a terminator, so the `..` sequence is still counted correctly and confinement holds |
| `Civil/./PONTO 03.md` | — | Accepted: `canon="Civil/PONTO 03.md"` (internal `.` collapsed) |
| Empty component | `"Civil//PONTO 03.md"` | Accepted: `canon="Civil/PONTO 03.md"` (double slash collapsed by `path.Clean`) |
| Casing differs from input | `"civil/PONTO 03.MD"` | Accepted: `canon="civil/PONTO 03.MD"` — output preserves exactly the casing that was *passed in*. Confirmed explicitly: `Resolve`/`Canonicalize` do no disk lookup, so there is no on-disk casing to reconcile against at this layer; that is Task 8's/the indexer's job. |

**Finding, not a vulnerability:** a NUL byte inside a path component is accepted rather than rejected outright. It does not enable traversal — as shown by the combined case, `..` sequences around a NUL are still resolved and confined correctly by Go's pure string-based `path`/`filepath` cleaning, which does not treat NUL as a component/path terminator the way a C string or an OS syscall might. The risk, if any, would only surface if a later layer (Task 8's file-open code) hands this string to an OS syscall that *does* truncate at NUL, producing a shorter — but still vault-relative, still non-traversing — path. I did not add a rejection for this because the brief does not ask for one and truncation cannot introduce an escape; flagging it here per the task's adversarial-case instructions rather than silently accepting it.

## Files changed

- `internal/vault/path.go` (new)
- `internal/vault/path_test.go` (new)

## Self-review

- **Completeness:** every symbol in the brief's "Produces" line exists: `CanonicalPath`, `Canonicalize`, `Resolve`, `ErrOutsideVault` (plus `ErrAbsolutePath`/`ErrEmptyPath`, which the brief's own Step 3 code defines and uses even though the "Produces" line only names `ErrOutsideVault`).
- **Security:** walked `Resolve` and `Canonicalize` by hand for every code path — absolute-path checks happen before any join with `root`, and the only path that returns a non-error `CanonicalPath` is the one that passes `filepath.Rel` and the `..`/`../`-prefix check on the *relative* result, which is component-wise, not string-prefix. No path found that returns a value for input outside the root. The sibling-prefix test (`../cofre-outro/...`) confirms this specifically, and it passes.
- **Discipline:** implemented exactly what the brief specified — no extra exported symbols, no speculative helpers, no `ctx`. `hasDriveLetter` is the one unexported helper the brief calls for.
- **Testing:** `go test -race ./internal/vault/ -v` output is pristine — no warnings, no skipped tests. The only shell warnings seen anywhere were `git`'s LF→CRLF autocrlf notices at commit time, unrelated to test output.

## Mechanical corrections to the brief's code

None. The brief's `path.go` and `path_test.go` compiled and passed verbatim.

## Concerns

- The NUL-byte-in-path-component finding above (informational, not a defect at this layer — flagging per the adversarial-case instructions since Task 8 will be the layer that actually opens files).
- The RED step's failure message differed slightly from the brief's stated expectation (`no non-test Go files` vs. `undefined: vault.Resolve`) because this is a from-scratch package with no pre-existing `path.go` stub — the underlying reason (implementation absent) is the same, just one step earlier in the same category of failure.

## Fix pass — security findings

Follow-up pass fixing nine review findings (device names writable, drive-letter leak through `path.Clean`, multiple canonical identities via trailing dot/space, Linux-hostile tests, untested `abs`, uncollapsed sentinels, uncovered `Canonicalize` rejections, and two overclaiming doc comments). Code and tests transcribed verbatim from `docs/superpowers/plans/2026-07-25-gobsidian-v01.md`, section "Task 7: Caminho canônico e confinamento" (commit `3f6606b`). No mechanical corrections were needed — the plan's code compiled and passed as written.

**Changes to `internal/vault/path.go`:**
- Added `ErrInvalidPath` sentinel (fourth sentinel; doc comment explains why the four stay distinct — each maps to a different MCP error code).
- Replaced `hasDriveLetter` with `validateLocal`, built on `filepath.IsLocal`. Called both in `Resolve` (before joining with `root`) and again at the end of `Canonicalize` (output invariant, since `Canonicalize` is called directly by Task 8's walker without going through `Resolve`).
- `validateLocal` also rejects NUL bytes and components ending in `.` or space (which `filepath.IsLocal` considers local and does not reject).
- Corrected the `CanonicalPath` doc comment (no longer claims "a grafia exata do disco" without qualification — added a paragraph clarifying this layer does no disk I/O) and the `Resolve` doc comment (no longer claims to reject "qualquer coisa que escape do cofre" — added a "LIMITE CONHECIDO" paragraph noting the check is lexical only and does not follow symlinks/junctions).

**Changes to `internal/vault/path_test.go`:** rewritten per the plan — `testRoot()` platform-correct helper (`runtime.GOOS` check), `windowsOnly` flag on Windows-specific cases (backslash conversion, device names), `errors.Is` assertions naming the expected sentinel per case instead of a bare `wantErr bool`, an `abs` assertion against `filepath.Join(root, ...)` on every success case, and a new `TestCanonicalizeRejectsStandalone` covering `Canonicalize`'s rejection paths directly (sibling-prefix, above-root, root-itself, drive-letter-inside-root).

### 1. `go test -race ./internal/vault/ -v`

All 22 `TestResolveConfinement` subcases + `TestCanonicalizeRejectsStandalone` (4 subcases) + `TestResolveRejectsSiblingWithSharedPrefix` + `TestCanonicalizeUsesForwardSlashes` — PASS, race-clean, pristine output (full transcript captured during the session; re-run confirmed cached PASS with no new output).

```
ok  	github.com/jonyd/gobsidian/internal/vault	1.504s
```

### 2. `GOOS=linux go vet ./...` and `GOOS=darwin go vet ./...`

Both clean (exit 0, no output). `windowsOnly`-gated cases mean the Linux/darwin runs (which I cannot execute directly on this Windows machine) would `t.Skip` the eight Windows-only subcases (backslash conversion, `NUL`/`Civil/NUL`/`COM1`/`CON`) rather than fail.

### 3. Findings 1–3 closed — empirical proof

Standalone Go program under scratch (`.../scratchpad/probe/{main.go,path.go}`, package `main` with a byte-for-byte copy of the fixed `path.go` logic, since `internal/` packages can't be imported cross-module). Run with `go run .`, then the whole probe directory was deleted with `rm -rf`.

```
root = C:\Users\jonyd\AppData\Local\Temp\vaultprobe3560372498\cofre

Resolve(root, "NUL") -> ERROR: caminho absoluto nao aceito: "NUL"
Resolve(root, "COM1") -> ERROR: caminho absoluto nao aceito: "COM1"
Resolve(root, "./C:/Windows/win.ini") -> ERROR: caminho absoluto nao aceito: "C:/Windows/win.ini"
Resolve(root, "Civil/A.md.") -> ERROR: caminho malformado: componente "A.md." termina em ponto ou espaco, o que o Windows remove ao abrir e faria o mesmo arquivo ter mais de um caminho canonico
```

All four now return errors (no `os.WriteFile` attempted since `Resolve` never produced a path). `NUL`/`COM1` are caught by `filepath.IsLocal`, which rejects Windows reserved device names; `./C:/Windows/win.ini` is caught by the same check after `path.Clean` strips the leading `./`; `Civil/A.md.` is caught by the explicit trailing-dot/space rejection.

### 4. Suite-bites mutation test

Temporarily replaced the component-wise confinement check in `Canonicalize` with a `strings.HasPrefix(slashedAbs, slashedRoot)` comparison (kept `validateLocal` call intact). Ran `go test ./internal/vault/ -v`:

```
=== RUN   TestCanonicalizeRejectsStandalone/irmao_com_prefixo_compartilhado
    path_test.go:147: Canonicalize("C:\\cofre-outro\\A.md") = "-outro/A.md", quer erro caminho fora do cofre
--- FAIL: TestCanonicalizeRejectsStandalone (0.00s)
    --- FAIL: TestCanonicalizeRejectsStandalone/irmao_com_prefixo_compartilhado (0.00s)
...
FAIL
FAIL	github.com/jonyd/gobsidian/internal/vault	0.392s
```

**Deviation from the plan's stated expectation, reported rather than papered over:** the plan's comment on `TestResolveRejectsSiblingWithSharedPrefix` claims it is "o unico teste do arquivo que uma implementacao com strings.HasPrefix reprovaria." Under this mutation that specific test still **passed**, because `Resolve`'s `validateLocal` call (via `filepath.IsLocal`, added in this fix pass) independently rejects any input with a leading `..` component — including `"../cofre-outro/A.md"` — before `Canonicalize` is ever reached. The mutation was instead caught by `TestCanonicalizeRejectsStandalone/irmao_com_prefixo_compartilhado`, which calls `Canonicalize` directly and bypasses `Resolve`'s lexical pre-check entirely — exactly the standalone-coverage gap that finding 7 asked to close. The suite as a whole still failed (`go test ./internal/vault/` exit 1) under the mutation, so the security property remains under test; it is now defended in depth by two independent checks rather than caught by exactly one named test. This makes the plan's test-file comment stale but does not indicate any gap in the implementation — I did not alter the comment, since the task scope was transcription, not rewriting plan prose.

Restored `Canonicalize` to the plan's exact version. Re-ran:

```
ok  	github.com/jonyd/gobsidian/internal/vault	(cached)
```

All green. `git diff -- internal/vault/path.go` afterward showed only the intended full-file changes (new `ErrInvalidPath`, `validateLocal`, updated doc comments) — no leftover mutation artifacts.

### 5. Final checks

`go vet ./...` — clean (exit 0). `gofmt -l .` — no output (exit 0). `git status --porcelain` — only `internal/vault/path.go` and `internal/vault/path_test.go` modified; no stray files (scratch probe directory was created and deleted outside the repo).

## Fix pass 2 — stale test comment

The doc comment above `TestResolveRejectsSiblingWithSharedPrefix` claimed it was the only test that would fail with `strings.HasPrefix`. After the two-layer confinement fix, this is no longer true: `validateLocal` (added in fix pass 1) rejects the leading `..` before `Canonicalize` is reached via `Resolve`, so the actual proof is in `TestCanonicalizeRejectsStandalone` (which calls `Canonicalize` directly). Updated the comment to reflect both tests defend against the same attack from different angles.

### 1. `go test -race ./internal/vault/ -v`

```
=== RUN   TestResolveConfinement
=== RUN   TestResolveConfinement/simples
=== RUN   TestResolveConfinement/ponto_inicial
=== RUN   TestResolveConfinement/duplo_ponto_interno
=== RUN   TestResolveConfinement/componente_vazio_colapsa
=== RUN   TestResolveConfinement/ponto_interno_colapsa
=== RUN   TestResolveConfinement/barra_invertida
=== RUN   TestResolveConfinement/escapa_do_cofre
=== RUN   TestResolveConfinement/escapa_com_muitos_niveis
=== RUN   TestResolveConfinement/duplo_ponto_sozinho
=== RUN   TestResolveConfinement/absoluto_unix_rejeitado
=== RUN   TestResolveConfinement/vazio_rejeitado
=== RUN   TestResolveConfinement/so_separadores
=== RUN   TestResolveConfinement/drive_apos_ponto_inicial
=== RUN   TestResolveConfinement/absoluto_com_drive
=== RUN   TestResolveConfinement/dispositivo_NUL
=== RUN   TestResolveConfinement/dispositivo_em_subpasta
=== RUN   TestResolveConfinement/dispositivo_COM1
=== RUN   TestResolveConfinement/dispositivo_CON
=== RUN   TestResolveConfinement/componente_termina_em_ponto
=== RUN   TestResolveConfinement/componente_termina_em_espaco
=== RUN   TestResolveConfinement/pasta_termina_em_ponto
=== RUN   TestResolveConfinement/byte_nulo
--- PASS: TestResolveConfinement (0.00s)
=== RUN   TestCanonicalizeRejectsStandalone
=== RUN   TestCanonicalizeRejectsStandalone/irmao_com_prefixo_compartilhado
=== RUN   TestCanonicalizeRejectsStandalone/acima_da_raiz
=== RUN   TestCanonicalizeRejectsStandalone/a_propria_raiz
=== RUN   TestCanonicalizeRejectsStandalone/letra_de_drive_dentro_da_raiz
--- PASS: TestCanonicalizeRejectsStandalone (0.00s)
=== RUN   TestResolveRejectsSiblingWithSharedPrefix
--- PASS: TestResolveRejectsSiblingWithSharedPrefix (0.00s)
=== RUN   TestCanonicalizeUsesForwardSlashes
--- PASS: TestCanonicalizeUsesForwardSlashes (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/vault	1.839s
```

All tests PASS, race-clean, pristine output.

### 2. `go vet ./...` and `gofmt -l .`

```
=== go vet PASSED ===
=== gofmt check PASSED ===
```

Both clean (exit 0, no output).

### 3. `git status --porcelain`

```
 M internal/vault/path_test.go
```

Only one file changed, no stray files.

### 4. `git show --stat HEAD`

```
commit 3f63611dc98f4ff62327845664faed34b8230208
Author: jonyduque <jonyduque@hotmail.com>
Date:   Sun Jul 26 10:07:12 2026 -0300

    docs: clarify test coverage for prefix-comparison security check
    
    The doc comment above TestResolveRejectsSiblingWithSharedPrefix claimed it
    was the only test in the file that would fail with strings.HasPrefix. This
    was true before a second confinement layer (validateLocal) was added to
    Resolve. The comment is now stale: in practice, the prefix bug is caught
    at validateLocal before reaching Canonicalize through Resolve, and the
    actual proof that component-wise comparison is required is in
    TestCanonicalizeRejectsStandalone.
    
    Updated the comment to clarify that both tests cover the same attack from
    different angles: validateLocal rejects it before Canonicalize is reached
    via Resolve, while TestCanonicalizeRejectsStandalone calls Canonicalize
    directly and is the one that would fail with strings.HasPrefix.
    
    Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>

 internal/vault/path_test.go | 9 ++++++---
 1 file changed, 6 insertions(+), 3 deletions(-)
```

Exactly one file changed (6 insertions, 3 deletions for the comment replacement).

## Fix pass 3 — portability and sentinel semantics

Third hardening pass fixed three regressions a reviewer confirmed by transcribing the standard library's Unix code paths and simulating a Linux run: (1) the unconditional trailing-dot/space rejection made legitimate Linux/macOS notes (`Notas `, `Arquivo.`) unreachable through `Resolve`, since the rule's entire justification is Windows-specific; (2) `validateLocal` guaranteeing a local, joined path made `ErrOutsideVault` unreachable from `Resolve` — traversal attempts surfaced as `caminho absoluto nao aceito` instead; (3) three tests (`drive apos ponto inicial`, `absoluto com drive`, and all of `TestCanonicalizeRejectsStandalone`) had no `windowsOnly` guard and would fail under `GOOS=linux`/`GOOS=darwin`, since `filepath.IsLocal`'s colon/device-name rejection lives only in the Windows implementation.

Code and tests transcribed verbatim from `docs/superpowers/plans/2026-07-25-gobsidian-v01.md`, section "Task 7: Caminho canônico e confinamento" (as updated by commit `f06751f`, which corrected the plan doc itself but had not yet been applied to the actual `internal/vault` code — this pass applied it). No mechanical corrections were needed; the plan's code compiled and passed as written.

**Changes:**
- `internal/vault/path.go`: rewrote `validateLocal` so check order determines the sentinel — NUL byte -> `ErrInvalidPath`; a `..` surviving `path.Clean` -> `ErrOutsideVault`; a rooted path, including the `C:algo` form `filepath.VolumeName` catches -> `ErrAbsolutePath`; anything else non-local (device names via `filepath.IsLocal`) -> `ErrInvalidPath`; then delegates to the new `validatePlatformPath`. Removed the inline trailing-dot/space loop from `validateLocal`.
- New `internal/vault/path_windows.go` (`//go:build windows`): holds the trailing dot/space rule, with its Win32-specific rationale in the doc comment.
- New `internal/vault/path_other.go` (`//go:build !windows`): no-op `validatePlatformPath` returning `nil`.
- `internal/vault/path_test.go`: traversal cases (`escapa do cofre`, `escapa com muitos niveis`, `duplo ponto sozinho`) now expect `ErrOutsideVault`; device-name cases now expect `ErrInvalidPath`; drive-letter cases (`drive apos ponto inicial`, `absoluto com drive`), device cases, and trailing dot/space cases all carry `windowsOnly: true`; `TestCanonicalizeRejectsStandalone` gained a `windowsOnly` field, applied to its drive-letter-inside-root case; `TestResolveRejectsSiblingWithSharedPrefix` reverted to asserting `errors.Is(err, vault.ErrOutsideVault)`.

No `if runtime.GOOS ==` was added to `path.go` — the two build-tagged files are the only platform gate, per the project rule that platform-specific behavior lives behind build tags, not runtime checks, in shared logic.

### 1. `go test -race ./internal/vault/ -v`

```
=== RUN   TestResolveConfinement
    (22 subcases, including drive/device/trailing-dot-space cases, all run natively since this machine is windows)
--- PASS: TestResolveConfinement (0.00s)
=== RUN   TestCanonicalizeRejectsStandalone
    (4 subcases)
--- PASS: TestCanonicalizeRejectsStandalone (0.00s)
=== RUN   TestResolveRejectsSiblingWithSharedPrefix
--- PASS: TestResolveRejectsSiblingWithSharedPrefix (0.00s)
=== RUN   TestCanonicalizeUsesForwardSlashes
--- PASS: TestCanonicalizeUsesForwardSlashes (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/vault	1.523s
```

All 26 subcases PASS, race-clean, pristine output (no skips — running natively on Windows, so every `windowsOnly` case executes rather than being skipped).

### 2. `GOOS=linux go vet ./...` and `GOOS=darwin go vet ./...`

Both exit 0, no output.

### 3. Critical regression proof — non-Windows acceptance

Standalone program at `scratchpad/probe_task7/main.go` (deleted after use), transcribing `Resolve`, `Canonicalize`, `validateLocal`, and the **non-Windows** `validatePlatformPath` (hardcoded as the no-op body from `path_other.go`) verbatim, fed `"Notas "`, `"Notas /A.md"`, `"Arquivo."`, `"Civil/A.md."` against root `/cofre`:

```
Resolve("Notas ") => ACCEPTED abs="\cofre\Notas " canon="Notas "
Resolve("Notas /A.md") => ACCEPTED abs="\cofre\Notas \A.md" canon="Notas /A.md"
Resolve("Arquivo.") => ACCEPTED abs="\cofre\Arquivo." canon="Arquivo."
Resolve("Civil/A.md.") => ACCEPTED abs="\cofre\Civil\A.md." canon="Civil/A.md."
```

All four accepted. (`filepath.Join`/`filepath.FromSlash` render the root with backslashes even though the program ran on the Windows toolchain — irrelevant to the proof, since the function under test is the no-op `validatePlatformPath`, the same one `GOOS=linux`/`GOOS=darwin` builds select. The Windows build of `filepath.IsLocal`, called earlier in `validateLocal`, does not reject any of these four inputs either — none are device names or contain a colon — so the result accurately reflects what a Linux/macOS build would do.) Probe directory deleted after the run.

### 4. Windows rule still holds

Probed via a temporary `internal/vault/zzz_probe_test.go` (created, run, then deleted — not part of the commit) calling the real, compiled `vault.Resolve` on this Windows machine with root `C:\cofre`:

```
Resolve("Civil/A.md.")           => err = caminho malformado: componente "A.md." termina em ponto ou espaco, ...
Resolve("Civil/A.md ")           => err = caminho malformado: componente "A.md " termina em ponto ou espaco, ...
Resolve("NUL")                   => err = caminho malformado: "NUL" nao e um caminho local
Resolve("COM1")                  => err = caminho malformado: "COM1" nao e um caminho local
Resolve("./C:/Windows/win.ini")  => err = caminho absoluto nao aceito: "C:/Windows/win.ini"
```

All five rejected. Sentinels: the two trailing dot/space cases and both device names return `ErrInvalidPath`; `./C:/Windows/win.ini` returns `ErrAbsolutePath` (caught by `filepath.VolumeName` after `path.Clean` strips the leading `./`).

### 5. Sentinels distinct again

Same temporary probe test, root `C:\cofre`:

```
Resolve("../outro/A.md") => ErrOutsideVault  (matches: true)
Resolve("C:/cofre/A.md") => ErrAbsolutePath  (matches: true)
Resolve("NUL")           => ErrInvalidPath   (matches: true)
```

All three match their expected sentinel via `errors.Is`.

### 6. Final checks

`go vet ./...` — clean (exit 0). `gofmt -l .` — no output (exit 0). `go build ./...` — clean. `git status --porcelain` after commit — clean working tree, no stray files (temporary probe files were deleted before the final check).

### 7. Commit

```
b85d695 fix(vault): gate windows-only path rules behind build tags
 4 files changed, 104 insertions(+), 51 deletions(-)
 create mode 100644 internal/vault/path_other.go
 create mode 100644 internal/vault/path_windows.go
```

No deviations from the plan's code — everything transcribed verbatim and compiled/passed on the first attempt.
