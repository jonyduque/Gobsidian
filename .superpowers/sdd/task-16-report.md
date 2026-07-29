# Task 16 report: hierarchical inline tags

## What was implemented

`internal/parser/ext_tag.go` replaces the stub with a goldmark inline extension
that recognizes `#tag` and `#tag/subtag` in the note body, following the same
shape as Task 14's `ext_wikilink.go` and Task 15's `ext_blockid.go`:
`TagNode` (`Kind()`, `Dump()`), `tagParser` (`Trigger()`, `Parse()`), and
`TagExtension.Extend` registering the parser at priority 120.

### Preceding-character rule (Rule 1)

A `#` only starts a tag when `text.Reader.PrecendingCharacter()` returns one
of:

- `'\n'` — start of line or start of document (this is what goldmark reports
  in both cases, so "start of line" needs no separate check).
- any Unicode whitespace (`unicode.IsSpace`).
- straight double quote `"` or straight single quote `'` — checked explicitly
  because both are Unicode category `Po` ("other punctuation"), not an
  "opening" category, but they open a quotation in ordinary use.
- any rune in Unicode category `Ps` (open punctuation: `(`, `[`, `{`, and
  their full-width/Unicode equivalents), `Pi` (initial quote punctuation:
  curly `“`, `‘`, guillemets `«`, `‹`), or `Pd` (dash punctuation: hyphen,
  en dash, em dash).

Everything else — letters, digits, closing punctuation (`)`, `]`, `Pe`
category), `#` itself — rejects the tag. This is what keeps
`https://x.com/a#secao` from producing a tag: the character before `#` is
`a`, a letter, which is in none of the allowed classes.

### Tag body (Rules 2–3)

The name is scanned greedily over `unicode.IsLetter(r) || unicode.IsDigit(r)
|| r == '-' || r == '_' || r == '/'`. It must contain at least one Unicode
letter (`#123` → rejected, `#1a` → accepted). Trailing `/` characters are
trimmed from the captured name before advancing the reader — `#tag/` yields
tag `tag`, not `tag/` with an empty final hierarchy segment; the trimmed
slash is left in the source as literal text. This trimming is a decision
beyond the brief's five rules (see "Two things I decided," below).

### Rule 4 (heading vs. tag) and Rule 5 (code contexts)

`#` followed by a space, by punctuation outside the alphabet, or by nothing
(end of line) yields a zero-width scan, which the `hasLetter`/`trimmed == 0`
check rejects — no special-casing needed. Code-block/code-span suppression
is entirely goldmark's; nothing here tracks fence state.

## TDD Evidence

**RED** — `go test ./internal/parser/ -run "TestInlineTags|TestTag" -v`
(test file written first, against the still-stub `ext_tag.go`):

```
=== RUN   TestInlineTags
    ext_tag_test.go:19: tags = [], quer [civil civil/obrigacoes proc-civil]
--- FAIL: TestInlineTags (0.00s)
=== RUN   TestTagRejections
--- PASS: TestTagRejections (0.00s)          # vacuously true: stub never produces a tag
=== RUN   TestTagsFromFrontmatterMerge
    ext_tag_test.go:62: tags = [civil penal], quer 3 unicas
--- FAIL: TestTagsFromFrontmatterMerge (0.00s)
=== RUN   TestTagPunctuationBoundary
--- FAIL: TestTagPunctuationBoundary (0.00s)
=== RUN   TestTagTrailingSlash
--- FAIL: TestTagTrailingSlash (0.00s)
=== RUN   TestTagDoubleSlash
--- FAIL: TestTagDoubleSlash (0.00s)
=== RUN   TestTagCharset
--- FAIL: TestTagCharset (0.00s)
=== RUN   TestTagOpeningPunctuation
--- FAIL: TestTagOpeningPunctuation (0.00s)
=== RUN   TestTagAdjacentHash
--- FAIL: TestTagAdjacentHash (0.00s)
=== RUN   TestTagDoubleHash
--- PASS: TestTagDoubleHash (0.00s)          # vacuously true
=== RUN   TestTagLoneHash
--- PASS: TestTagLoneHash (0.00s)            # vacuously true
=== RUN   TestTagInsideWikilinkAlias
--- PASS: TestTagInsideWikilinkAlias (0.00s) # vacuously true
=== RUN   TestTagInHeading
--- FAIL: TestTagInHeading (0.00s)
=== RUN   TestTagRealURLs
--- PASS: TestTagRealURLs (0.00s)            # vacuously true
FAIL
```

All failures are the expected "produced zero tags, wanted some" shape —
confirms the test file exercises real behavior the stub doesn't provide, not
a typo in the test.

**GREEN** — `go test -race ./internal/parser/ -v` after implementing
`ext_tag.go`:

```
ok  github.com/jonyd/gobsidian/internal/parser  1.764s
```

Every subtest in the file passed (23 tag-related test functions/subtests),
plus all of Tasks 12–15's existing tests (`TestWikilinkForms`,
`TestWikilinkSuppressedInCode`, `TestBlockID*`, `TestExtractHeadings*`,
`TestSplitFrontmatter*`, `TestSlug`, etc.) — 90 total test cases in the
package, all passing under `-race`.

## The twelve extra checks

| # | Input | Actual result | Notes |
|---|-------|----------------|-------|
| 1 | `#tag.` / `#tag,` | Tag = `tag`; the period/comma is not consumed | Punctuation isn't in the tag alphabet, so the scan stops before it. Sentence-final punctuation never leaks into a tag. |
| 2 | `#tag/` trailing slash, and `#a//b` | `#tag/` → tag `tag` (trailing `/` trimmed, left as literal text). `#a//b` → **single** tag `a//b` (both slashes preserved literally) | Trailing-slash trimming was a deliberate decision (a hierarchy segment can't be empty). The double-slash case is a genuine gap: `/` is "preserved literally" per Rule 3 with no segment-emptiness check, so `a//b` becomes one tag with an embedded empty segment. Documented, not fixed — outside the five rules, and fixing it (rejecting/splitting on empty segments) wasn't specified. |
| 3 | `#tag-com-hifen`, `#tag_com_underscore`, `#Maiúscula`, `#ação` | All produce the full tag verbatim: `tag-com-hifen`, `tag_com_underscore`, `Maiúscula`, `ação` | `unicode.IsLetter` covers accented Latin letters (Lu/Ll categories), so non-ASCII letters work without special-casing. |
| 4 | `(#tag)`, `[#tag]`, `"#tag"` | All produce tag `tag`; the closing bracket/quote is left as literal text | `(` and `[` are Unicode category `Ps`; `"` is allowed via the explicit straight-quote check. |
| 5 | `#tag#outra` | One tag: `tag`. `outra` is not extracted | After consuming `#tag`, the reader's preceding character for the second `#` is `g` (a letter) — Rule 1 rejects it, same rule that stops `a#b`. |
| 6 | Tag right after an em dash, and after an opening quote | `texto—#tag` → tag `tag`. `“#tag”` → tag `tag` | Em dash is category `Pd`; curly opening double quote `“` is category `Pi`. Both allowed. |
| 7 | `##tag` | Zero tags | First `#`: the byte right after it is `#`, not in the tag alphabet, so the scan is empty → rejected. Second `#`: preceding character is `#` (category `Po`, not in the allowed set) → rejected. |
| 8 | `#` alone at end of line, `#` followed only by punctuation | Zero tags in both cases | `#\n` → zero-width scan (next byte is `\n`). `#!` → zero-width scan (`!` isn't in the alphabet). Both hit the `trimmed == 0` rejection. |
| 9 | `[[nota|#tag]]` | Zero tags; `note.Links[0].Alias == "#tag"` | The wikilink parser (Task 14) consumes the entire `[[...]]` span in one `Parse` call, including the `#` inside the alias, before the tag parser ever gets offered that byte. |
| 10 | `## Título #tag` | One tag: `tag` | ATX heading content is parsed for inlines like any other leaf block; the `#` in `#tag` is preceded by a space, same as in a paragraph. |
| 11 | `#1a` | Tag = `1a` | Digit-first is fine as long as at least one letter is present anywhere in the scanned run (Rule 2 only forbids all-digit names). |
| 12 | Tag inside fenced code block, inside inline code span | Zero tags in both | Confirmed via `TestTagRejections/dentro_de_codigo` and `.../codigo_inline` — goldmark suppresses the inline-parser trigger inside both contexts, exactly as the brief states; no fence-tracking code was needed. |

## Self-review

- **Correctness (URL fragments):** tested three real-looking URLs —
  `https://en.wikipedia.org/wiki/Go_(programming_language)#Concurrency`
  (preceded by `)`, category `Pe`/close — not allowed),
  `https://github.com/user/repo#readme` (preceded by `o`, a letter — not
  allowed), `https://docs.python.org/3/library/re.html#re.match` (preceded
  by `l`, a letter — not allowed). All three produce zero tags
  (`TestTagRealURLs`, passing).
- **Purity:** no I/O, no `ctx`, no package-level mutable state. `tagParser`
  and `TagExtension` are stateless value/pointer receivers; `tagNameChar` and
  `tagPrecedingOK` are pure functions of a single rune.
- **Discipline:** `TagNode` carries only `Name string` (plus the embedded
  `gast.BaseInline`), matching the brief's stated interface
  (`parser.TagNode{Name string}`) exactly — no `Start`/`End` fields were
  added even though the sibling `WikilinkNode`/`BlockIDNode` have them,
  because nothing in this task's brief or in `ast.go`'s `*TagNode` arm
  consumes offsets for tags.
- **Testing:** `go test -race ./internal/parser/ -v` output is pristine
  (all PASS, no skips, no `t.Log` noise beyond assertion failures which
  don't occur in the final run).
- **Lint parity:** `golangci-lint run ./internal/parser/...` (v2.12.2,
  matching CI's pin) reports 3 `revive` "exported method needs doc comment"
  findings on `TagNode.Kind`, `TagNode.Dump`, `TagExtension.Extend` — but
  the *identical* findings already exist on `WikilinkNode`, `BlockIDNode`,
  and `InlineFieldNode`. This is the established baseline for the package
  (Tasks 14/15 left it this way), not a regression introduced here.

## Two things I decided beyond the brief's five rules

1. **Trailing-slash trimming.** The brief's Rule 3 only says `/` produces
   hierarchy and is preserved literally for the `civil/obrigacoes` case; it
   says nothing about a trailing `/` with no following segment. I chose to
   trim trailing `/` from the captured name (leaving it as literal text
   after the tag) because a tag ending in `/` implies an empty hierarchy
   segment, which downstream consumers (tag index, `tags` filter) would have
   to special-case anyway. Covered by `TestTagTrailingSlash`.
2. **Double slash left un-normalized.** `#a//b` produces the single tag
   `a//b` rather than stopping at the first `/` or rejecting the empty
   segment. I did not fix this because nothing in the five rules addresses
   consecutive slashes, and inventing segment-validation logic not asked for
   risked exactly the "beyond the brief" scope creep the task warns against.
   Documented and tested (`TestTagDoubleSlash`) as known behavior for a
   future task to tighten if it turns out to matter.

## Files changed

- `internal/parser/ext_tag.go` — full extension implementation (was a stub).
- `internal/parser/ext_tag_test.go` — new; brief's three required tests plus
  12 additional test functions covering the extra checks above.

## Concerns

None blocking. The one open question is whether `#a//b` → `a//b` (a tag
with an embedded empty segment) is acceptable for the tag index and
`tags` filter consumers in later milestones — flagged above, not resolved,
since it's outside this task's five rules.
