# Re-review 156 — Fix 1 (b354dcf..86f07e5)

## README.md flag table verification

| Flag | Table says | Grep says | Match? |
|---|---|---|---|
| `--read-only` | `serve`, `daemon`, `doctor` | `serve`, `daemon`, `doctor` | ✓ |
| `--cache-dir` | `serve`, `daemon` | `serve`, `daemon` | ✓ |
| `--debounce-ms` | `serve`, `daemon`, `doctor` | `serve`, `daemon`, `doctor` | ✓ |
| `--log-level` | `serve`, `daemon` | `serve`, `daemon` | ✓ |
| `--eager-search` | `serve`, `daemon` | `serve`, `daemon` | ✓ |
| `--max-results` | `serve`, `search`, `daemon`, `doctor` | `serve`, `search`, `daemon`, `doctor` | ✓ |

## Wiki identifier check

Identifiers cited in the new paragraph of docs/wiki/entities/formato-do-cache.md:

- `cacheCodecVers = CacheFormatVersion` — exists in persist_codec.go:51 ✓
- `cacheMagic` as `var` with `fmt.Sprintf("GBS%d", CacheFormatVersion)` — exists in persist_codec.go:54 ✓
- `CacheFormatVersion = 6` — exists in persist.go:23 ✓

## Files touched

Diff touches only:
- README.md ✓
- docs/wiki/entities/formato-do-cache.md ✓

VERDICT: APPROVED
