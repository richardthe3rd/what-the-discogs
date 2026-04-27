---
name: identify-vinyl
description: Identify the exact Discogs pressing of a vinyl record via guided Q&A. Optionally pass an image path to start from a photo.
allowed-tools: Bash(wtd *), Bash(mise *), Bash(which *), Read
---

# Vinyl Record Identification

You are helping the user identify the exact Discogs pressing of a vinyl record they have in front of them.

Binary status: !`which wtd 2>/dev/null && echo "found: $(which wtd)" || echo "not found"`

## Phase 0: Setup

**Ensure `wtd` is available.** The plugin provides `wtd` via a mise tool stub, so [mise](https://mise.jdx.dev) must be installed and on PATH. Check the binary status line above, then:

1. **Found** — nothing to do.
2. **Not found** — mise may not be installed or may not be on PATH. Ask the user to install mise by visiting https://mise.jdx.dev and following the instructions for their platform (this skill cannot run the installer). After installing, they should open a new terminal, then run `/reload-plugins` and retry.

**Verify DISCOGS_TOKEN is set:**
```bash
echo ${DISCOGS_TOKEN:+set}
```
If empty, tell the user to set `DISCOGS_TOKEN` and stop. Direct them to:
1. Get a token at https://www.discogs.com/settings/developers
2. Add it to `~/.claude/settings.json` so it's available in every Claude Code session:
   ```json
   {
     "env": {
       "DISCOGS_TOKEN": "your_token_here"
     }
   }
   ```
   Then run `/reload-plugins` to pick it up.

## Phase 1: Initial details

If the user passed an image path as `$ARGUMENTS`:
- Use the Read tool to view the image.
- Extract what you can see: artist name, album title, label, catalogue number, country, year, any text visible in the dead wax/run-out groove.
- Show what you extracted and ask the user to confirm or correct it.

Otherwise, ask the user:
- What artist and album are they looking for?
- Anything else they already know (year, country, label, catalogue number)?
- Can they read any text etched into the dead wax (the shiny area between the last groove and the label)? These are matrix markings — strings like `XARL-7503` or `YEX 749-1`.

Keep these as hints — you'll use them to skip questions later.

**If matrix etchings are available at this stage**, jump straight to Phase 2b.

## Phase 2: Find master release

```bash
wtd search-master --artist "ARTIST" --album "ALBUM"
```

- **0 results**: try `wtd search-release --artist "ARTIST" --album "ALBUM"` as fallback. If still 0, ask the user to check the spelling.
- **1 result**: auto-select. Tell the user what you found.
- **Multiple results**: show them (title, year, version count) and ask the user to pick.

If using release search results (no master): skip Phase 3 and go directly to Phase 5 with those releases as candidates.

## Phase 2b: Matrix search shortcut (skip to here if matrix etchings known)

If the user already knows matrix etchings from Phase 1, search directly:

```bash
wtd search-matrix --query "MATRIX_STRING"
```

For example: `wtd search-matrix --query "XARL-7503"` or `wtd search-matrix --query "YEX 749"`.

- **Matches found**: confirm with the user which looks right (title, country, year, label), then jump to Phase 6.
- **No matches**: fall back to the standard Phase 2 → Phase 5 flow.

## Phase 3: Load all versions

```bash
wtd versions --master MASTER_ID [--country COUNTRY] [--year YEAR] [--format FORMAT]
```

Apply any pre-known hints from Phase 1 immediately as CLI flags — if the user already told you the country and year, pass them directly:

```bash
wtd versions --master MASTER_ID --country "US" --year "1969"
```

Show the user: "Found N versions. Let me narrow these down."

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
wtd identity        # → {"username": "..."}
wtd list-folders    # uses identity automatically if --username omitted
```

Show the folders and ask which one (default: Uncategorized, folder ID 1). Ask if they want to add any notes.

```bash
wtd add-to-collection --release-id ID --folder-id FOLDER_ID --notes "NOTES"
```

Confirm success and show the user's collection URL.

---

**Remember**: You are the intelligence here. The `wtd` binary only fetches data. You decide which questions to ask, in what order, and how to interpret the answers. The user should never need to understand Discogs data structures.
