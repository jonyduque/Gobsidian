# Task 17 report — goldmark extension: Dataview inline fields

## What was implemented

`internal/parser/ext_inline_field.go` replaces the stub with a real goldmark
inline extension recognizing two Dataview forms:

- Plain form: `chave:: valor` — value runs to end of line.
- Bracketed form: `[chave:: valor]` — value runs to the closing `]`.

`InlineFieldNode{Key, Value string}` is unchanged from the stub's shape (no
`Start`/`End` — the brief's interface only asks for the pair). `collect` in
`ast.go` already had the `*InlineFieldNode` arm wired to `note.Inline`
(`map[string][]string`), so no changes were needed there.

**Key alphabet** (rule 2): Unicode letters, Unicode digits, literal space
(`' '`, not general whitespace — `'\n'`/`'\t'` are excluded on purpose),
hyphen, underscore. Implemented once in `inlineFieldKeyChar` and shared by
both forms.

**Design, two entry points, two different techniques:**

- `Trigger() []byte{':', '['}`, priority 130 (before wikilink's 150 and
  CommonMark's link parser, so `[chave:: valor]` gets first refusal/claim on
  `[` and `::` gets first refusal/claim on `:`).
- `'['` → `parseFromBracket`: pure forward, single-shot match over the
  current line (same style as `wikilinkParser`/`blockIDParser`/`tagParser`:
  read the whole candidate span, decide, advance once if it matches).
- `':'` → `parseFromColon`: the key was already scanned as ordinary text by
  goldmark before the `::` trigger fires, so there's no way to "reconsume" it
  going forward. Instead it scans **backward** through the raw source bytes
  from the `::` position, accumulating key-alphabet runes, until it hits a
  disallowed character or the start of the *block's own line*. That
  lower bound comes from `parent.Lines()` — exactly the same technique
  `BlockIDNode.Start` uses — not from scanning for `'\n'` in the raw buffer.
  This distinction is load-bearing: the raw source array still physically
  contains `"- "` / `"> "` syntax prefixes before a list-item/blockquote
  line's content, and `'-'` is a legal key character. Without the
  `parent.Lines()` bound, `"- autor:: Fulano"` back-scans straight through
  the list marker and produces key `"- autor"` instead of `"autor"`. Two
  tests (`TestInlineFieldInListItem`, `TestInlineFieldInBlockquote`) pin this
  down; both failed key-content-wise before the fix was in the code (caught
  during design, not after a red run — see Testing note below).

Two colons is enforced identically in both entry points: reject if a third
`:` immediately follows.

## TDD Evidence

**RED** — wrote `internal/parser/ext_inline_field_test.go` first (brief's
three given tests + the additional ones below), ran against the still-stub
`InlineFieldExtension`:

```
$ go test ./internal/parser/ -run TestInlineField -v
=== RUN   TestInlineFields
    ext_inline_field_test.go:18: autor = [], quer [Fulano]
    ...
--- FAIL: TestInlineFields (0.00s)
--- FAIL: TestInlineFieldRepeatedKey (0.00s)
--- PASS: TestInlineFieldNotInCode (0.00s)   # stub is a no-op, so "nothing produced" trivially passes
...
FAIL
FAIL	github.com/jonyd/gobsidian/internal/parser	0.759s
```
All failures were exactly the expected "map is empty because the extension
is a no-op" reason — nothing unexplained.

**GREEN** — after implementing `ext_inline_field.go`:

```
$ go test ./internal/parser/ -run TestInlineField -v
... (18 test functions, all subtests)
PASS
ok  	github.com/jonyd/gobsidian/internal/parser	0.735s
```

Full suite with race detector, including Tasks 12–16:

```
$ go test -race ./internal/parser/ -v
... (all TestInline*, TestTag*, TestWikilink*, TestSplitFrontmatter*,
     TestDecodeFrontmatter*, TestExtractHeadings*, TestSlug)
PASS
ok  	github.com/jonyd/gobsidian/internal/parser	1.936s
```

## The twelve extra checks

| # | Case | Result |
|---|------|--------|
| 1 | `14:30` (single colon, time) | No field. `line[1] != ':'` rejects at the first `:` before any key logic runs. |
| 2 | `a:b` (single colon) | No field. Same rejection. |
| 3 | `http://exemplo.com` | No field. At the first `:` in `http:`, next char is `/`, not `:` — rejected. |
| 4 | `C:\caminho` (Windows path) | No field. Next char after `:` is `\`, not `:` — rejected. |
| 5 | `chave::valor` (no space after `::`) | Field produced: key `"chave"`, value `"valor"`. Value is `TrimSpace`d, so absence of a separating space doesn't matter. Test: `TestInlineFieldNoSpaceAfterColons`. |
| 6 | `chave::` (empty value) | Field produced: key `"chave"`, value `""` (present in the map, not omitted). Test: `TestInlineFieldEmptyValue`. |
| 7 | `:: valor` (no key) | No field. Backward scan finds nothing left of `pos` (start of line) → `scanInlineFieldKeyBackward` returns `ok=false`. Test: `TestInlineFieldNoKeyRejected`. |
| 8 | `data de leitura:: 2026-01-01` (key with internal space) | Field produced: key `"data de leitura"`, value `"2026-01-01"` — space is in the key alphabet by design. Test: `TestInlineFieldKeyWithSpace`. |
| 9 | `[chave:: valor]` and `[chave:: valor] texto depois` | Both produce key `"chave"`, value `"valor"`; trailing text after `]` is left untouched (only the matched span through `]` is consumed). Test: `TestInlineFieldBracketedTrailingText`. |
| 10 | `a:: 1 b:: 2` (two fields, one line, no brackets) | **One** field: key `"a"`, value `"1 b:: 2"`. The plain form consumes to end of line per rule 3, so the embedded `b::` never gets its own trigger chance — it's swallowed as literal value text. This matches real Dataview behavior (plain inline fields also consume the rest of the line there; only the bracketed form supports multiple fields per line). Verified separately that `[a:: 1] [b:: 2]` **does** produce two fields (`TestInlineFieldBracketedTwoPerLine`). Tests: `TestInlineFieldTwoPerLine`, `TestInlineFieldBracketedTwoPerLine`. |
| 11 | Field inside a list item / inside a blockquote | Both found, with the correct key (not contaminated by the `"- "` or `"> "` prefix) — see the `parent.Lines()` discussion above. Tests: `TestInlineFieldInListItem`, `TestInlineFieldInBlockquote`. |
| 12a | `fonte:: [[STJ]]` (value contains a wikilink) | Field produced: key `"fonte"`, value `"[[STJ]]"` (literal string). The wikilink is **not** separately collected in `note.Links` — the plain form's `block.Advance(len(line))` consumes the whole line opaquely, the same way `WikilinkNode.Alias` text is never itself inline-parsed. Documented and pinned by `TestInlineFieldValueWithWikilink`, which asserts both the string value and that `note.Links` is empty. This is a real, intentional simplification (P1 scope, "doesn't reimplement Dataview"), not treated as a bug to fix. |
| 12b | Field inside a fenced code block / inline code span | Both produce nothing — goldmark suppresses inline-parser triggers in both contexts, as already guaranteed for the other three extensions. Tests: `TestInlineFieldNotInCode` (brief-given), `TestInlineFieldInlineCodeSpan` (added). |
| — | `chave:::valor` (three colons) | No field. At the first `:`, the immediate lookahead check (`line[2] == ':'`) rejects outright. Traced by hand what happens on the goldmark retry at the second `:` too (falls back to plain-text advance and retrigger): the immediately-preceding byte is then the *first* colon, which isn't a key character, so the backward key-scan comes up empty and rejects again. Net result across all retrigger attempts: no field anywhere in the line. Test: `TestInlineFieldTripleColonRejected`. |

## Files changed

- `internal/parser/ext_inline_field.go` — full extension (196 lines, was a
  29-line stub).
- `internal/parser/ext_inline_field_test.go` — new, 241 lines: the brief's 3
  tests plus 15 more covering the checks above.

Both files are the only ones touched, per scope.

## Self-review

- **Correctness / false positives:** three realistic Portuguese sentences
  with ordinary colons (`"Nota: leia..."`, `"Fonte: STJ, REsp 123456."`,
  `"Prazo: 15 dias uteis..."`) were run through `parser.Parse` via a
  throwaway `go run` script (created outside `internal/`, deleted
  immediately after, never staged/committed) — all three produced
  `Inline=map[]`. No false positives.
- **Purity:** no `ctx`, no I/O, no package-level mutable state; the two new
  helper functions (`scanInlineFieldKeyBackward`, `inlineFieldKeyChar`) are
  pure functions of their arguments, matching the existing extensions'
  style. `md` (the shared `goldmark.Markdown`) is unchanged except for the
  new option registration, which happens once at package init as before —
  safe for concurrent `Parse`.
- **Discipline:** nothing beyond what the brief/rules ask for. No
  `Start`/`End` fields were added to `InlineFieldNode` (the brief's
  interface is `{Key, Value string}` only, unlike `BlockIDNode`/`Link`).
  No new file named `helpers.go`/`utils.go`/`common.go`. No changes outside
  the two in-scope files.
- **Testing output:** `go test -race ./internal/parser/ -v` is clean, ASCII
  test names throughout (Portuguese without diacritics in identifiers, per
  the existing test files' convention), no stray prints, no skipped tests
  (RF-18/no-op fallback in the brief's Step 5 was not needed — the feature
  landed without needing to invoke that escape hatch).

## Concerns

- The `a:: 1 b:: 2` behavior (rule 3 taken literally: plain form consumes to
  end of line, so only one field survives) may surprise a future reader who
  expects "greedy field detection." It's intentional, spec-compliant, and
  matches real Dataview, but it's worth flagging since it's the kind of
  thing that looks like a bug on first read of `note.Inline`.
- Wikilinks embedded inside a plain-form field's value are not collected as
  links (see check 12a). This mirrors an existing precedent in the codebase
  (`WikilinkNode.Alias` isn't itself inline-parsed either) but is a real,
  if narrow, gap in the link graph for notes that write `fonte:: [[Nota]]`.
  Given P1 priority and the brief's explicit "don't reimplement Dataview's
  query language" scope, I left it as documented behavior rather than
  building delimiter-stack machinery to fix it — flagging in case product
  wants this revisited later (M6+ maybe, alongside the `Link.Start`/`End`
  gap already noted in `types.go` for M5).

## Fix pass 2 — line-level plain fields

`fix(parser): stop inline fields eating links, scope block ids to the last
line, widen tag prefixes` (`61484c6`) made the plain form's value
non-consuming — the right call, since it put wikilinks, embeds and Markdown
links back in `note.Links` for `fonte:: [[STJ]]` / `capa:: ![[img.png]]` /
`fonte:: [STJ](stj.md)`. But offering that value span back to the inline
scanner meant a later `::` in the same line could retrigger the field
parser:

- `a:: 1 b:: 2` produced `Inline["a"]="1 b:: 2"` **and** a spurious
  `Inline["1 b"]="2"` — the second `::` retriggered on bytes that already
  belonged to `a`'s value.
- `#tag::valor` produced a spurious `Inline["tag"]="valor"` — a key
  assembled from bytes the tag parser (priority 120, fires first) had
  already claimed.

Both artifacts invent metadata nobody wrote, straight into the map
`note_list`/`vault_search` filter on.

### The fix

Dataview's plain inline field is line-level, not span-level: the key must
begin at the start of the line — after a list marker or blockquote prefix
(which `parent.Lines()` already discounts from `lower`), with no other text
before it. The bracketed form is untouched and keeps working anywhere in the
line, which is how `[a:: 1] [b:: 2]` still yields two fields.

Reused the exact bound (`lower`, from `parent.Lines()`) that the list-marker
fix already established, rather than adding a second mechanism. The only
code change is in `scanInlineFieldKeyBackward`: previously, hitting a
non-key rune (or invalid UTF-8) before reaching `lower` did `break` and fell
through to accept whatever key-alphabet run had accumulated so far —
allowing a "key" that starts partway through the line. Now both cases
`return "", false` instead — the loop only completes normally when it
reaches `lower` unobstructed, i.e. the entire span from line-start to `::`
consists of key-alphabet characters. That single change:

- Rejects `"a:: 1 b:: 2"`'s second attempt: back-scanning from the second
  `::` hits the `:` of the first `a::` pair before reaching line-start
  (position 0) — the run stops at `"1 b"`, which isn't `lower`, so reject.
- Rejects `"#tag::valor"`: the tag parser advanced past `"#tag"` but the raw
  source buffer still has `'#'` sitting there; the back-scan from `::` hits
  `'#'` (not a key character) before reaching `lower` — reject.
- Leaves the first field in `"a:: 1 b:: 2"` alone: `"a"` back-scans cleanly
  to `lower` (position 0, nothing precedes it), so it still matches.
- Leaves `"- autor:: Fulano"` / `"> autor:: Fulano"` alone: `Lines()`
  already discounts the `"- "` / `"> "` prefix from `lower`, so `"autor"`
  back-scans cleanly to that (already-adjusted) line start.

### Test changes

- `TestInlineFieldTwoPerLine` updated: asserts `Inline["1 b"]` is now
  **absent** and `len(note.Inline) == 1` (only `"a"`), replacing the old
  assertion that the spurious key existed. Comment rewritten to explain the
  rejection path instead of documenting it as accepted behavior.
- Added `TestInlineFieldAfterTagNotField`: asserts `#tag::valor` produces
  empty `note.Inline` and still produces `note.Tags == ["tag"]` (the tag
  extraction itself is unaffected — only the spurious field is gone).

### Mutation check (per `proving-tests-can-fail`)

Reverted the two `return "", false` back to `break` (restoring the pass-1
behavior of accepting a partial, non-line-start key) and reran
`go test ./internal/parser/ -run TestInlineField -v`:

```
=== RUN   TestInlineFieldTwoPerLine
    ext_inline_field_test.go:176: inline = map[1 b:[2] a:[1 b:: 2]], "1 b" nao deveria existir (regra 2: campo simples e de linha)
    ext_inline_field_test.go:179: inline = map[1 b:[2] a:[1 b:: 2]], quer so a chave "a"
--- FAIL: TestInlineFieldTwoPerLine (0.00s)
=== RUN   TestInlineFieldAfterTagNotField
    ext_inline_field_test.go:197: inline = map[tag:[valor]], quer vazio ("tag" nao pode virar chave de campo)
--- FAIL: TestInlineFieldAfterTagNotField (0.00s)
```

Both failures name exactly the spurious keys the fix exists to prevent —
confirming the tests actually catch the regression, not just happen to pass.
Restored the fix immediately after; full suite green again
(`go test -race ./internal/parser/ -v`, `go vet ./...`, `gofmt -l .`,
`GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...` all clean).

### No-regression check

Re-verified the seven cases the pass-1 fix existed for, all still hold with
pass 2 applied:

| Case | Result |
|---|---|
| `fonte:: [[STJ]]` | `Inline["fonte"]=["[[STJ]]"]` + wikilink `STJ` in `Links` |
| `capa:: ![[img.png]]` | `Inline["capa"]` set + embed `img.png` (`LinkEmbed`) in `Links` |
| `fonte:: [STJ](stj.md)` | `Inline["fonte"]` set + markdown link `stj.md` in `Links` |
| `[Nota:: veja](destino.md)` | `Inline` empty (bracket form declines before `(`) + markdown link `destino.md` present |
| `[a:: 1] [b:: 2]` | Two fields: `a=[1]`, `b=[2]` |
| `- autor:: Fulano` | `Inline["autor"]=["Fulano"]`, marker not in key |
| `> autor:: Fulano` | `Inline["autor"]=["Fulano"]`, prefix not in key |
| `tema::` repeated key (two notes) | Two values accumulate under one key (`TestInlineFieldRepeatedKey`, untouched by this pass) |

### Files changed

- `internal/parser/ext_inline_field.go` — `scanInlineFieldKeyBackward`
  changed to reject (not truncate-and-accept) when the backward key scan
  doesn't reach line-start; comment rewritten to explain regra 2 as a
  line-level constraint.
- `internal/parser/ext_inline_field_test.go` — `TestInlineFieldTwoPerLine`
  updated to assert the spurious key is gone; added
  `TestInlineFieldAfterTagNotField`.

Scope held to exactly these two files, per the brief.
