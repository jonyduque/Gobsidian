# Task 18 report: golden files do parser

## Status

DONE — commit `6c5a241` "test(parser): golden file corpus covering documented edge cases".

## Corpus inventory (48 fixtures)

| Diretorio | Arquivos | Contagem |
|---|---|---|
| `wikilinks/` | simples, alias, heading, bloco, heading_alias, caminho, embed, embed_imagem, multiplos, triplo, aninhado_ambiguo | 11 |
| `codeblocks/` | cercado, cercado_lang, inline, indentado, til, escapado, colchete_literal | 7 |
| `frontmatter/` | completo, vazio, ausente, malformado, tags_string, tags_lista, aliases, nao_fechado | 8 |
| `headings/` | hierarquia, acentos, fechamento, duplicados, nivel_pulado, cerca_quatro_crases | 6 |
| `blocks/` | simples, multiplos, em_lista, invalido | 4 |
| `tags/` | emphasis, segmento_vazio, url_fragmento | 3 |
| `inline_fields/` | valor_wikilink, multiplos_por_linha | 2 |
| `edge/` | vazio, sem_newline_final, crlf, crlf_misto, bom, so_frontmatter, nota_gigante | 7 |
| **Total** | | **48** |

The brief's table is the floor (wikilinks 9, codeblocks 7, frontmatter 8, headings 5, blocks 4,
edge 7 = 40 files). Two directories are additions beyond the table (`tags/`, `inline_fields/`),
required by the "what the corpus must cover" bullet list but not named as directories in the
brief's table. Extra files beyond the floor:

- `wikilinks/triplo.md` and `wikilinks/aninhado_ambiguo.md` — `[[[triplo]]]` and
  `[[[a]] b](d.md)`, the two degenerate triple-bracket cases the bullet list calls out by name.
  They belong with wikilinks, not codeblocks, because codeblocks/ has a hard zero-links
  requirement and `aninhado_ambiguo.md` produces a real markdown link on purpose.
- `headings/cerca_quatro_crases.md` — four-backtick fence containing a three-backtick line,
  end-to-end through the golden harness (existing unit coverage was
  `TestExtractHeadingsFenceRequiresMatchingCloseLength`; this pins the same rule via `Parse`).

Every `.md` fixture that isn't self-explanatory carries an HTML comment (`<!-- ... -->`)
explaining what it pins. HTML comments are goldmark HTML blocks — no inline parsing happens
inside them, so they don't leak spurious links/tags/blocks into the fixture they document.
One placement constraint mattered: comments can never precede a leading `---` frontmatter
delimiter, because `SplitFrontmatter` requires the literal first three bytes of the file to be
`---` (`bytes.HasPrefix(data, fmDelim)`). Four fixtures — `frontmatter/malformado.md`,
`frontmatter/tags_lista.md`, `frontmatter/aliases.md`, `frontmatter/nao_fechado.md` — were first
written with a leading comment, which silently defeated frontmatter detection entirely (caught
before generating goldens, by re-reading the fixtures against `SplitFrontmatter`'s contract).
Fixed by moving the comment into the YAML body as `#`-prefixed YAML comments (three files) or
after the un-terminated `---` block as a trailing HTML comment (`nao_fechado.md`, where a leading
comment would have broken the exact "opening delimiter never closes" case the fixture exists to
test).

## Generated JSON review

All 48 `.json` files were read in full (or, for `nota_gigante.json`, read plus counted with
`grep -c` against the generator's loop bounds) and checked against the parser source before being
accepted. Every `codeblocks/*.json` is exactly `{}` — no links, tags, blocks, inline fields, or
headings, confirming the goldmark suppression the directory exists to prove.

**One legitimately surprising result, judged correct on inspection:**

`edge/bom.json` is `{}`. `golden_test.go` reads the raw file bytes and calls `parser.Parse`
directly — it does not call `vault.StripBOM`, matching how a real parser test corpus should
exercise `Parse` in isolation. `SplitFrontmatter`'s doc comment states the contract explicitly:
"Exige entrada ja sem BOM UTF-8. [...] Sem ela, `\xEF\xBB\xBF---`" — and by the same mechanism,
`\xEF\xBB\xBF# Titulo` — "nao bate com o prefixo esperado [...] as linhas viram conteudo da
nota." Tracing it through: `ExtractHeadings`'s line scanner does `strings.TrimLeft(line, " ")`,
which doesn't touch the three BOM bytes, so `trimmed[0]` is `0xEF`, not `#` — no heading is
recognized, and the line becomes an ordinary (garbled) text paragraph. The tag parser's
preceding-character check also declines: the BOM's three bytes decode as one rune, U+FEFF (byte
order mark / zero width no-break space, Unicode category Cf), which fails
`unicode.IsSpace` and isn't in the accepted opening-punctuation categories, so even if a `#` were
scanned there, `tagPrecedingOK` would refuse it. Net effect: the fixture pins that
**`parser.Parse` alone does not repair a BOM-prefixed file** — that responsibility belongs to
`vault.StripBOM`, called before `Parse` in the real pipeline (`internal/vault`, not yet wired to
`internal/parser` as of Task 18). This is the documented contract working as intended, not a bug;
flagging it here because a reader of just the `.json` would reasonably expect at least one
heading and be confused without this trace.

**One fixture whose result differs from its original design intent but is correct and now
documented:** `inline_fields/multiplos_por_linha.json` has `"a": ["1 b:: 2", "1"]` — both the
no-brackets line (`a:: 1 b:: 2`, absorbs `1 b:: 2` as the whole value) and the bracketed line
(`[a:: 1] [b:: 2]`, value `1`) use the same key `a`, so `note.Inline["a"]` accumulates both in
document order via the same repeated-key rule `TestInlineFieldRepeatedKey` already covers. I
initially expected the two forms to read as separate, independent assertions; instead they
compose under the same key, which is real, correct, in-spec behavior — the fixture's comment was
updated to name this explicitly instead of leaving a reader to rediscover it.

Everything else matched design intent on first generation: wikilink offsets, embed dual-syntax
collection (`![[x]]` and `![alt](x)`), the triple-bracket decline-and-retry behavior including the
`Alias: " b"` byte-exact detail from `[[[a]] b](d.md)`, frontmatter type preservation
(`time.Time`, lists vs. comma strings), tag dedup/sort, block-id offset asymmetry in multi-line
blockquotes/lists, and CRLF/mixed-EOL/no-final-newline/empty-file handling.

## Mutation evidence (harness can fail)

Hand-edited `testdata/parser/wikilinks/simples.json`, changing `"start": 24` to `"start": 25`
(single off-by-one, no other field touched):

```
--- FAIL: TestGolden/wikilinks\simples.md (0.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/parser	0.875s
```

Failure names the exact subtest (`wikilinks\simples.md`); the diff output (`--- esperado ---` /
`--- obtido ---`) shows the full JSON on both sides. Reverted the edit, then re-ran
`go test -race ./internal/parser/` — passes.

## `-update` is not load-bearing in CI

`go test ./internal/parser/` (no `-update` flag) passes clean on the checked-out tree — confirmed
both immediately after generation and after the mutation-and-restore cycle above.
`go test -race ./internal/parser/ -v` also passes clean (all `TestGolden/*` subtests plus every
pre-existing unit test in the package).

## CRLF and BOM survival after commit

```
git check-attr text -- testdata/parser/edge/crlf.md testdata/parser/edge/crlf_misto.md testdata/parser/edge/bom.md
testdata/parser/edge/crlf.md: text: unset
testdata/parser/edge/crlf_misto.md: text: unset
testdata/parser/edge/bom.md: text: unset
```

`text: unset` confirms the `.gitattributes` `-text` entries are active for exactly these three
paths (the blanket `*.md text eol=lf` rule earlier in the same file would otherwise apply).

Byte comparison, working tree vs. the committed blob (`git show HEAD:path`), both before and
after the commit:

| File | Working tree | `git show HEAD:...` |
|---|---|---|
| `edge/crlf.md` | 7 CR bytes | 7 CR bytes |
| `edge/crlf_misto.md` | 4 CR bytes (mixed CRLF/LF — one line is pure LF by design) | 4 CR bytes |
| `edge/bom.md` | first 4 bytes `EF BB BF 23` | first 4 bytes `EF BB BF 23` |

Did not force an actual `git checkout` of these paths to prove the round-trip a second way — the
task brief bans `git checkout`/`restore`/`reset`/`clean` in this repo unconditionally (a prior
subagent destroyed uncommitted work that way), and the auto-mode permission classifier declined
the command when attempted. `git show HEAD:path` reads the exact bytes git will materialize on
any future checkout of this commit (it's the same blob, same smudge/clean attribute rules), so
this is equivalent evidence without the destructive-command risk.

## Files changed

- `internal/parser/golden_test.go` (new) — harness, verbatim from the brief plus the
  `io/fs.DirEntry` import fix already anticipated there.
- `testdata/parser/**` (new) — 48 `.md` fixtures + 48 `.json` goldens, 96 files.
- `.gitattributes` (modified) — appended the three `-text`/`-text binary` entries for the CRLF and
  BOM fixtures, with a comment explaining why (the pre-existing `*.md text eol=lf` rule would
  otherwise normalize them away on checkout).

## Self-review

- Confirmed no parser source file was touched (`git status` before commit showed only
  `internal/parser/golden_test.go`, `testdata/parser/**`, `.gitattributes`).
- Confirmed `gofmt -l .` and `go vet ./...` are clean after adding the new Go file.
- Re-ran the full suite with `-race` after the final regeneration (post comment-edit on
  `inline_fields/multiplos_por_linha.md`) to make sure the byte-offset shift from editing that
  fixture's comment didn't leave a stale golden anywhere else — it didn't; `git status` showed
  only that one pair of files changing after the `-update` re-run.
- Did not add a fixture for "note with frontmatter, body offset exercised end to end" as a
  separate file — folded it into `frontmatter/completo.md`, which already needed a heading and a
  link past the frontmatter block to be a realistic "complete" fixture. Verified in its `.json`
  that `headings[0].start` (133) and `links[0].start` (162) land well past the frontmatter's byte
  length, not at body-relative offsets.
- `edge/nota_gigante.md` is generated by a bash loop (120 sections), not committed as a script —
  the generation script itself was not persisted anywhere in the repo, only its output. This
  matches "testdata/ for fixtures, never long strings embedded in test code" without adding a
  generator script the reviewer would have to separately vet.

## Concerns

None blocking. Two worth flagging for whoever reviews this task:

1. `edge/bom.md`'s `{}` output is correct but easy to misread as "the parser is broken" without
   the trace above — I've put that trace in this report and in the fixture's nature is
   self-documenting once the reader knows `SplitFrontmatter`'s doc comment. Consider whether a
   future BOM-handling task (wiring `vault.StripBOM` ahead of `parser.Parse` in the real pipeline)
   should add a *second* fixture that runs `StripBOM` first, so the corpus also pins the "happy
   path" and not only the pre-integration gap. Out of scope for Task 18 (parser package can't
   import `internal/vault` without inverting the dependency graph documented in `ESTRUTURA.md`).
2. `tags/` and `inline_fields/` are new directories not named in the brief's table. They're
   consistent with the file-per-extension convention the rest of the corpus follows
   (`ext_tag.go` -> `tags/`, `ext_inline_field.go` -> `inline_fields/`) and directly satisfy bullet
   points in "what the corpus must cover" that had no other home in the given table. Flagging in
   case the brief's table was meant to be exhaustive rather than a floor — the task description
   explicitly says "treat it as the floor," so I read this as within scope.
