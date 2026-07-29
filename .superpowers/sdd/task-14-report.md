# Task 14 report: goldmark extension for wikilinks and embeds

Commit: `6a6b315` — `feat(parser): goldmark inline parser for wikilinks and embeds`
Base: `c80fc01` (Task 13 complete)

## What was implemented

- `internal/parser/ext_wikilink.go`: `WikilinkNode` AST type, `wikilinkParser`
  (goldmark `InlineParser`) that reads `[[target]]`, `[[target|alias]]`,
  `[[target#anchor]]`, `[[target#^block]]`, `![[target]]`, registered via
  `WikilinkExtension` at inline priority 150.
- `internal/parser/parser.go`: the package facade. `Parse([]byte) (*ParsedNote, error)`
  splits frontmatter, decodes it, extracts headings, runs the goldmark body
  parse, walks the AST, dedupes tags. `md` is a single package-level
  `goldmark.Markdown`, built once with all four extensions.
- `internal/parser/ast.go`: `collect` (the single AST walk that fans nodes out
  into `ParsedNote` fields) plus `tagsFromFrontmatter`, `aliasesFromFrontmatter`,
  `titleFromFrontmatter`, `dedupeTags`.
- `internal/parser/ext_wikilink_test.go`: the brief's three tests, plus one I
  added (see below).
- Stubs for Tasks 15-17 (see "What was stubbed").

## goldmark API vs. the brief

Checked against the pinned `github.com/yuin/goldmark v1.8.4` on disk via `go doc`.
The brief's understanding of `InlineParser`, `WithInlineParsers`, `text.Reader`,
`text.Segment`, `parser.Context`, `util.Prioritized`, and `goldmark.Extender` all
matched exactly — no adaptation needed there. `PeekLine` returns `([]byte, text.Segment)`
where `Segment.Start`/`Segment.Stop` are absolute positions into the reader's
source (the *body*, since `Parse` builds the reader from `body`, not the raw file).
`Trigger()` is consulted per candidate byte the block-level parser hands to the
inline dispatcher, not per byte of the whole buffer — confirmed by reading
`goldmark/parser/parser.go`'s `initSync.Do` (priority tables built once) and the
per-call `Parse` (fresh `root`/`Context`/`blockReader` each call, no other shared
mutable state) — this is also what makes concurrent `Parse` calls safe.

### Mechanical corrections to the brief's code (had to fix to compile)

1. **Field/method name collision.** The brief's `WikilinkNode` declared a field
   `Kind LinkKind` *and* a method `func (n *WikilinkNode) Kind() gast.NodeKind`.
   Go disallows a type having both a field and a method with the same name —
   this does not compile. Renamed the field to `LinkKind`. Updated the two
   places that touched it: the parser's `node.LinkKind = LinkWiki` /
   `LinkEmbed` assignment, and `collect`'s `Kind: node.LinkKind` when building
   `Link{}`.
2. **Stub node types need `Kind()` and `Dump()` too.** The brief says to make
   `BlockIDNode`, `TagNode`, `InlineFieldNode` "minimal structs embedding
   `gast.BaseInline`." `gast.BaseInline`/`BaseNode` do **not** provide `Kind()`
   or `Dump()` — every concrete node implements those itself (confirmed via
   `go doc ast.BaseNode`, which lists neither). Without them, `internal/parser/ast.go`'s
   type switch (`case *BlockIDNode:`) fails to compile: "impossible type switch
   case ... missing method Dump". Added a package-level `gast.NewNodeKind(...)`
   and trivial `Kind()`/`Dump()` to each of the three stub types.

No other deviation. The rest of `ext_wikilink.go`, `parser.go`, `ast.go` is the
brief's code as written, modulo the rename above.

## TDD evidence

RED:

```
$ go test ./internal/parser/ -run TestWikilink -v
# github.com/jonyd/gobsidian/internal/parser_test [github.com/jonyd/gobsidian/internal/parser.test]
internal\parser\ext_wikilink_test.go:31:24: undefined: parser.Parse
internal\parser\ext_wikilink_test.go:76:24: undefined: parser.Parse
internal\parser\ext_wikilink_test.go:90:22: undefined: parser.Parse
internal\parser\ext_wikilink_test.go:110:22: undefined: parser.Parse
FAIL	github.com/jonyd/gobsidian/internal/parser [build failed]
```

(Failed for the expected reason: `parser.Parse` doesn't exist yet.)

After implementing `ext_wikilink.go`, discovering the field/method collision via
`go build ./...`, fixing it, discovering the missing `Kind()`/`Dump()` on stubs,
fixing that too:

GREEN:

```
$ go test -race ./internal/parser/ -run TestWikilink -v
=== RUN   TestWikilinkForms
    ... 9 subtests PASS
=== RUN   TestWikilinkSuppressedInCode
    ... 7 subtests PASS
=== RUN   TestWikilinkOffsetsPointAtSource
--- PASS
=== RUN   TestWikilinkOffsetsWithFrontmatter
--- PASS
PASS
ok  	github.com/jonyd/gobsidian/internal/parser	2.080s
```

That's 13 + 4 = 17 sub-results total (brief said "treze subcasos" counting only
its own three tests' cases; my added test brings the wikilink-specific total to
18 assertions across 4 top-level tests).

Full package, all tasks (12+13+14 tests together), race-enabled:

```
$ go test -race ./internal/parser/ -v
... all PASS (TestWikilink*, TestSplitFrontmatter*, TestDecodeFrontmatter*,
    TestExtractHeadings*, TestSlug)
PASS
ok  	github.com/jonyd/gobsidian/internal/parser	1.845s
```

Whole repo:

```
$ go test -race ./...
ok  	github.com/jonyd/gobsidian/cmd/gobsidian
ok  	github.com/jonyd/gobsidian/internal/config
ok  	github.com/jonyd/gobsidian/internal/doctor
ok  	github.com/jonyd/gobsidian/internal/lifecycle
ok  	github.com/jonyd/gobsidian/internal/mcpsrv
ok  	github.com/jonyd/gobsidian/internal/parser
ok  	github.com/jonyd/gobsidian/internal/service
ok  	github.com/jonyd/gobsidian/internal/vault
ok  	github.com/jonyd/gobsidian/tools/netcheck
```

`go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`
all clean (no output).

## The extra test I added

The brief's `TestWikilinkOffsetsPointAtSource` uses a note with no frontmatter,
so `bodyOffset == 0` — it cannot distinguish "bodyOffset was added" from
"bodyOffset was silently dropped." The task instructions explicitly call this
out and ask to verify by hand. I turned that hand-verification into a
permanent regression test, `TestWikilinkOffsetsWithFrontmatter`, in
`ext_wikilink_test.go`:

```go
func TestWikilinkOffsetsWithFrontmatter(t *testing.T) {
	src := "---\ntitle: x\n---\nabc [[nota]] def\n"
	note, err := parser.Parse([]byte(src))
	...
	l := note.Links[0]
	if got := src[l.Start:l.End]; got != "[[nota]]" {
		t.Errorf(...)
	}
}
```

This fails if `collect`'s `bodyOffset + node.Start` / `bodyOffset + node.End`
addition is ever dropped (e.g. accidentally simplified to just `node.Start`).
I did not add anything else beyond the brief's three tests and the three stubs.

## The nine (eleven, counting sub-cases) extra suppression checks

All run as individual probes against the built `Parse` function, then removed
(no permanent test file left behind for these — see "gap found" note below).

| # | Case | Input | Result | Verdict |
|---|------|-------|--------|---------|
| 1 | Indented code block (4 spaces) | `"    [[nao e link]]\n"` | 0 links | Correctly suppressed |
| 2 | 4-backtick fence containing a 3-backtick line | `` "````\n```\n[[x]]\n```\n````\n" `` | 0 links | Correctly suppressed — goldmark handles this natively, same as Task 13 fixed for headings |
| 3 | `[[` with no closing `]]` before end of line | `"[[abc\ndef\n"` | 0 links | Correctly suppressed |
| 4 | `[[` with no closing `]]` before end of buffer | `"[[abc"` | 0 links | Correctly suppressed |
| 5 | Wikilink spanning two lines | `"[[abc\ndef]]\n"` | 0 links | Correctly suppressed — matches the design comment "um wikilink nao atravessa linha" |
| 6 | `[[[triplo]]]` | `"[[[triplo]]]\n"` | **1 link**, `Target="[triplo"` (leading bracket leaks into Target), `Raw="[[[triplo]]"` | **Real gap, flagged below — not fixed** |
| 7 | Empty target `[[]]` | `"[[]]\n"` | 0 links | Correctly suppressed (`target=="" && anchor==""` guard) |
| 8 | `[[|alias]]` | `"[[|alias]]\n"` | 0 links | Correctly suppressed, same guard (empty target with only an alias is treated as no link) |
| 9 | Inside a blockquote | `"> [[nota]]\n"` | 1 link, correct fields | Correctly found |
| 10 | Inside a list item | `"- [[nota]]\n"` | 1 link, correct fields | Correctly found |
| 11 | `![[x]]` with the `!` preceded by a backslash | `"\\![[x]]\n"` | 1 link, `Kind=wikilink` (not embed), `Raw="[[x]]"` | Correct — CommonMark's backslash-escape parser consumes `\!` into a literal-text node before our trigger ever sees a `!`, so the wikilink parser only ever sees `[[x]]` and correctly reports it as a plain wikilink, not an embed |

### Gap found: `[[[triplo]]]` — not fixed, flagged for follow-up

With three opening brackets, `wikilinkParser.Trigger` fires at the very first
`[` (offset 0). It finds `[[` at positions 0-1, then searches for the first
`]]` after that, which is at the end of the string. The middle `[` (position 2)
ends up folded into `inner`, so `target` becomes `"[triplo"` instead of
`"triplo"` or a clean suppression. `Raw` is preserved faithfully for whatever
byte range was consumed (`"[[[triplo]]"`, 11 bytes, leaving a stray trailing
`]`), so `note_move` rewriting from `Raw` would still round-trip correctly —
the defect is only in `Target`, which would fail to resolve against any real
note title (a silently-broken link, not a silently-wrong one pointing at the
wrong note).

I did not add a test pinning this behavior, and did not attempt a fix, for two
reasons: (1) I have no reference for what real Obsidian actually renders for
triple-bracketed wikilinks — a plausible first-`]]`-match tokenizer (which is
what Obsidian's own parser likely is) would produce the *same* malformed
target, in which case "fixing" it would be diverging from parity rather than
correcting a bug; (2) the task brief's suppression table doesn't cover this
shape, and Task 25's Obsidian-plugin parity run is the actual authority here.
Recommend: capture as a parity question for Task 25, and only add handling if
that run shows Obsidian treats it differently (e.g. suppressing entirely, or
treating the extra bracket as literal leading text).

## Fix pass — review findings

Base: `dcac49b`. A reviewer confirmed the twelve-context suppression argument
and the positive-context/offset/`splitWikilink`/concurrency claims as sound
(no changes made there) and found five real defects plus three minor items.
This section documents the fixes, transcribed from the plan's Task 14 section
(`docs/superpowers/plans/2026-07-25-gobsidian-v01.md`) and the Task 12
`types.go`/`offsetUnknown` block, with no mechanical corrections needed — the
plan compiled as written against the already-renamed `LinkKind`/`closeIdx`.

### Finding 1 — triple-bracket guessing destroyed a real Markdown link

`wikilinkParser.Parse` now declines when the third byte after `[[` is also
`[` (`internal/parser/ext_wikilink.go:72-74`), instead of guessing where the
wikilink starts. Declining makes goldmark re-offer the `[`/`!` trigger one
byte later, where `[[x]]` is unambiguous.

Before the fix, `[[[a]] b](d.md)` produced one spurious wikilink
(`Target="[a"`) and the real Markdown link to `d.md` was gone entirely. After:

```
"[[[a]] b](d.md)\n"
  target="d.md" kind=markdown raw="d.md" start=-1 end=-1
  target="a"    kind=wikilink raw="[[a]]" start=1 end=6
```

`d.md` is present. The wikilink recovered from the retried trigger position
is a plausible side-effect the finding explicitly leaves open ("what
`[[[x]]]` should produce is a parity question for Task 25") — the only hard
requirement, that `d.md` not be lost, is met.

### Finding 2 — offsetUnknown sentinel

`internal/parser/types.go`: added `const offsetUnknown int64 = -1` and
extended the `Link.Start`/`Link.End` doc comment to state the sentinel and
why (mirrors `config.Flags`' `ReadOnlySet`/`DebounceMSSet` pattern — zero is
a legitimate position, so it can't double as "unknown"). `*gast.Link` and
`*gast.Image` in `collect` (`internal/parser/ast.go`) now set
`Start: offsetUnknown, End: offsetUnknown` explicitly instead of leaving the
struct's zero value implicit.

### Finding 3 — Markdown images invisible to the graph

Added a `case *gast.Image:` arm in `collect` (`internal/parser/ast.go`),
collecting it as `Kind: LinkEmbed` with `offsetUnknown` offsets — same kind as
`![[x]]`, since it's the same concept in a different spelling.

```
"![alt](diagrama.png)\n"   -> target=diagrama.png kind=embed raw=diagrama.png start=-1 end=-1
"![[diagrama.png]]\n"      -> target=diagrama.png kind=embed raw=![[diagrama.png]] start=0 end=17
```

Both yield a link, both `LinkEmbed`.

### Finding 4 — two dead guards

Added to `TestWikilinkSuppressedInCode`
(`internal/parser/ext_wikilink_test.go`): `"[[]]\n"`, `"[[   ]]\n"`,
`"[[|apenas]]\n"` (from the plan, covering the empty-target guard) plus one
I had to construct myself: `"[[x[[]]\n"`, covering the nested-`[[` guard.

The plan's three degenerate rows do not exercise the nested-`[[` guard: I
verified by hand-tracing and then empirically that inputs like `"[[a[[b]]\n"`
produce exactly one link *whether or not* the nested-`[[` guard exists —
declining at the outer position just makes goldmark retry at the inner `[[`
and find the *same* closing `]]`, producing a different `Target` but the same
link count. A count-based ("quer nenhum") assertion structurally cannot catch
that. `"[[x[[]]\n"` works because the retried inner position (`[[]]`) hits the
*empty*-target guard instead, so with both guards intact the whole line
suppresses to zero links; with only the nested-`[[` guard removed, the outer
attempt captures `Target="x[["` (non-empty, so the empty-target guard doesn't
save it) and produces one spurious link. See the mutation table below.

### Finding 5 — Raw byte-exactness

Added `{"espacos interiores", "[[ nota ]]", "nota", "", "", parser.LinkWiki}`
to `TestWikilinkForms` (from the plan). Proven below to catch a reconstructing
implementation.

### Minor items

- `splitWikilink`/`trimSpace`: converted from `bytes.IndexByte([]byte(inner), ...)`/
  `bytes.TrimSpace([]byte(s))` to `strings.IndexByte`/`strings.TrimSpace`
  operating directly on the string — no more `string`→`[]byte`→`string`
  round-trips on the indexing path. `trimSpace` helper removed (now just
  `strings.TrimSpace` inline).
- `WikilinkNode.Dump` now includes `"LinkKind": n.LinkKind.String()`.
- `KindWikilink` → `kindWikilink` (unexported), matching `kindBlockID`,
  `kindTag`, `kindInlineField`. Confirmed no references outside the package
  (`grep -rn KindWikilink` only matched the plan doc and the file itself).

### Mutation table

Each mutant applied to the just-committed code, tested, then reverted before
the next mutant (verified green after each revert, `-count=1`).

| Mutant | Change | Subtests failing | Verdict |
|---|---|---|---|
| Nested-`[[` guard removed | `bytes.ContainsAny(inner, "\n") \|\| bytes.Contains(inner, []byte("[["))` → `bytes.ContainsAny(inner, "\n")` | `TestWikilinkSuppressedInCode/colchete_duplo_aninhado_antes_do_fechamento` | Caught |
| Empty-target guard removed | deleted `if target == "" && anchor == "" { return nil }` | `TestWikilinkSuppressedInCode/alvo_vazio`, `/alvo_so_espacos`, `/so_alias,_sem_alvo`, `/colchete_duplo_aninhado_antes_do_fechamento` | Caught (4 subtests) |
| `Raw` reconstructed from parts instead of byte-sliced | `raw := "[[" + target + (...anchor/alias...) + "]]"`, prefixed `"!"` for embed | `TestWikilinkForms/espacos_interiores` only | Caught |

After each mutant, `go build ./...` and `go test ./internal/parser/ -run TestWikilink -v -count=1` were re-run and confirmed green post-revert.

### Sentinel confirmation

```
abc [[nota]] def        kind=wikilink  Start=4  End=12   (src[4:12] == "[[nota]]")
texto [texto](d.md)     kind=markdown  Start=-1 End=-1
texto ![alt](d.png)     kind=embed     Start=-1 End=-1
```

### Final verification

```
$ go test -race ./internal/parser/ -v -count=1
... all PASS, 4 TestWikilink* top-level tests / 26 subtests total, plus
    TestSplitFrontmatter*, TestDecodeFrontmatter*, TestExtractHeadings*, TestSlug
ok  	github.com/jonyd/gobsidian/internal/parser	2.088s

$ go test -race ./...
ok  	github.com/jonyd/gobsidian/cmd/gobsidian
ok  	github.com/jonyd/gobsidian/internal/config
ok  	github.com/jonyd/gobsidian/internal/doctor
ok  	github.com/jonyd/gobsidian/internal/lifecycle
ok  	github.com/jonyd/gobsidian/internal/mcpsrv
ok  	github.com/jonyd/gobsidian/internal/parser
ok  	github.com/jonyd/gobsidian/internal/service
ok  	github.com/jonyd/gobsidian/internal/vault
ok  	github.com/jonyd/gobsidian/tools/netcheck
```

`go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`
all clean. `git status --porcelain` after commit: clean (only the four scoped
files touched: `internal/parser/ext_wikilink.go`, `ast.go`, `types.go`,
`ext_wikilink_test.go`).

### Concern carried forward

The nested-`[[` guard's dead-to-the-suite problem is structural, not just a
missing test row: any suppression test built from an input containing a
nested `[[` where the *first* `]]` found is non-empty will pass whether or not
the guard exists, because goldmark's per-byte retry recovers the same link
from the inner position. `"[[x[[]]\n"` is the one shape that isolates it
(nested attempt lands on the empty-target guard instead). Worth a comment in
the plan itself if Task 15-17's block-ID/tag/inline-field guards have an
analogous shape, so the same coverage gap doesn't recur silently there.

## What was stubbed for Tasks 15-17

Per the brief's Step 5, minimal stand-ins so the package compiles; each lives
in the file its real Task 15/16/17 implementation will replace (per the plan's
own `Files: Create` lines for those tasks):

- `internal/parser/ext_blockid.go`: `BlockIDNode{ID string; Start, End int64}`
  embedding `gast.BaseInline`, `BlockIDExtension{}` with a no-op `Extend`.
- `internal/parser/ext_tag.go`: `TagNode{Name string}` embedding `gast.BaseInline`,
  `TagExtension{}` with a no-op `Extend`.
- `internal/parser/ext_inline_field.go`: `InlineFieldNode{Key, Value string}`
  embedding `gast.BaseInline`, `InlineFieldExtension{}` with a no-op `Extend`.

Each stub node type also got a package-level `gast.NewNodeKind(...)` and
trivial `Kind()`/`Dump()` methods — required for `gast.Node` interface
satisfaction (see "Mechanical corrections" above), not present in the brief's
literal text but necessary for the type switch in `ast.go` to compile.

## Files changed

- `internal/parser/ext_wikilink.go` (new)
- `internal/parser/ext_wikilink_test.go` (new)
- `internal/parser/parser.go` (new)
- `internal/parser/ast.go` (new)
- `internal/parser/ext_blockid.go` (new, stub)
- `internal/parser/ext_tag.go` (new, stub)
- `internal/parser/ext_inline_field.go` (new, stub)

`go.mod`/`go.sum`: untouched. `git status --porcelain` confirmed after build —
no `go.sum` gap surfaced for goldmark's transitive dependencies.

## Self-review

- **Completeness:** all nine forms in the brief's `TestWikilinkForms` table
  pass; `Raw` is preserved byte-for-byte as authored in every case (verified
  by the table's own assertion, not just spot-checked).
- **Correctness:** `src[l.Start:l.End]` slices exactly the link text both with
  and without frontmatter (`TestWikilinkOffsetsPointAtSource` and the
  `TestWikilinkOffsetsWithFrontmatter` I added) — confirms `bodyOffset` really
  is being added in `collect`, not just declared as a parameter.
- **Concurrency:** confirmed by reading goldmark's own `parser.go`:
  `(*parser).Parse` builds its priority-sorted parser tables exactly once
  behind `sync.Once`, then each call works only with locally-created
  `root`/`Context`/`blockReader`. Our `wikilinkParser` is a stateless
  zero-field struct, one instance shared across calls. `md` is constructed
  once at package-var-init time (single goroutine, before any worker pool
  exists), so extension registration is never racing a `Parse` call.
- **Discipline:** the only thing added beyond the brief's three tests and
  three stubs is `TestWikilinkOffsetsWithFrontmatter`, which encodes a
  correctness property the brief explicitly told me to hand-verify — kept as
  a permanent regression rather than a one-off check. The nine-plus
  suppression probes were exploratory (temporary file, deleted before commit)
  and are not part of the committed diff.
- **Testing hygiene:** `go test -race ./internal/parser/ -v` output is clean;
  no skipped tests, no stray fmt.Println/debug output left in committed code.

## Concerns for the controller / next task

1. The brief's own snippet had a compile-breaking field/method collision
   (`Kind` field vs. `Kind()` method) — worth fixing in the plan document
   itself before Task 15's implementer transcribes the same node-definition
   pattern for `BlockIDNode` (their stub is a *different* field set, so it
   doesn't inherit the bug, but the pattern — "embed BaseInline, add a `Kind`
   field" — is exactly what would reintroduce it if copied carelessly).
2. `[[[triplo]]]` is a real anomaly (see above) — recommend carrying it as a
   named question into Task 25's parity checklist rather than silently
   forgetting it.
3. The `*gast.Link` case in `collect` (standard Markdown links) never sets
   `Start`/`End` — they default to 0. This is exactly what the brief's own
   `ast.go` snippet specifies, so I transcribed it faithfully rather than
   inventing offset-tracking the brief didn't ask for, but it means
   `note_move` cannot safely rewrite the offsets of a standard `[text](url)`
   link today. Flagging since offset correctness was this task's entire
   point for wikilinks; whichever task is meant to close this for Markdown
   links should know it's still open.
