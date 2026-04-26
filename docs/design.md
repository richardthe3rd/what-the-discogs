# what-the-discogs: Design

## Overview

`what-the-discogs` identifies the exact Discogs pressing of a vinyl record via a guided Q&A session. The user tells it (or shows it a photo of) the record; it queries Discogs and asks only the questions needed to narrow down from hundreds of candidate pressings to one.

## Architecture

Two components:

1. **`wtd` Go binary** — pure data tool. Fetches and structures Discogs API data, outputs JSON to stdout. No user interaction, no reasoning.
2. **Claude Code skill** (`/identify-vinyl`) — orchestrates the session. Claude reads `wtd` output, reasons about which fields discriminate between candidates, asks the user targeted questions, and interprets matrix etchings.

See `docs/adr/001-skill-plus-binary-architecture.md` for the rationale.

## Identification Session: 7 Phases

### Phase 0 — Binary setup

Skill checks for `wtd` on PATH → tries `go install` → falls back to `go build -o wtd .`. Verifies `DISCOGS_TOKEN` is set.

### Phase 1 — Initial details

If the user passed an image path (`/identify-vinyl /path/to/photo.jpg`), Claude reads it with the Read tool and uses vision to extract: artist, album, label name, catalogue number, country, year, any visible matrix etchings.

Otherwise Claude asks conversationally for artist and album, plus any other details the user knows.

### Phase 2 — Master release search

```
wtd search-master --artist "..." --album "..."
```

- 0 results → fall back to `wtd search-release`
- 1 result → auto-select, confirm with user
- Multiple results → show list, ask user to pick

If no master exists (one-off pressings), treat release search results as the candidate set and skip to Phase 5.

### Phase 3 — Load all versions

```
wtd versions --master {id}
```

Fetches all versions (paginated, up to 500/page). Applies any pre-known hints (year, country, format) from Phase 1 to immediately narrow the set.

Shows: "Found N versions. Let me narrow these down."

### Phase 4 — Structured field narrowing

Claude examines the version data and identifies which of these fields have multiple distinct non-empty values across the candidate set:

| Field | Plain-English question |
|---|---|
| `format` | What format is your record? (LP, Single, EP...) |
| `country` | What country is printed on the label? |
| `label` | What record label is on the label? |
| `year` | What year is printed on the label? |
| `catno` | What's the catalogue number? |
| `format_descriptions` | Any additional descriptors? (Stereo, Mono, Promo...) |

**Ordering**: Claude asks about fields with the fewest distinct values first (binary choices are easiest). Fields where all candidates share one value are skipped.

After each answer, Claude filters the candidate set in-memory. If an answer matches no candidates, it's discarded and the next field is tried.

Loop continues until ≤ 3 candidates remain or no more discriminating fields exist.

### Phase 5 — Detail narrowing (matrix etchings)

```
wtd release --id {id}    # once per remaining candidate (≤ 10)
```

Claude fetches full release detail for remaining candidates and checks `identifiers` for `Matrix / Runout`, `Matrix`, `Runout`, and `Barcode` types.

If matrix strings vary between candidates:
- Claude explains what matrix etchings are (etched text in the dead wax/run-out groove between the last groove and the label)
- Asks the user to read what's scratched into Side A (and Side B if needed)
- Uses fuzzy reasoning to match user's transcription against the known strings
- Accounts for: spacing differences, handwritten additions, pressing plant codes, minor typos

If barcodes vary (and release is post-~1985): asks for barcode number.

### Phase 6 — Present result

- **1 candidate**: Show formatted result with artist, album, year, country, label, cat#, format, matrix strings, and Discogs URL (`https://www.discogs.com/release/{id}`)
- **2–5 candidates**: Show comparison table with Discogs URLs
- **0 candidates**: Explain and offer to retry with relaxed constraints

### Phase 7 — Optional collection add

After a confident single match, offer to add it to the user's Discogs collection.

```
wtd identity                                                  # get username
wtd list-folders --username {username}                        # list folders
wtd add-to-collection --username {username} --release-id {id} --folder-id {fid} --notes "..."
```

## `wtd` Binary Subcommands

All output JSON arrays or objects to stdout. Errors to stderr, non-zero exit.

| Subcommand | Flags | Output |
|---|---|---|
| `search-master` | `--artist`, `--album` | `[]MasterResult` |
| `search-release` | `--artist`, `--album` | `[]Version` |
| `versions` | `--master` | `[]Version` |
| `release` | `--id` | `ReleaseDetail` |
| `identity` | — | `Identity` |
| `list-folders` | `--username` | `[]Folder` |
| `add-to-collection` | `--username`, `--release-id`, `--folder-id`, `[--notes]` | `CollectionInstance` |

### Rate limiting

1 request/second (enforced in `discogs.go`). Well within the 25/second authenticated limit. On HTTP 429: exponential backoff 2s → 4s → 8s, up to 3 retries.

### Caching

In-memory cache keyed by URL. Deduplicates requests within a single binary invocation (e.g., re-fetching the same release when narrowing candidates).

## Key Types (see `types.go`)

- `MasterResult` — master release search result
- `Version` — version list entry (bulk data, no identifiers)
- `ReleaseDetail` — full release with `Identifiers []Identifier`
- `Identifier` — `{Type, Value, Description}` — matrix etchings have `Type = "Matrix / Runout"`
- `Folder`, `CollectionInstance`, `Identity` — collection management

## Edge Cases

| Situation | Handling |
|---|---|
| No master found | Fall back to release search |
| 0 versions after filter | Discard that filter, try next field |
| >500 versions | Fetch all pages; warn if > 1000 |
| Matrix data absent | Warn "limited data", show shortlist |
| User answers "Not sure" | Skip that field, continue |
| Multiple masters found | Ask user to pick (show year + version count) |
| Pre-1985 release | Skip barcode question |
| Release-only (no master) | Skip to Phase 5 directly |
