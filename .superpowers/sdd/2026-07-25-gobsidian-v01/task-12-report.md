# Task 12 report: parser types, frontmatter, slug

## Status

DONE

Commit: `5bc655b` — feat(parser): note types, frontmatter split with body offset, anchor slugs

## Files created

- `internal/parser/types.go` — `LinkKind`, `Heading`, `Block`, `Link`, `ParsedNote`
- `internal/parser/slug.go` — `Slug`
- `internal/parser/frontmatter.go` — `SplitFrontmatter`, `DecodeFrontmatter`
- `internal/parser/slug_test.go`
- `internal/parser/frontmatter_test.go`

## Deviation from the brief's verbatim code

The brief's code for `types.go` has no doc comments on the exported types
(`LinkKind`, the `LinkWiki`/`LinkEmbed`/`LinkMarkdown` const block, `Heading`,
`Block`, `Link`, `ParsedNote`). Running `golangci-lint` (revive) against it
produced 6 findings — the repo's standard is zero findings on all three
targets, and other packages in this codebase (e.g. `internal/vault.Vault`)
do carry doc comments on every exported type. I added one Portuguese doc
comment per exported type/const-block, explaining *why* the type exists,
without changing any field, value, or behavior from the brief. After that,
`golangci-lint run ./...` (GOOS=windows) reports `0 issues`.

## go.sum / go.mod

No `go get` was needed. `go.sum` already contained full `h1:` hashes for
both `gopkg.in/yaml.v3 v3.0.1` and `golang.org/x/text v0.40.0` before this
task started (verified with `grep -E "yaml.v3|x/text" go.sum`), and `go
build ./internal/parser/` succeeded on the first try with no changes to
`go.mod` or `go.sum` (confirmed via `git diff go.mod go.sum` — empty). The
`// indirect` markers on those two lines in `go.mod` were left untouched,
since only `go mod tidy` would remove them and that command is forbidden by
project policy; this package now imports them directly but no tool ran to
update the comment, and the build/vet/test suite is fully green regardless.

## Step-by-step

1. Wrote `slug_test.go` and `frontmatter_test.go` verbatim from the brief.
2. Ran `go test ./internal/parser/ -v`: failed with
   `no non-test Go files in ... internal\parser` (package didn't exist yet)
   — equivalent confirmation that tests couldn't pass before implementation.
3. Wrote `types.go`, `slug.go`, `frontmatter.go` verbatim from the brief
   (later added doc comments per the deviation above).
4. `go build ./internal/parser/` succeeded immediately, no `go get` needed.
5. `go test -race ./internal/parser/ -v`: all 4 tests pass (`TestSlug`,
   `TestSplitFrontmatter` with 5 subtests, `TestDecodeFrontmatterPreservesTypes`,
   `TestDecodeFrontmatterMalformedReturnsError`).
6. Fixed the revive findings with doc comments (see deviation above),
   re-ran lint to confirm `0 issues`.
7. Ran full verification suite (see below).
8. Ran 6 mutation-testing rounds proving each test can fail (see below),
   restored originals after each, and diffed against backups to confirm
   byte-for-byte restoration before committing.
9. Committed.

## Verification suite (final, after all mutation restores)

```
$ go clean -testcache && go test -race ./...
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	3.466s
ok  	github.com/jonyd/gobsidian/internal/config	1.756s
ok  	github.com/jonyd/gobsidian/internal/doctor	1.814s
ok  	github.com/jonyd/gobsidian/internal/lifecycle	3.034s
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	3.432s
ok  	github.com/jonyd/gobsidian/internal/parser	1.771s
ok  	github.com/jonyd/gobsidian/internal/service	1.707s
ok  	github.com/jonyd/gobsidian/internal/vault	2.961s
ok  	github.com/jonyd/gobsidian/tools/netcheck	8.565s

$ go vet ./...
(no output — clean)

$ gofmt -l .
(no output — clean)

$ GOOS=linux go vet ./...
(no output — clean)

$ GOOS=darwin go vet ./...
(no output — clean)

$ GOOS=windows golangci-lint run ./...
0 issues.
```

`golangci-lint` (v2.12.2) was available at `/c/Users/jonyd/go/bin/golangci-lint`
and was run with `GOOS=windows` as instructed.

## Mutation proofs

Each mutation was applied to a backup-restored copy, one at a time, run
against the specific test, then reverted (`cp` from a `/tmp` backup taken
before any mutation) and re-verified with a full-package test pass before
moving to the next mutation. Final restoration was confirmed byte-identical
to the backups with `diff`.

### Mutation 1 — `TestSlug`: disable accent stripping

Changed `Slug` to discard the `transform.String` result and use the raw
input instead (kept the call itself to avoid an unused-import build error,
which would have been a false/uninformative failure).

```go
_, _, err := transform.String(
	transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC),
	text,
)
stripped := text // MUTATION: result of accent stripping discarded, using original text
if err != nil {
	stripped = text
}
```

Command: `go test ./internal/parser/ -run TestSlug -v`

```
=== RUN   TestSlug
    slug_test.go:23: Slug("Capítulo 118") = "capítulo 118", quer "capitulo 118"
    slug_test.go:23: Slug("CAPÍTULO 118") = "capítulo 118", quer "capitulo 118"
    slug_test.go:23: Slug("Ação & Reação") = "ação reação", quer "acao reacao"
--- FAIL: TestSlug (0.00s)
FAIL
```

Verdict: CONFIRMED — test catches exactly the mutated behavior (accents
retained instead of stripped).

### Mutation 2 — `TestSlug`: punctuation becomes a space instead of nothing

```go
default:
	// MUTATION: pontuacao agora vira espaco
	if !lastSpace && b.Len() > 0 {
		b.WriteByte(' ')
		lastSpace = true
	}
```

Command: `go test ./internal/parser/ -run TestSlug -v`

```
=== RUN   TestSlug
    slug_test.go:23: Slug("Art. 1.234 — CPC") = "art 1 234 cpc", quer "art 1234 cpc"
--- FAIL: TestSlug (0.00s)
FAIL
```

Verdict: CONFIRMED — catches the specific "punctuation must vanish, not
become a space" rule the comment in the code calls out.

### Mutation 3 — `TestSplitFrontmatter`: off-by-one on the returned body offset

```go
if bytes.Equal(bytes.TrimRight(line, "\r"), fmDelim) {
	fmEnd := int(offset) - (firstNL + 1)
	body := rest[advance:]
	// MUTATION: offset off-by-one
	return data[firstNL+1 : firstNL+1+fmEnd], body, offset + int64(advance) + 1
}
```

Command: `go test ./internal/parser/ -run TestSplitFrontmatter -v`

```
=== RUN   TestSplitFrontmatter/presente
    frontmatter_test.go:60: offset = 18, quer 17
--- FAIL: TestSplitFrontmatter (0.00s)
    --- FAIL: TestSplitFrontmatter/presente (0.00s)
```

Verdict: CONFIRMED — this is exactly the offset bug called out in the brief
as the one that "só aparece em notas que têm frontmatter" and corrupts
`note_read` section offsets.

### Mutation 4 — `TestSplitFrontmatter`: unclosed delimiter treated as closed

```go
// MUTATION: delimitador nao fechado agora e tratado como fechado no fim do arquivo
return data[firstNL+1:], nil, int64(len(data))
```

(replacing the `nil, data, 0` fallback for the unclosed-delimiter case)

Command: `go test ./internal/parser/ -run TestSplitFrontmatter -v`

```
=== RUN   TestSplitFrontmatter/delimitador_nao_fechado
    frontmatter_test.go:54: fm = "title: A\n# Corpo\n", quer ""
    frontmatter_test.go:57: body = "", quer "---\ntitle: A\n# Corpo\n"
--- FAIL: TestSplitFrontmatter (0.00s)
    --- FAIL: TestSplitFrontmatter/delimitador_nao_fechado (0.00s)
```

Verdict: CONFIRMED.

### Mutation 5 — `TestSplitFrontmatter`: delimiter no longer required to be the first line

```go
// MUTATION: nao exige mais que o delimitador de abertura seja a
// primeira linha do arquivo -- aceita "---" em qualquer lugar.
if !bytes.Contains(data, fmDelim) {
	return nil, data, 0
}

firstNL := bytes.IndexByte(data, '\n')
if firstNL < 0 {
	return nil, data, 0
}
```

(dropped the strict "first line must equal exactly `---`" check)

Command: `go test ./internal/parser/ -run TestSplitFrontmatter -v`

```
=== RUN   TestSplitFrontmatter/tres_tracos_no_meio_nao_conta
    frontmatter_test.go:57: body = "nao e frontmatter\n", quer "# Corpo\n---\nnao e frontmatter\n"
--- FAIL: TestSplitFrontmatter (0.00s)
    --- FAIL: TestSplitFrontmatter/tres_tracos_no_meio_nao_conta (0.00s)
```

Verdict: CONFIRMED — this is the specific "three dashes in the middle of the
file doesn't count as frontmatter" guarantee, isolated from the other
subtests (which still pass under this mutation).

### Mutation 6 — `TestDecodeFrontmatterMalformedReturnsError`: swallow the yaml error

```go
var out map[string]any
// MUTATION: erro de unmarshal ignorado
_ = yaml.Unmarshal(fm, &out)
return out, nil
```

Command: `go test ./internal/parser/ -run TestDecodeFrontmatterMalformedReturnsError -v`

```
=== RUN   TestDecodeFrontmatterMalformedReturnsError
    frontmatter_test.go:95: frontmatter malformado deveria devolver erro
--- FAIL: TestDecodeFrontmatterMalformedReturnsError (0.00s)
```

Verdict: CONFIRMED — this is exactly the "malformed frontmatter must be
reported, not silently accepted" guarantee documented in the brief's
comment above the test.

### Mutation 7 — `TestDecodeFrontmatterPreservesTypes`: ignore the actual input

```go
var out map[string]any
// MUTATION: decodifica bytes vazios em vez do frontmatter recebido
if err := yaml.Unmarshal(nil, &out); err != nil {
	return nil, fmt.Errorf("decodificando frontmatter: %w", err)
}
return out, nil
```

Command: `go test ./internal/parser/ -run TestDecodeFrontmatterPreservesTypes -v`

```
=== RUN   TestDecodeFrontmatterPreservesTypes
    frontmatter_test.go:75: titulo = <nil> (<nil>), quer string
    frontmatter_test.go:78: numero = <nil> (<nil>), quer int 42
    frontmatter_test.go:81: ativo = <nil> (<nil>), quer bool true
    frontmatter_test.go:84: tags = <nil> (<nil>), quer lista de 2
    frontmatter_test.go:87: aliases = <nil> (<nil>), quer lista de 2
--- FAIL: TestDecodeFrontmatterPreservesTypes (0.00s)
```

Verdict: CORRECTED IN ROUND 1 — this was reported as CONFIRMED, and that was
wrong. "unmarshal returns nil" zeroes *every* field, so *any* assertion in
the test catches it — the failure output above is real, but it does not
show that this test distinguishes a type-preserving decoder from a
type-collapsing one, which is the guarantee this test is supposed to cover
(it loads `data: 2026-07-25` specifically to check that yaml.v3 hands back
`time.Time`, not a string). Round-1 review caught this: a mutation that
collapses `time.Time` to a formatted string passed the whole suite,
including this test, because nothing asserted the `data` field at all. See
"Round 1 fixes" below for the corrected test and the mutation that now
catches it.

## Restoration verification

```
$ diff /tmp/slug.go.orig internal/parser/slug.go && diff /tmp/frontmatter.go.orig internal/parser/frontmatter.go && echo "FILES_MATCH_ORIGINAL"
FILES_MATCH_ORIGINAL
```

Followed by a fresh `go clean -testcache && go test -race ./...` (see
"Verification suite" above) to confirm no mutation residue remained before
committing.

## Concerns (as of the initial submission — see "Round 1 fixes" below)

Originally reported as "None," with the claim that all seven mutation
proofs "directly target the guarantee each test claims to cover." That
claim was wrong for Mutation 7 (see the corrected verdict above) and round-1
review is what caught it: `TestSplitFrontmatter`'s offset assertion was
guarded by `wantOffset != 0`, so `bodyOffset` was checked in 1 of 5
subtests, and `TestDecodeFrontmatterPreservesTypes` loaded a `time.Time`
fixture without ever asserting it. Both defects trace to the brief's
verbatim test code, not to transcription — see "Round 1 fixes" for what
changed and the new mutation proofs.

---

# Round 1 fixes

Review findings: `.superpowers/sdd/2026-07-25-gobsidian-v01/task-12-findings-round1.md`
Corrected plan commit transcribed from: `a1ec366`

Spec passed; task quality did not, on the deliverable itself: test coverage
force in `internal/parser`. Three items, all confined to
`internal/parser/frontmatter.go` and `internal/parser/frontmatter_test.go`.

## Finding 1 — `bodyOffset` asserted in 1 of 5 subtests

`TestSplitFrontmatter`'s offset check was guarded by `tt.wantOffset != 0`,
so only the `presente` subtest (offset 17) exercised it. The other four
subtests never checked the offset at all — and zero is the *correct*
expected value for three of them (`ausente`, `tres tracos no meio nao
conta`, `delimitador nao fechado`), which is exactly why skipping the
assertion there hides a bug that always returns zero. The `frontmatter
vazio` case (`---\n---\n# Corpo\n`) has a non-zero correct offset (8) and
was not checked at all.

Fix (transcribed from plan commit `a1ec366`): removed the `!= 0` guard,
declared `wantOffset` explicitly on all five subtests (0, 0, 0, 8, and the
existing 17), and asserted unconditionally.

### Mutation proof — offset zeroed specifically on the empty-frontmatter path

```go
if bytes.Equal(bytes.TrimRight(line, "\r"), fmDelim) {
	fmEnd := int(offset) - (firstNL + 1)
	body := rest[advance:]
	if fmEnd == 0 {
		// MUTATION: offset zerado no caso de frontmatter vazio
		return data[firstNL+1 : firstNL+1+fmEnd], body, 0
	}
	return data[firstNL+1 : firstNL+1+fmEnd], body, offset + int64(advance)
}
```

Command: `go test ./internal/parser/ -run TestSplitFrontmatter -v`

```
=== RUN   TestSplitFrontmatter
=== RUN   TestSplitFrontmatter/presente
=== RUN   TestSplitFrontmatter/ausente
=== RUN   TestSplitFrontmatter/tres_tracos_no_meio_nao_conta
=== RUN   TestSplitFrontmatter/delimitador_nao_fechado
=== RUN   TestSplitFrontmatter/frontmatter_vazio
    frontmatter_test.go:70: offset = 0, quer 8
--- FAIL: TestSplitFrontmatter (0.00s)
    --- PASS: TestSplitFrontmatter/presente (0.00s)
    --- PASS: TestSplitFrontmatter/ausente (0.00s)
    --- PASS: TestSplitFrontmatter/tres_tracos_no_meio_nao_conta (0.00s)
    --- PASS: TestSplitFrontmatter/delimitador_nao_fechado (0.00s)
    --- FAIL: TestSplitFrontmatter/frontmatter_vazio (0.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/parser	0.918s
FAIL
```

Verdict: CONFIRMED — fails, naming `frontmatter_vazio` specifically, while
the other four subtests (where zero is the correct value) still pass. This
demonstrates the fixed test now distinguishes "offset correctly computed"
from "offset defaulted to zero," which the pre-fix guard could not do.

Restored via `cp /tmp/frontmatter.go.round1backup internal/parser/frontmatter.go`
and reverified with `diff` (byte-identical) plus a full package test pass.

## Finding 2 — type-preservation test never asserted the date field

`TestDecodeFrontmatterPreservesTypes` loads a fixture ending in
`data: 2026-07-25` but never checks `got["data"]`. `yaml.v3` decodes that
field as `time.Time`; a mutation that formats it into a plain string passes
the whole suite undetected. This is the exact guarantee my original
Mutation 7 claimed to cover but did not: that mutation (`yaml.Unmarshal(nil,
&out)`) zeroed every field, so all five existing assertions failed
together — it never isolated the date/type-preservation guarantee from the
rest.

Fix (transcribed from plan commit `a1ec366`): added `"time"` to the test's
imports and a new assertion `got["data"].(time.Time)` formatted back to
`"2026-07-25"`.

### Mutation proof — `time.Time` collapsed to a formatted string in `DecodeFrontmatter`

```go
var out map[string]any
if err := yaml.Unmarshal(fm, &out); err != nil {
	return nil, fmt.Errorf("decodificando frontmatter: %w", err)
}
// MUTATION: colapsa time.Time em string formatada
for k, v := range out {
	if t, ok := v.(time.Time); ok {
		out[k] = t.Format("2006-01-02")
	}
}
return out, nil
```

Command: `go test ./internal/parser/ -run TestDecodeFrontmatterPreservesTypes -v`

```
=== RUN   TestDecodeFrontmatterPreservesTypes
    frontmatter_test.go:104: data = 2026-07-25 (string), quer time.Time 2026-07-25
--- FAIL: TestDecodeFrontmatterPreservesTypes (0.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/parser	0.641s
FAIL
```

Verdict: CONFIRMED, and this time genuinely isolated — only the `data`
assertion fails; `titulo`, `numero`, `ativo`, `tags`, and `aliases` all
still pass under this mutation, because the mutation only touches
`time.Time` values. That isolation is what makes this a real proof of the
type-preservation guarantee, unlike the original Mutation 7.

Restored via `cp /tmp/frontmatter.go.round1backup internal/parser/frontmatter.go`
and reverified with `diff` (byte-identical) plus a full package test pass.

## Finding 3 — BOM contract undocumented (documentation only, no behavior change)

`SplitFrontmatter` silently finds no frontmatter when input carries a UTF-8
BOM (`\xEF\xBB\xBF---` does not match the `---` prefix check), and the
failure mode is silent, not an error: `FrontmatterErr` never fires,
tags/aliases/title just disappear, and the `---` lines become note body
content. Per the finding, this is not to be fixed in the parser —
`internal/vault.StripBOM` (`internal/vault/eol.go:54`) already owns BOM
stripping, and duplicating it here would create two divergent
implementations. The parser↔vault wiring only happens in Task 19, so the
contract needs to be written down now or whoever wires it has no way to
know.

Fix: added a doc-comment paragraph to `SplitFrontmatter` stating the
precondition (input must already be BOM-free), naming `vault.StripBOM` as
the producer of that guarantee, and describing the silent-failure mode if
the precondition is violated. No behavior changed; the parser still does
not import `internal/vault`.

No mutation proof applies here — it is a comment-only change with no new
executable guarantee to falsify.

## Regression verification (after both test fixes, mutations reverted)

```
$ go clean -testcache && go test -race ./...
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	3.498s
ok  	github.com/jonyd/gobsidian/internal/config	1.717s
ok  	github.com/jonyd/gobsidian/internal/doctor	1.805s
ok  	github.com/jonyd/gobsidian/internal/lifecycle	3.120s
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	3.507s
ok  	github.com/jonyd/gobsidian/internal/parser	1.758s
ok  	github.com/jonyd/gobsidian/internal/service	1.708s
ok  	github.com/jonyd/gobsidian/internal/vault	2.933s
ok  	github.com/jonyd/gobsidian/tools/netcheck	8.813s

$ go vet ./...
(no output — clean)

$ GOOS=linux go vet ./...
(no output — clean)

$ GOOS=darwin go vet ./...
(no output — clean)

$ gofmt -l .
(no output — clean)

$ GOOS=windows golangci-lint run ./...
0 issues.
```

## Scope check

```
$ git status
Changes not staged for commit:
	modified:   internal/parser/frontmatter.go
	modified:   internal/parser/frontmatter_test.go

$ git diff --stat
 internal/parser/frontmatter.go      |  7 +++++
 internal/parser/frontmatter_test.go | 51 ++++++++++++++++++++++++-------------
 2 files changed, 41 insertions(+), 17 deletions(-)
```

Only `internal/parser/` touched, as scoped. The five out-of-scope items
listed in the findings file (trailing whitespace on delimiter lines, `Slug`
merging distinct headings, symbol-only headings, the redundant
`!lastSpace && b.Len() > 0` condition in `slug.go`, and the undocumented
nil-vs-empty contract of `SplitFrontmatter`/`DecodeFrontmatter`) were left
untouched.

## Round 1 status

DONE — see the commit SHA reported to the coordinator alongside this
report; it carries exactly the two-file diff shown in "Scope check" above.
