# Review — Task 181 (white-box tests for `internal/index/persist_codec.go`)

Base 833906f, commit reviewed b2561e0.

## Spec compliance

- **Step 1 (cobertura antes):** report pastes per-function `go tool cover -func` lines for the base commit. Plausible given no `persist_codec_test.go` existed before this commit (persist_codec.go was previously only exercised indirectly through `persist_test.go`'s black-box save/load tests). Not independently re-measured against 833906f (would require checking out that commit, which the review brief forbids via `git checkout`/`restore`); accepted on the strength of the "depois" cross-check below, which I did reproduce byte-for-byte.
- **Step 2 (the tests):** `internal/index/persist_codec_test.go` exists, package `index` (not `index_test` — correct, needed for white-box access to unexported `escritor`/`leitor`). 8 test functions, matching brief's step-2 list. `go test -race -run TestCodec -v` — reran myself, 8/8 PASS (see Commands). Full package `go test -race -count=1 ./internal/index/` — green.
- **Step 3 (mutation proofs):** both `mutate.ps1` invocations reran myself — both exit 0, both show a genuine `--- FAIL` under the mutated code and a restored, hash-verified file afterward. Not proof written in the conditional; real captured output.
- **Step 4 (cobertura depois / ESTADO.md):** reran `go test ./internal/index/ -coverprofile=... && go tool cover -func=... | grep persist_codec.go` myself — the per-function percentages match the report's "depois" table and `docs/ESTADO.md`'s table line for line, including the two lines the report's own commit message highlights (`escritor.value` 65.8%→92.1%, `leitor.value` 61.1%→91.7%). Package total 83.5%→89.7% also reproduced.
- **Step 5 (gate + commit):** `git show --stat b2561e0` — only `docs/ESTADO.md`, `docs/SUGESTOES.md`, `internal/index/persist_codec_test.go`; no production file. Reran `pwsh -File scripts/verify.ps1 -SkipCross -SkipNet` myself — green, 11/11 `[OK]`, same 6 pre-existing skips, ends "Bateria completa. Pode commitar." — output matches the report's pasted transcript.

All five steps done as specified.

## Quality

- **N1 (low-medium, `internal/index/persist_codec.go:295-298` and `:659-660`).** The `valSliceNil` tag pair (nil-typed `[]any` at the frontmatter-value level, the counterpart of `valMapNil`) is at 0% coverage before and after this task — confirmed via `go tool cover` line-level counts (see Commands). `TestCodecValorRoundTripPorTipo`'s `casos` list covers a nil interface, an empty `[]any{}`, and a non-empty `[]any{...}`, but never a typed-nil `[]any` (e.g. `var s []any; any(s)`). The commit message claims "round-trip for every frontmatter value type," which slightly overclaims given this gap — the sibling `valMapNil` branch only got covered incidentally (via a note with nil `Frontmatter`, reached elsewhere in the package's existing black-box tests), not through this task's own round-trip case list. This is not a false pass — the reported coverage numbers are honest and don't claim 100% on `value` — but it is an asymmetry the round-trip test could have closed with one more `casos` entry, the same way `TestCodecStrSliceNilDistintoDeVazio` closes it for `[]string`. Given it's plausible this branch is unreachable from real YAML-decoded frontmatter (same shape of finding as B21), the two acceptable fixes are: add the case, or write it up as a SUGESTOES.md finding the way B21 was. Neither was done. Not a blocker for a test/doc-only task.
- **N2 (nit, `internal/index/persist_codec_test.go`, `TestCodecNotaTruncadaEmCadaByteERecusada`).** The per-prefix-length loop asserts only `l.err == nil` → fail, not `errors.Is(l.err, ErrIndexCacheCorrupted)`. Verified this isn't a false-pass risk: every `leitor` error path is set exclusively through `l.falha`, and every call site wraps `ErrIndexCacheCorrupted` (`grep -n '\.err = \|\.falha('` — confirmed no unwrapped error path exists on the leitor side). So the assertion is weaker than the rubric wanted but not wrong. Cosmetic; add `errors.Is` for rigor.

No other findings. Test bodies match their comments' claims (checked `TestCodecTimeBlobPreservaZona`'s zone-offset claim and `TestCodecStrSliceNilDistintoDeVazio`'s nil-vs-empty claim against the actual `timeBlob`/`strSlice` code — both true, see (e)/(f) below). No duplication with `persist_test.go`'s existing round-trip/corruption tests — those operate at the `SaveIndexCache`/`LoadIndexCache` (black-box, CRC-guarded) layer; this task's tests operate directly on `escritor`/`leitor`, a different layer, and the truncation test specifically bypasses the CRC check that the black-box `TestIndexCacheByteCorruptedRefused` already covers.

## Rulings check

- **R1** (bufio.Writer): honored. All 7 `escritor{w: ...}` constructions in the test file use `bufio.NewWriter(&buf)` + `bw.Flush()` before reading back, not `&escritor{w: &buf}` from the brief's stale sketch.
- **R2** (ESTADO.md placement): honored. No "dívida 1.12" heading anywhere in `docs/ESTADO.md` (grepped — none exists, confirming the ruling's premise). The new coverage block sits directly under the existing `## Formato de cache` header, with per-function lines pasted verbatim and an explicit note that the package-total number is not a computed per-file average ("Não é a média do arquivo — `go tool cover -func` não dá esse número direto").
- **R3** (B21 as a SUGESTOES finding, not a production edit): honored, and the underlying claim is correct — verified independently by mutating the `default` branch's `l.falha(...)` call at `persist_codec.go:685` to `panic("morto")` and running the whole `TestCodec*` family: it passes clean (no panic, no failure), proving the branch is genuinely unreachable through any of these tests — i.e., dead code, exactly as B21 describes. `docs/SUGESTOES.md` B21 entry is accurate and correctly scoped (documented, not fixed).

## Commands I ran

```
$ go build ./...
(clean)

$ go test -race -count=1 ./internal/index/
ok  	github.com/jonyd/gobsidian/internal/index	3.221s

$ go test -race -run TestCodec ./internal/index/ -v   (implicitly reran via the mutate.ps1 harness and standalone; both green, 8/8 PASS)

$ pwsh -File scripts/mutate.ps1 -Path internal/index/persist_codec.go -Anchor 'if profundidade > limiteValorProfund {' -Replacement 'if false {' -Test TestCodecValorProfundidadeAlemDoLimiteERecusada -Package ./internal/index/
--- FAIL: TestCodecValorProfundidadeAlemDoLimiteERecusada (0.00s)
    persist_codec_test.go:148: profundidade 70 devia ser recusada na leitura, tenho <nil>
[OK] internal/index/persist_codec.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0

$ pwsh -File scripts/mutate.ps1 -Path internal/index/persist_codec.go -Anchor 'case valInt64:' -Replacement 'case valInt64 + 100:' -Test TestCodecValorRoundTripPorTipo -Package ./internal/index/
--- FAIL: TestCodecValorRoundTripPorTipo (0.00s)
    persist_codec_test.go:47: lendo 1099511627776: index cache file corrupted: tag de valor desconhecida 3
[OK] internal/index/persist_codec.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0

# (c) B21 verification — mutate the "dead" default branch itself:
$ pwsh -File scripts/mutate.ps1 -Path internal/index/persist_codec.go -Anchor 'l.falha("%w: tag de valor desconhecida %d", ErrIndexCacheCorrupted, tag)' -Replacement 'panic("morto")' -Test TestCodec -Package ./internal/index/
ok  	github.com/jonyd/gobsidian/internal/index	1.779s
[OK] internal/index/persist_codec.go restaurado byte a byte (SHA-256 confere).
[!] O teste PASSOU com a regra mutada.
    TestCodec nao consegue reprovar sem essa regra: ela esta escrita, nao verificada.
EXIT=1
# Confirms B21: the default branch is unreachable — no test in the TestCodec family
# can distinguish panic("morto") from the real code there. This is EXPECTED given
# the branch is dead code, not a defect in the test suite (nothing not-yet-written
# could exercise unreachable code); it corroborates the SUGESTOES.md B21 write-up.

# (d) note-truncation test — reread persist_codec.go's note()/str()/varint()/
# uvarint() bound checks: every field read is strictly bounds-checked and varint
# continuation bits make partial-varint truncation always fail to decode (no byte
# with the continuation bit set can terminate early). So any strict prefix
# b[:i], i < len(b), of a full note encoding cannot decode as a *complete*,
# differently-shaped valid note — structurally, not just empirically. The test's
# `l.err == nil` check (not `errors.Is(l.err, ErrIndexCacheCorrupted)`) is
# sufficient here because every leitor failure path is wrapped through l.falha
# with ErrIndexCacheCorrupted (grep below); no unwrapped error exists on this side.
$ grep -n '\.err = \|\.falha(' internal/index/persist_codec.go
(all leitor-side hits go through l.falha(...ErrIndexCacheCorrupted...); confirmed)

# (e) strSlice nil-vs-empty — read strSlice at :175 (escritor) / :510 (leitor):
# escritor writes varint(-1) for nil vs varint(len) for non-nil (incl. 0); leitor
# returns nil for n<0, else make([]string, n) which is non-nil even at n=0. Real
# distinction, not a same-nil-both-sides false pass. Confirmed by reading code.

# (f) timeBlob zone preservation — read timeBlob at :156 (escritor) / :491 (leitor):
# wraps time.MarshalBinary/UnmarshalBinary (stdlib), which preserves the zone
# OFFSET (not the zone name) through a fixed-offset Location on decode. The test's
# off1/off2 comparison asserts exactly the property the code comment claims
# ("wall clock, fuso e precisao, exatos"); a mutation that dropped/altered the
# offset bytes would surface as an offset mismatch, not just a decode error.

# (g)/(h) checked test comments against actual code behavior (all accurate) and
# grepped persist_test.go for existing round-trip coverage — no duplication; that
# file operates at the SaveIndexCache/LoadIndexCache (CRC-guarded) layer, this
# task's tests operate directly on escritor/leitor.

$ go test ./internal/index/ -coverprofile=$SCRATCH/cov_depois_review.out && \
  go tool cover -func=$SCRATCH/cov_depois_review.out | grep persist_codec.go
(all lines match the report's "depois" table and docs/ESTADO.md's table, verbatim;
 package total 89.7% reproduced)

# coverage-gap check behind N1:
$ grep "persist_codec.go" $SCRATCH/cov_depois_review.out | awk '$3=="0"{print}'
(persist_codec.go:295.15,298.4 2 0   -- escritor []any nil branch, uncovered
 persist_codec.go:659.19,660.20 1 0  -- leitor valSliceNil branch, uncovered
 plus a long tail of io.Writer-error guard clauses, expected uncovered without
 fault injection, out of this task's scope)

$ git show --stat b2561e0
 docs/ESTADO.md                       |  49 +++++++
 docs/SUGESTOES.md                    |  16 +++
 internal/index/persist_codec_test.go | 239 +++++++++++++++++++++++++++++++++++
 3 files changed, 304 insertions(+)
(no production file)

$ pwsh -File scripts/verify.ps1 -SkipCross -SkipNet
[...] 1. go build            [OK]
[...] 2. go test -race       [OK]
[...] 3. testes pulados      [!] 6 pulados (pre-existentes, informativo)
[...] 4. go test (latencia)  [OK]
[...] 5. go vet (windows)    [OK]  (vet cruzado pulado por flag)
[...] 6. gofmt                [OK]
[...] 7. golangci-lint        [OK]
[...] 8. golangci-lint (linux)[OK]  (check_net pulado por flag)
[...] 9. check_tool_params    [OK]
[...] 10. check_doc_refs      [OK]
[...] 11. check_readme_anchors[OK]
[OK] Bateria completa. Pode commitar.
[exited with code 0]
```

## Verdict

**APPROVED_WITH_NITS**

Findings: N1 (low-medium, uncovered `valSliceNil` branch pair, mild commit-message overclaim), N2 (low, missing `errors.Is` assertion in the truncation test, no false-pass risk). Neither blocks the task: coverage numbers are honestly measured and match `docs/ESTADO.md` exactly, both mutation proofs are real, R1–R3 all honored (B21 independently re-verified as genuinely dead code), commit is test/doc-only, and `verify.ps1` reruns green.
