# Task 15 report — block-id goldmark extension

## What was implemented

`internal/parser/ext_blockid.go` replaces the M1 stub with a real
goldmark inline extension:

- `kindBlockID` / `BlockIDNode` — same shape as `WikilinkNode` from Task 14:
  `gast.BaseInline`, `ID string`, `Start`/`End int64`, `Kind()`, `Dump()`.
- `blockIDParser` — `Trigger() []byte{'^'}`. `Parse` runs entirely on the
  current line (`block.PeekLine()`), never crosses lines, and never tracks
  fence state — code contexts are already suppressed by goldmark before the
  parser is ever offered the trigger.
- `BlockIDExtension.Extend` registers it via
  `util.Prioritized(&blockIDParser{}, 100)`, the priority named in the brief.
  No known collision with wikilink (150) or CommonMark's link parser (200):
  none of them trigger on `'^'`.

### Rules implemented (brief Step 3, numbered to match)

1. **End-of-line only.** After the candidate id, every remaining byte on the
   line must be `' '`, `'\t'`, `'\r'`, or `'\n'` — anything else means the
   `^` was mid-line text, not a marker.
2. **Alphabet.** `blockIDChar` accepts `[A-Za-z0-9-]` only; the id-scan loop
   stops at the first byte outside that set. An empty scan (`^` immediately
   followed by a non-alphabet byte, including newline) is rejected.
3. **`Start` is the parent block's start, not the caret's.** Reached via
   `parent.Lines().At(0).Start`. This works because of how goldmark's inline
   parser is invoked — verified in `parser/parser.go@v1.8.4`:
   `p.walkBlock(root, func(node) { p.parseBlock(blockReader, node, pc) })`
   calls `parseBlock` on **every** node, but `parseBlock` immediately does
   `block.Reset(parent.Lines())`; if `Lines()` is empty, `PeekLine` returns
   `nil` and no inline parser — including this one — is ever offered a
   trigger. So by the time `blockIDParser.Parse` runs, `parent` is
   necessarily the *leaf* block that owns the line the `^` sits on:
   `Paragraph`, `TextBlock` (tight list items), or whatever block type wraps
   that text — with list markers and blockquote `"> "` prefixes already
   stripped by their block parsers, because those parsers write `Lines()`
   segments that start after the prefix. `ast.Node.Lines()` is part of the
   base `Node` interface (valid for block nodes), so no type assertion is
   needed on `parent`.
4. **Code contexts.** Not handled by this file at all — confirmed by the
   `dentro de codigo` subtest and the inline-code-span test both passing
   with zero blocks, with no fence-tracking code written.

`collect` in `ast.go` already had a `case *BlockIDNode` from Task 14's stub
work, adding `bodyOffset` to both `Start` and `End` — untouched by this task,
confirmed still correct by `TestBlockIDExtraction`'s exact-slice assertion.

`offsetUnknown` (-1) is not used: every path either returns a real
`Start`/`End` pair or returns `nil` (no node at all). The one place a `nil`
`Lines()` could theoretically appear is guarded (return `nil`, i.e. "no
marker here") rather than indexing out of bounds — reachability analysis
above says this can't actually happen, but the guard costs nothing and turns
a would-be panic into "not a marker" if goldmark's invariant ever changes.

## TDD Evidence

**RED** — `go test ./internal/parser/ -run TestBlockID -v`, run against the
stub before touching `ext_blockid.go`:

```
=== RUN   TestBlockIDExtraction
    ext_blockid_test.go:17: blocos = 0, quer 2: []
--- FAIL: TestBlockIDExtraction (0.00s)
=== RUN   TestBlockIDRejectsNonTerminal
--- PASS: TestBlockIDRejectsNonTerminal (0.00s)   # vacuously — stub never produces nodes
=== RUN   TestBlockIDListItem
    ext_blockid_test.go:64: blocos = 0, quer 2: []
--- FAIL: TestBlockIDListItem
=== RUN   TestBlockIDBlockquote
    ext_blockid_test.go:94: blocos = 0, quer 1: []
--- FAIL: TestBlockIDBlockquote
=== RUN   TestBlockIDMultiplePerParagraph
--- FAIL
=== RUN   TestBlockIDDuplicateAcrossNote
--- FAIL
=== RUN   TestBlockIDMultilineParagraph
--- FAIL
=== RUN   TestBlockIDTrailingSpaces
--- FAIL
=== RUN   TestBlockIDCharset
--- FAIL (all three subcases)
=== RUN   TestBlockIDCaretAtLineStart
--- FAIL
=== RUN   TestBlockIDInlineCodeSpan
--- PASS   # vacuously
FAIL
```

Every failure is "blocos = 0, quer N" — the expected reason (stub produces
no nodes). `TestBlockIDRejectsNonTerminal` and the code-span test pass
vacuously against the stub, which is fine: they assert absence, and the stub
guarantees absence for the wrong reason. They get re-verified as real
negatives once the real parser exists (see GREEN).

**GREEN** — `go test -race ./internal/parser/ -run TestBlockID -v` after
implementing `ext_blockid.go`:

```
--- PASS: TestBlockIDExtraction (0.00s)
--- PASS: TestBlockIDRejectsNonTerminal (0.00s)
    --- PASS: TestBlockIDRejectsNonTerminal/no_meio_da_linha (0.00s)
    --- PASS: TestBlockIDRejectsNonTerminal/dentro_de_codigo (0.00s)
    --- PASS: TestBlockIDRejectsNonTerminal/circunflexo_sozinho (0.00s)
    --- PASS: TestBlockIDRejectsNonTerminal/caracteres_invalidos (0.00s)
--- PASS: TestBlockIDListItem (0.00s)
--- PASS: TestBlockIDBlockquote (0.00s)
--- PASS: TestBlockIDMultiplePerParagraph (0.00s)
--- PASS: TestBlockIDDuplicateAcrossNote (0.00s)
--- PASS: TestBlockIDMultilineParagraph (0.00s)
--- PASS: TestBlockIDTrailingSpaces (0.00s)
--- PASS: TestBlockIDCharset (0.00s)
    --- PASS: TestBlockIDCharset/maiusculas (0.00s)
    --- PASS: TestBlockIDCharset/hifen (0.00s)
    --- PASS: TestBlockIDCharset/so_digitos (0.00s)
--- PASS: TestBlockIDCaretAtLineStart (0.00s)
--- PASS: TestBlockIDInlineCodeSpan (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/parser	1.755s
```

Full package (`go test -race ./internal/parser/ -v`) and full module
(`go build ./... && go test -race ./...`) both pass — Tasks 12–14's tests
unaffected.

`go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet
./...` all produced no output (clean).

## The eleven extra checks

| # | Check | Result |
|---|-------|--------|
| 1 | Block id on a **list item** | `Start` = start of the item's text, *after* the `"- "` marker. `parent` is the `Paragraph`/`TextBlock` goldmark builds for the item's content; the marker is block syntax the list's block-parser already consumed before `Lines()` was set. Test: `TestBlockIDListItem`, first block: `"primeiro item ^item1"` (no leading `"- "`). |
| 2 | Block id on a **nested list item** | Same rule, one level deeper: `Start` is still the nested item's own text start, not the outer item's, not the document start. Test: `TestBlockIDListItem`, second block: `"filho aninhado ^filho1"`. |
| 3 | Block id inside a **blockquote** | `Start` excludes the `"> "` prefix — the blockquote's block parser strips it before the inner paragraph's `Lines()` is recorded. Test: `TestBlockIDBlockquote`: slice = `"uma citacao ^cite1"`, not `"> uma citacao ^cite1"`. |
| 4 | Block id on a **table row** | **No GFM table extension is registered** in `parser.go` (`md := goldmark.New(goldmark.WithExtensions(WikilinkExtension{}, BlockIDExtension{}, TagExtension{}, InlineFieldExtension{}))` — no `extension.Table`/`extension.GFM`). Pipe syntax is inert; consecutive non-blank lines merge into one plain paragraph. Verified with a throwaway test (not committed): `"| a | b |\n| - | - |\n| x | y ^row1 |\n"` → **zero blocks**, because the id is followed by `" |"` before the newline, which fails rule 1 (end-of-line-only) exactly as intended. Moving the id to the true end of the last line (`"...y ^row1\n"`, no trailing `" |"`) does produce a block, but `Start` covers the *entire* multi-line paragraph (all three source lines), since without the table extension there is no row boundary for the parser to know about. This is a real limitation worth flagging for whoever adds table support later, not a bug in this task's scope. |
| 5 | Block id on a **fenced block's closing line** | Verified with a throwaway test: `"```\ncode\n```^abc123\nmore text\n"` → zero blocks. CommonMark requires a closing fence line to contain only fence characters plus optional whitespace; `` ```^abc123 `` has trailing non-whitespace, so it is **not** a valid closing fence. The code block never closes, so everything through EOF (including `"more text"`) stays raw code content, and no inline parser — including this one — is ever offered. Confirmed as the correct/expected CommonMark behavior, not a gap. |
| 6 | **Two block ids in the same paragraph** | Both extracted independently, same `ID` order as source, and both report the **same** `Start` (the shared paragraph's first line) since they belong to the same block. Test: `TestBlockIDMultiplePerParagraph`. |
| 7 | **Same id used twice in a note** | Produces two separate `Block` entries with the same `ID` and different `Start`/`End` — the parser does not deduplicate; that's correctly left to a higher layer. Test: `TestBlockIDDuplicateAcrossNote`. |
| 8 | Paragraph **spans several lines** before the marker | `Start` reaches the paragraph's first source line, not the line the `^` is on. Test: `TestBlockIDMultilineParagraph`: 3-line paragraph, slice = all three lines through the marker. |
| 9 | `^abc123` followed by **trailing spaces** | Accepted (trailing spaces satisfy the end-of-line rule); `End` stops right after the id, excluding the spaces. Test: `TestBlockIDTrailingSpaces`. |
| 10 | **Uppercase**, **hyphen**, **digits-only** ids | All three accepted verbatim (`ABC123`, `abc-123`, `123456`) — matches the `[A-Za-z0-9-]` alphabet exactly. Test: `TestBlockIDCharset` (3 subcases). |
| 11 | `^` at the **very start of a line**, nothing before it | Accepted; no rule requires preceding text. Test: `TestBlockIDCaretAtLineStart`: `"^soleiro\n"` → one block, slice = `"^soleiro"`. |

(The brief's own two "must produce nothing" cases — fenced code block and
inline code span — are covered by `TestBlockIDRejectsNonTerminal/dentro_de_codigo`
and the added `TestBlockIDInlineCodeSpan`.)

Real gaps found and turned into permanent tests: none of the eleven revealed
a parser defect. Item 4 (table row) surfaced a **pre-existing, documented
absence** (no table extension registered) rather than a bug in this task —
reported above, not added as a committed test, since it would be testing an
absence that belongs to a future table-extension task, not this one.

## Files changed

- `internal/parser/ext_blockid.go` — real implementation, replaces the stub.
- `internal/parser/ext_blockid_test.go` — brief's two required tests plus
  nine more covering the checks above (`TestBlockIDListItem`,
  `TestBlockIDBlockquote`, `TestBlockIDMultiplePerParagraph`,
  `TestBlockIDDuplicateAcrossNote`, `TestBlockIDMultilineParagraph`,
  `TestBlockIDTrailingSpaces`, `TestBlockIDCharset`,
  `TestBlockIDCaretAtLineStart`, `TestBlockIDInlineCodeSpan`).

Commit: `7fb623f feat(parser): block id extraction with block-level offsets`.

## Self-review

- **Correctness:** every test asserts `body[b.Start:b.End]` (or the
  package's own equivalent slicing) against an exact expected substring —
  not just presence of the marker, not the whole document. Checked by hand
  for the list-item and blockquote cases specifically, since those are where
  a wrong `Start` would silently produce a block that includes list/quote
  syntax it shouldn't.
- **Purity:** `blockIDParser` is a stateless `struct{}`; `Parse` only reads
  from `block`/`parent` and returns a new node — no package-level mutable
  state, no I/O, no `ctx`. `BlockIDExtension.Extend` only registers options
  on the `goldmark.Markdown` passed in, matching `WikilinkExtension`'s
  shape exactly.
- **Discipline:** nothing beyond the four brief rules was implemented — no
  speculative handling of CRLF beyond treating `'\r'` as a blank trailing
  byte (needed anyway since `PeekLine` can include it), no table-extension
  workaround, no dedup logic. The scratch throwaway test file used to probe
  checks 4 and 5 was deleted before committing; `git status` after `git add`
  showed only the two in-scope files.
- **Testing:** ran with `-race`; output is clean pass/fail with no stray
  prints. `gofmt -l .` and cross-`GOOS` `go vet` are clean.

## Concerns

- Item 4's finding (no GFM table extension registered) is out of this
  task's scope but worth a note for whoever eventually adds table support:
  a block id "on a table row" today behaves as part of whatever plain
  paragraph the pipe-delimited lines collapse into, with `Start` spanning
  every merged line, not just that row. Not a defect in Task 15 — the
  extension is behaving correctly given the markdown that's actually being
  parsed — but it will need re-verification once `extension.Table` (or
  equivalent) lands.

## Fix pass — combined 15/16/17 review

Starting commit `9116b0c`. Scope: `ext_blockid.go`, `ext_blockid_test.go`,
`ext_tag.go`, `ext_tag_test.go`, `ext_inline_field.go`,
`ext_inline_field_test.go`, `types.go` (`Block.Start` doc comment only).

### Critical 1 — inline fields were deleting links

`parseFromColon` and `parseFromBracket` used to consume the whole value span
(`block.Advance(len(line))` / up to and including the closing `]`), so no
other inline parser ever saw a wikilink, embed, or markdown link that lived
inside a field's value. Fix: read the value via the already-peeked `line`
to record on the node, but `Advance` only past `chave::` (2 bytes) in the
colon form, and past `[chave::` (`i+2` bytes) in the bracket form. goldmark
then parses the rest of the line normally.

Evidence (`Parse` output, `Inline` and `Links`):

```
"fonte:: [[STJ]]\n"
  Inline: map[fonte:[[[STJ]]]]
  Links:  [{Raw:"[[STJ]]" Target:"STJ" Kind:wikilink(0) Start:8 End:15}]

"capa:: ![[img.png]]\n"
  Inline: map[capa:[![[img.png]]]]
  Links:  [{Raw:"![[img.png]]" Target:"img.png" Kind:embed(1) Start:7 End:19}]

"fonte:: [STJ](stj.md)\n"
  Inline: map[fonte:[[STJ](stj.md)]]
  Links:  [{Raw:"stj.md" Target:"stj.md" Alias:"STJ" Kind:markdown(2) Start:-1 End:-1}]
```

All three table rows from the review now produce both the recorded value
and the link.

### Critical 2 — bracketed field ate a neighbouring Markdown link

`[Nota:: veja](destino.md)` was matched as a field at priority 130, before
CommonMark's link parser (~200) ever saw the `[`. Fix: after finding the
closing `]` (now via depth-tracking, see minor item below), decline if the
byte right after it is `(` or `[`, so the standard link parser gets first
crack at the whole span.

Evidence:

```
"[Nota:: veja](destino.md)\n"
  Links: [{Raw:"destino.md" Target:"destino.md" Alias:"Nota veja" Kind:markdown(2) Start:-1 End:-1}]
```

The markdown link to `destino.md` is present. (A side effect: the colon-form
field parser still fires independently on the `::` inside the still-open
link label, producing a spurious `Inline["Nota"] = ["veja](destino.md)"]`
entry — see new minor item below. It does not affect the link.)

### Critical 3 — block id ranges spanned lines the user never referenced

`Start` came from the block's first line, `End` from the marker's line, with
no check that the marker's line was the block's last line. Fix: reject a
`^id` whose line is not `lines.At(lines.Len()-1)`.

Evidence:

```
"linha um\nlinha dois ^b\nlinha tres ^c\n"
  Blocks: 1
    ID="c" Start=0 End=36 text="linha um\nlinha dois ^b\nlinha tres ^c"
```

One block, not two overlapping ones; `^b` (mid-paragraph) is rejected and
falls through as literal text. `TestBlockIDMultiplePerParagraph` was
rewritten to assert this rejection instead of the old overlapping-range
behaviour; added `TestBlockIDThreeLineParagraphOnlyLastMarkerCounts` for the
exact three-line case from the review.

### Important 4 — tags adjacent to emphasis markers

`tagPrecedingOK` now also admits `*`, `_`, `~` as valid preceding characters
(they are Unicode `Po`/`Pc`, not opening punctuation, so they were excluded
before).

Evidence:

```
tags("**#civil**\n") = [civil]
tags("*#a*\n")        = [a]
```

New test `TestTagAdjacentToEmphasisMarkers` covers bold/italic(`*`)/
italic(`_`)/strikethrough. The `_` case documents an orthogonal,
pre-existing alphabet collision: `_` is also a legal tag body character, so
`_#civil_` yields tag `civil_` (trailing `_` consumed), not `civil` — that
is correct given the existing alphabet, not a regression from this fix.

### Important 5 — empty hierarchy segments

Added `collapseTagSlashes`: splits on `/`, drops empty segments, rejoins.
Applied to the tag name before creating `TagNode`, in addition to the
existing trailing-slash trim on the consumed byte range (unchanged).

Evidence:

```
tags("#a//b\n")   = [a/b]
tags("#/civil\n") = [civil]
tags("#a//b/c\n") = [a/b/c]
```

`TestTagDoubleSlash` (which enshrined `a//b`) was replaced by
`TestTagEmptyHierarchySegments`, covering internal double slash, leading
slash, and a combined case.

### Important 6 — documented the multi-line block limitation

`Block.Start` in `types.go` now carries a long comment explaining that
Start..End is a contiguous raw-buffer range, so the first line's syntax
prefix (`> `, list marker+indent) is excluded but any continuation line's
prefix is included — and that M4's `replace_block` must re-emit continuation
prefixes when writing multi-line content back. Two new pinning tests,
`TestBlockIDMultilineBlockquotePrefixAsymmetry` and
`TestBlockIDMultilineListItemPrefixAsymmetry`, both assert the exact
current substring (including the asymmetric prefix) so a future change to
this logic is caught rather than silently altering M4's contract.

### Minor items

- `[fonte:: [[STJ]]]` truncating at `[[STJ` — fixed, cheaply: the
  bracket-form closing-`]` search now tracks nesting depth instead of using
  `bytes.IndexByte` for the first `]`, so a wikilink's own `]]` no longer
  terminates the field's value early. New test
  `TestInlineFieldBracketedNestedWikilinkValue` pins
  `Inline["fonte"] == "[[STJ]]"`.
- `#tag::valor` producing both a tag and a field with an invented key — not
  fixed, reported. Two independently-firing inline parsers (`tagParser` at
  priority 120, the colon-form field parser at 130) both read raw source
  bytes; the field parser's backward key scan does not know `#tag` was
  already claimed by a `TagNode`, so it invents key `"tag"` from the same
  bytes. Fixing this would require cross-extension state (e.g. a shared
  `gparser.Context` marking already-claimed spans), out of proportion to
  the problem: no data is lost, just one extra `Inline` entry with a
  plausible-but-wrong key.
- A new instance of the same class of artifact surfaced while implementing
  Critical 1: making the colon-form parser non-consuming means a second
  `::` inside what used to be an opaque value now also fires the parser.
  `"a:: 1 b:: 2\n"` produces the correct `Inline["a"] == ["1 b:: 2"]` plus
  a spurious `Inline["1 b"] == ["2"]`. `TestInlineFieldTwoPerLine` was
  rewritten to document both. Same reasoning as above applies: not fixed,
  because avoiding it needs per-line state (`gparser.Context`) that the
  review's stated fix ("advance only past `chave::`") does not call for,
  and no correct data is lost.
- `texto ^^abc` yielding block id `abc` (second caret retriggers) — not a
  bug: the first `^` has no valid id after it (`^` is not in
  `blockIDChar`'s alphabet) so it is declined and left as literal text; the
  second `^` starts a fresh, valid candidate. This is the expected
  consequence of goldmark offering every occurrence of the trigger byte,
  not a defect to fix.
- The `tagPrecedingOK` comment implicitly crediting itself for rejecting
  Markdown link destinations — fixed: comment rewritten to state that a
  valid Markdown link's `(dest)` is consumed atomically by CommonMark's
  link parser before the tag parser ever sees the `#` inside it, and that
  `tagPrecedingOK`'s own Ps-category rule would in fact admit `(` — proven
  by a malformed link like `[S] (#introducao)` (space before `(`, so not a
  valid link) still producing a tag.

### Mutation table

| # | Fix | Mutation | Subtest that failed | Restored, green? |
|---|-----|----------|----------------------|-------------------|
| 1 | Critical 1 (colon form: `Advance(2)` not full line) | `block.Advance(2)` -> `block.Advance(len(line))` | `TestInlineFieldValueWithWikilink`, `TestInlineFieldValueWithEmbed`, `TestInlineFieldValueWithMarkdownLink`, `TestInlineFieldTwoPerLine` | Yes |
| 2 | Critical 2 (decline before `(`/`[`) | short-circuited the `after < len(rest) && (...)` check with `false &&` | `TestInlineFieldBracketedDeclinesBeforeMarkdownLink` | Yes |
| 3 | Critical 3 (block id must be on last line) | short-circuited the `lastLine` bounds check with `false &&` | `TestBlockIDMultiplePerParagraph`, `TestBlockIDThreeLineParagraphOnlyLastMarkerCounts` | Yes |
| 4 | Important 4 (`*`/`_`/`~` admitted before `#`) | removed the three runes from `tagPrecedingOK` | `TestTagAdjacentToEmphasisMarkers` | Yes |
| 5 | Important 5 (`collapseTagSlashes`) | body replaced with `strings.TrimSpace(s)` (no collapsing) | `TestTagEmptyHierarchySegments` | Yes |
| 6 | Important 6 (multi-line prefix-asymmetry pinning) | `Start` computed from `lines.At(lines.Len()-1)` instead of `lines.At(0)` | `TestBlockIDMultilineBlockquotePrefixAsymmetry`, `TestBlockIDMultilineListItemPrefixAsymmetry` | Yes |

All six mutations were caught by name; each was reverted and the full suite
re-confirmed green (`go test -race -count=1 ./internal/parser/` -> `ok`)
before moving to the next.

### Verification

```
go test -race -count=1 ./internal/parser/ -v   # ok, all pass
go vet ./...                                   # clean
gofmt -l .                                     # clean, no output
GOOS=linux go vet ./...                        # clean
GOOS=darwin go vet ./...                       # clean
git status --porcelain                         # only the 7 in-scope files modified, no stray files
```

Files touched: `internal/parser/ext_blockid.go`,
`internal/parser/ext_blockid_test.go`, `internal/parser/ext_inline_field.go`,
`internal/parser/ext_inline_field_test.go`, `internal/parser/ext_tag.go`,
`internal/parser/ext_tag_test.go`, `internal/parser/types.go`.
