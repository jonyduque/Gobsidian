## Anchor links are not broken links

`[x](nota.md#heading)`, `[x](#heading)` and `[[#heading]]` counted as
`target_missing`: the parser handed the anchor to the resolver glued to the
target, and the resolver looked for a note named `nota.md#heading`. Measured on
four real vaults before touching anything: no broken target was a URL —
scheme links were already `external` — and the false positives were the anchor
forms (one vault carried 267 `#`-only links and 10 empty targets, every one
counted as broken; another carried 372).

The `#` split now happens once, before percent-decoding, in the same function
wikilinks already used, so `%23` never becomes a separator. An anchor-only link
resolves to its own note and is then checked as any other anchor. The metadata
cache carries a new parser version, so an index built by v1.4.x is rebuilt on
the first start.

## `vault_broken_links`

A new read tool lists every `target_missing` and `anchor_missing` link in the
vault — source, target, anchor, alias, kind, state and the text around it —
filtered by `state` and by a path `prefix`, paginated with `limit` (default
100, cap 500) and `offset`. `total` is the count before pagination. External
links are never listed. That makes 14 tools: 9 read, 5 write.

## `note_move` rewrites the moved note's own links

A note that cited itself failed halfway through the move: the body had
already been renamed and the link rewrite read the old path — `The system
cannot find the file specified`. Its self-references are now rewritten in place at the new
path, and the metadata index moves the resolved target with them. Anchor-only
self-references (`[[#h]]`) are left alone: they never named the file.

## Docs

`TOOLS.md`, the README and the wiki describe the new tool and the dry-run
asymmetry of `note_move` (diffs under the old path, `rewritten` under the new
one). `ESTADO.md` records the two gate weaknesses found on the way —
`pre_commit_docs.ps1` and `audit_reports.ps1` match words, not evidence — as
open debt, not as fixed.

---

**Full Changelog**: https://github.com/jonyduque/Gobsidian/compare/v1.4.1...v1.5.0
