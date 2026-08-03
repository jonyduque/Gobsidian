# Task 1 Report — Bootstrap do módulo e do CI

## Status: DONE_WITH_CONCERNS

One deviation from the environment brief was required and is documented in detail below (go.mod's `go` directive could not stay at `1.24`). Everything else matches the brief.

## What was implemented

- `go.mod` / `go.sum` — module `github.com/jonyd/gobsidian`, all nine pinned dependencies resolved (`go-sdk@v1.5.0` exactly, plus goldmark, fsnotify, xxhash/v2, cobra, x/text, x/sys, yaml.v3, x/tools/go/analysis).
- `tools/netcheck/netcheck.go` — the `netcheck.Analyzer` (`go/analysis`) that reports any `net` / `net/*` import, verbatim per the brief.
- `tools/netcheck/netcheck_test.go` + `tools/netcheck/testdata/src/a/a.go` — `analysistest`-based test with the fixture importing `net/http` (forbidden) and `os` (allowed).
- `scripts/check_net.ps1` — copied verbatim from `docs/ESTRUTURA.md` §4, prefixed with `Set-StrictMode -Version Latest` / `$ErrorActionPreference = "Stop"` as instructed.
- `.github/workflows/ci.yml` — `test` (matrix: windows/ubuntu/macos, `go vet` + `go test -race`), `netcheck` (runs `scripts/check_net.ps1` via `shell: pwsh`), `lint` (`golangci-lint-action@v6`). Verbatim from the brief.
- `.golangci.yml` — verbatim from the brief.
- `Makefile` — verbatim from the brief (`test`, `lint`, `netcheck`, `build`, `bench` targets).
- `.gitignore` — merged the brief's entries (`bin/`, `coverage.out`, `*.exe`, `cofre-debug.log`, `gobsidian-debug.log`) into the existing file; `.superpowers/` preserved.
- `LICENSE` — MIT, copyright holder `jonyd`, year 2026.

No files named `helpers.go`/`utils.go`/`common.go`. No domain code (`internal/`, `cmd/`) was created — out of scope for Task 1.

## TDD Evidence

**RED** — `go test ./tools/netcheck/ -run TestNetCheck -v` (run before `netcheck.go` existed, only the test file and fixture were present):

```
github.com/jonyd/gobsidian/tools/netcheck: no non-test Go files in C:\Users\jonyd\Projetos\Gobsidian\tools\netcheck
FAIL	github.com/jonyd/gobsidian/tools/netcheck [build failed]
FAIL
```

This is the expected failure category (package doesn't build because `Analyzer` isn't defined yet). The brief anticipated the message as `undefined: netcheck.Analyzer`; the actual Go toolchain phrases the same root cause as "no non-test Go files" because `netcheck.go` was entirely absent rather than present-but-missing-a-symbol. Same cause (analyzer not implemented), different exact wording — noted per the task's guidance to report analyzer-test discrepancies.

**GREEN** — `go test ./tools/netcheck/ -run TestNetCheck -v` (after `netcheck.go` was added):

```
=== RUN   TestNetCheck
--- PASS: TestNetCheck (9.33s)
PASS
ok  	github.com/jonyd/gobsidian/tools/netcheck	10.870s
```

## Verification (Step 7)

```
go vet ./...            -> clean, no output
go test -race ./...     -> ok  github.com/jonyd/gobsidian/tools/netcheck  ...
gofmt -l .               -> clean, no output
go build ./...            -> clean, no output
go mod verify              -> all modules verified
pwsh -NoProfile -File scripts/check_net.ps1
  -> [OK] Nenhum pacote de internal/ ou cmd/ importa rede
```

All ran cleanly, matching the brief's "sem erros; netcheck passa; nenhum outro pacote existe ainda."

## Files changed

- `go.mod`, `go.sum` (new)
- `tools/netcheck/netcheck.go` (new)
- `tools/netcheck/netcheck_test.go` (new)
- `tools/netcheck/testdata/src/a/a.go` (new)
- `scripts/check_net.ps1` (new)
- `.github/workflows/ci.yml` (new)
- `.golangci.yml` (new)
- `Makefile` (new)
- `LICENSE` (new)
- `.gitignore` (modified — merged brief's entries)

Commit: `8f951e5` — `chore: bootstrap module, CI, and network-import analyzer`

## The deviation: go.mod's `go` directive

The environment brief said: *"`go.mod` must still declare `go 1.24` — that is a minimum-version floor, not the toolchain version. Do not raise it."*

I verified this is technically impossible while depending on the pinned `github.com/modelcontextprotocol/go-sdk@v1.5.0`. That module's own `go.mod` reads:

```
module github.com/modelcontextprotocol/go-sdk

go 1.25.0
```

Go's module graph rules (in effect since Go 1.21, module graph pruning) require the main module's `go` directive to be **at least** the highest `go` directive among all modules in the build list. This is not a style choice enforced by `go get`/`go mod tidy` as a convenience — the tooling actively rewrites (or refuses to build without rewriting) the main module's directive to satisfy this invariant. I confirmed this empirically:

1. Manually set `go.mod` back to `go 1.24` after adding all dependencies.
2. Ran `go build golang.org/x/tools/go/analysis/analysistest` → `go: updates to go.mod needed; to update it: go mod tidy`.
3. Ran `go mod tidy -e` → it silently rewrote the directive to `go 1.25.0` with no way to keep it at `1.24` short of dropping the pinned SDK version.

Since the brief's Global Constraints are explicit and unconditional that `github.com/modelcontextprotocol/go-sdk` "must resolve to v1.5.0 ... If it does not resolve, STOP and report BLOCKED — do not fall back to `@latest`," and since that version's own minimum Go requirement is what's forcing the directive up, I treated the SDK pin as the higher-priority constraint and let `go.mod` settle at the minimum necessary value: `go 1.25.0`. This is still a floor (not a toolchain pin — the installed toolchain is 1.26.4, which remains untouched and above the floor), so it preserves the spirit of "floor, not toolchain version" even though it numerically exceeds `1.24`.

**Downstream effect on `.github/workflows/ci.yml`:** I left `go-version: '1.24'` in the CI YAML verbatim, per the brief's exact-content instruction. This should still work because Go's automatic toolchain switching (`GOTOOLCHAIN=auto`, the default) will detect `go.mod`'s `go 1.25.0` requirement and transparently download/use a matching toolchain during `go vet`/`go test` on the CI runners. I could not verify this on an actual GitHub Actions runner (only locally, where the installed toolchain is already 1.26.4 and the question doesn't arise). If GitHub Actions' `GOTOOLCHAIN` policy or network egress differs from a normal `go` install, this could fail — flagging as something to watch on first CI run.

I did not see a way to avoid this conflict without violating one of the two constraints; I made the call to preserve the SDK version pin (explicitly called out as a product decision, PRD D6) since numerically raising a "floor" while keeping the actual toolchain (1.26.4) fixed does not compromise the intent as much as breaking the pinned SDK dependency would.

## Self-review

- **Completeness:** All Step 1–8 deliverables present; nothing from Task 2+ implemented.
- **Quality:** File contents match the brief verbatim except for the necessary `go.mod`/`go.sum` state (dependency resolution is deterministic content, not authored text) and the `go` directive deviation above.
- **Discipline:** No domain code, no extra packages, no `helpers.go`/`utils.go`/`common.go`.
- **Testing:** `TestNetCheck` exercises real analyzer behavior against a fixture with one forbidden (`net/http`) and one allowed (`os`) import; output is pristine (no stray warnings in `go test -race -v`).

## Issues / concerns

1. **go.mod `go` directive is `1.25.0`, not `1.24`** — see detailed explanation above. This is the main item needing product-owner acknowledgment; it was unavoidable given the pinned SDK version's own requirement.
2. **`golangci-lint` was not run locally** — not installed in this environment, consistent with the brief's anticipated friction. `.golangci.yml` is written per spec and left for CI's `golangci-lint-action@v6` to execute.
3. **`go.mod`'s other 7 pinned dependencies (goldmark, fsnotify, xxhash/v2, cobra, x/text, x/sys, yaml.v3) currently show as `// indirect`** with no direct importer, because Task 1 has no domain code. This is expected: only `golang.org/x/tools` (via `netcheck`) is actually imported by any `.go` file right now. Their *versions* were pinned via explicit `go get <module>` (no `@latest` used anywhere), and `go.sum` has verified checksums for them, so subsequent tasks that write the actual `import` statements will not silently drift to different versions — module resolution is deterministic from the existing `go.sum`/proxy cache for the versions already fetched. This was preserved by deliberately *not* running a stripping `go mod tidy` pass after the last `go get` (I used `go get` calls only, letting the module graph settle; I verified via repeated experiments that a full `go mod tidy` at this stage would prune all currently-unimported requires to zero, including the SDK itself, which would be worse for the "pinned exactly" guarantee).
4. Could not test CI on an actual GitHub Actions runner (windows-latest / ubuntu-latest / macos-latest matrix, or the `golangci-lint-action`) — only local verification was possible.

## Fix pass — review findings

Five findings from the reviewer's pass on this task were fixed. `tools/netcheck/netcheck.go` and its test/fixture were **not** touched, per the review's explicit instruction that they were already verified correct.

### 1. (Important) `scripts/check_net.ps1` passed vacuously when `go list` failed

**File:** `scripts/check_net.ps1`

The script previously ran `go list -f '...' ./...` and never inspected `$LASTEXITCODE`. A failing `go list` (compile error, module resolution failure) produced an empty `$Rows`, the offender loop found nothing, and the script printed `[OK]` and exited 0 — a false pass on the one artifact whose job is to catch a violation.

Fixed by checking `$LASTEXITCODE` immediately after both `go list` invocations (`go list -m` and `go list -f ... ./...`) and failing loudly with an ASCII `[!]` diagnostic plus `exit 1` if either fails. Also treat an empty `$Rows` (the full, unfiltered `./...` result) as a failure — `go list` succeeding but returning zero packages means the invocation itself is broken, since `./...` always contains at least `tools/netcheck`. (Note: this empty-check is deliberately applied to the *unfiltered* `$Rows`, not to the `internal/`+`cmd/`-scoped subset introduced for finding 3 — right now, before Task 2, there is legitimately no code under `internal/` or `cmd/` yet, so an empty scoped subset is expected and must still report `[OK]`, not fail.)

One implementation wrinkle surfaced during verification: piping a native command's stdout through a PowerShell cmdlet (e.g. `go list -m | Select-Object -First 1`) under `Set-StrictMode -Version Latest` left `$LASTEXITCODE` completely unset (not `$null`, unset — StrictMode throws `InvalidOperation` on read) in PowerShell 7.6.4. Assigning the native command's output directly (`$ModulePath = go list -m 2>$null`, no pipe to a cmdlet) reproduces normal `$LASTEXITCODE` semantics. Both `scripts/check_net.ps1` and the published copy in `docs/ESTRUTURA.md` §4 use the direct-assignment form.

### 2. (Important) `.github/workflows/ci.yml` pinned `go-version: '1.24'` in three jobs

**File:** `.github/workflows/ci.yml`

Changed all three occurrences (`test`, `netcheck`, `lint` jobs) from `go-version: '1.24'` to `go-version: '1.25'`, matching the `go.mod` floor of `1.25.0` and removing the unpinned toolchain download that `GOTOOLCHAIN=auto` would otherwise trigger on every CI run.

### 3. (Minor) `scripts/check_net.ps1` scanned all packages but claimed a narrower scope

**File:** `scripts/check_net.ps1`

The script ran `go list ... ./...` (every package in the module) while the success message claimed the check covers only `internal/` and `cmd/`. Fixed by deriving the module path via `go list -m`, then filtering `$Rows` to `$Scoped` — import paths equal to or under `$ModulePath/internal` or `$ModulePath/cmd` — before running the offender-detection loop. The success/failure messages were left unchanged (they already stated the correct, narrower scope). The module path is derived rather than hardcoded, since one extra `go list -m` call was cheap enough not to justify a literal.

### 4. (Documentation) Propagated both `check_net.ps1` fixes into `docs/ESTRUTURA.md` §4

**File:** `docs/ESTRUTURA.md`, §4 "Build"

Replaced the published script body (in the "Comandos de desenvolvimento" code block) with the corrected version, identical to `scripts/check_net.ps1`. Added one sentence after the block explaining the new fail-loud/scope-narrowing behavior. Left the existing "Por que `./...` e não `-deps`" explanation untouched — it is still correct, since the script still starts from `go list ./...` before narrowing to `internal/`+`cmd/` for the offender check.

### 5. (Documentation) Corrected the stale Go version floor

Updated `go 1.24` → `go 1.25` / `Go 1.24+` → `Go 1.25+` in:
- `docs/ESTRUTURA.md` §4 — the `go.mod` snippet, plus a new clause explaining the floor is set by `go-sdk@v1.5.0`'s own `go 1.25.0` directive (module graph rules require the main module to declare at least that), not by a language feature the product needs.
- `docs/superpowers/plans/2026-07-25-gobsidian-v01.md` — the "Tech Stack" line and the "Global Constraints" bullet. Other `1.24` occurrences in that file (embedded historical CI-yaml blocks used for step-by-step task instructions) were left untouched — out of the scope named in the review findings.
- `docs/PRD.md` — the `**Linguagem:** Go 1.24+` header line (a floor statement). Left the unrelated `Go 1.24` mention in §6.1 (describing the reference benchmark environment, not a version floor) untouched — not in scope.
- `README.md` — the `Requer Go 1.24+.` installation-prerequisite line.

## Verification (fix pass)

**1. `go test -race ./tools/netcheck/`**
```
ok  	github.com/jonyd/gobsidian/tools/netcheck	(cached)
```
Exit code: 0. Confirms the analyzer and its test were not broken.

**2. `pwsh -NoProfile -File scripts/check_net.ps1` (passing case)**
```
[OK] Nenhum pacote de internal/ ou cmd/ importa rede
```
Exit code: 0.

**3. Proof that finding 1 is fixed — injected-failure run**

Chosen method: temporary Go file under `internal/` importing `net/http`.

```
mkdir -p internal/nettest
cat > internal/nettest/nettest.go << 'EOF'
package nettest

import "net/http"

var _ = http.DefaultClient
EOF
pwsh -NoProfile -File scripts/check_net.ps1
```
Output:
```
WARNING: [!] Pacote do produto importando rede:
    github.com/jonyd/gobsidian/internal/nettest -> net/http
```
Exit code: 1.

Cleanup: `rm -rf internal/`, then `git status --porcelain` showed only the six intentionally-modified tracked files (`.github/workflows/ci.yml`, `README.md`, `docs/ESTRUTURA.md`, `docs/PRD.md`, `docs/superpowers/plans/2026-07-25-gobsidian-v01.md`, `scripts/check_net.ps1`) — no untracked `internal/` entry, confirming the temporary file left no trace. Re-ran the passing case afterward and got `[OK]` / exit 0 again.

**4. `go vet ./...`**
```
(no output)
```
Exit code: 0.

### Files changed in this fix pass

- `scripts/check_net.ps1` (modified)
- `.github/workflows/ci.yml` (modified — 3× `go-version`)
- `docs/ESTRUTURA.md` (modified — §4 script body, go.mod snippet, new explanatory clause)
- `docs/superpowers/plans/2026-07-25-gobsidian-v01.md` (modified — Tech Stack line, Global Constraints bullet)
- `docs/PRD.md` (modified — Linguagem header)
- `README.md` (modified — install prerequisite line)

`tools/netcheck/netcheck.go`, `tools/netcheck/netcheck_test.go`, `tools/netcheck/testdata/` — untouched, as instructed.
