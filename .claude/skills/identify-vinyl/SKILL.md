---
name: identify-vinyl
description: Identify the exact Discogs pressing of a vinyl record via guided Q&A. Optionally pass an image path to start from a photo.
allowed-tools: Bash(wtd *), Bash(go *), Bash(which *), Bash(mise *), Read
---

# Vinyl Record Identification

You are helping the user identify the exact Discogs pressing of a vinyl record they have in front of them.

Binary status: !`which wtd 2>/dev/null && echo "found at $(which wtd)" || echo "not found"`

## Phase 0: Setup

**Locate the `wtd` binary** (check the binary status line above):

1. If `wtd` is found on PATH: use it directly.
2. If not found, try: `go install github.com/richardthe3rd/what-the-discogs@latest`
3. If that fails (Go not installed or package not published): `go build -o wtd .` from the repo root.
4. If all fail: tell the user to follow the README setup steps and stop.

**Verify DISCOGS_TOKEN**: Run `echo ${DISCOGS_TOKEN:+set}`. If not set, tell the user to add it to `.env` and stop.

> NOTE: This skill is a work in progress. The `wtd` binary subcommands are not yet fully implemented. Phase 2 of development will complete the binary; Phase 3 will complete this skill. For now, verify setup works and provide a helpful "coming soon" message if the binary returns "not implemented".

## Phase 1: Initial details

If the user passed an image path as `$ARGUMENTS`:
- Use the Read tool to view the image.
- Extract what you can see: artist name, album title, label, catalogue number, country, year, any text visible in the dead wax/run-out groove.
- Show what you extracted and ask the user to confirm or correct it.

Otherwise, ask the user:
- What artist and album are they looking for?
- Anything else they already know (year, country, label, catalogue number)?

Keep these as hints — you'll use them to skip questions later.

## Phase 2: Find master release

```bash
wtd search-master --artist "ARTIST" --album "ALBUM"
```

- **0 results**: try `wtd search-release --artist "ARTIST" --album "ALBUM"` as fallback. If still 0, ask the user to check the spelling.
- **1 result**: auto-select. Tell the user what you found.
- **Multiple results**: show them (title, year, version count) and ask the user to pick.

If using release search results (no master): skip Phase 3 and go directly to Phase 5 with those releases as candidates.

## Phase 3: Load all versions

```bash
wtd versions --master MASTER_ID
```

Show the user: "Found N versions. Let me narrow these down."

Apply any pre-known hints from Phase 1 immediately — if the user already told you the country, filter to just those versions now.

## Phase 4: Structured narrowing

Examine the version data. For each of these fields, check if there are multiple distinct non-empty values across your candidate set:

- `format` (LP, Single, EP...)
- `country`
- `label`
- `year` (the `released` field)
- `catno`
- `format_descriptions` (Stereo, Mono, Reissue, Promo...)

**Ask only about fields that actually vary.** Skip fields where all candidates share the same value, or where the field is empty for most candidates.

Ask in order of fewest distinct values first (binary choices are easiest). For each question:
- List the actual options from the data (not generic options)
- Include a "Not sure / skip" option
- Filter your candidate set based on the answer
- Show the remaining count after each filter

If an answer produces 0 matches, tell the user and skip that filter (don't apply it).

Stop when you have ≤ 3 candidates, or when no more fields discriminate.

## Phase 5: Matrix etching detail

For each remaining candidate (maximum 10), fetch release details:

```bash
wtd release --id RELEASE_ID
```

Look at the `identifiers` array. Focus on entries with `type` values like `"Matrix / Runout"`, `"Matrix"`, or `"Runout"`.

If matrix strings vary between candidates:

Explain to the user:
> Matrix etchings (also called run-out etchings) are text scratched into the shiny dead wax — the area between the last groove and the label. Look closely at the record, especially at an angle under a light. They typically look like `XYZ-1A` or `ABCD 123-A`. You may see both machine-stamped text and handwritten additions.

Ask them what they can read for Side A (and Side B if still needed after Side A).

Use your language reasoning to match their input against the known strings. Be generous with spacing differences, minor typos, and extra handwritten characters. A partial match like "YEX 749" matching "YEX 749-1 PECKO DUCK" is valid.

If barcodes vary and the record is likely from after ~1985: ask for the barcode number under the barcode stripes.

## Phase 6: Present result

**Single match**: Show a clear summary:
```
Identified: ARTIST – ALBUM
Year: YEAR  Country: COUNTRY  Label: LABEL  Cat#: CATNO
Format: FORMAT DESCRIPTIONS
Matrix A: ...  Matrix B: ...

https://www.discogs.com/release/ID
```

**2–5 matches**: Show a comparison table with the key differentiating fields and a Discogs URL for each. Tell the user these are your best candidates.

**0 matches**: Explain what happened and offer to retry with relaxed constraints.

## Phase 7: Add to collection (optional, single match only)

Ask: "Would you like to add this to your Discogs collection?"

If yes:
```bash
wtd identity                          # get username
wtd list-folders --username USERNAME  # list their folders
```

Show the folders and ask which one (default: Uncategorized, folder ID 0). Ask if they want to add any notes.

```bash
wtd add-to-collection --username USERNAME --release-id ID --folder-id FOLDER_ID --notes "NOTES"
```

Confirm success.

---

**Remember**: You are the intelligence here. The `wtd` binary only fetches data. You decide which questions to ask, in what order, and how to interpret the answers. The user should never need to understand Discogs data structures.
