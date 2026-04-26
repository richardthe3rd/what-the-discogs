# what-the-discogs

Identify the exact Discogs pressing of a vinyl record in front of you — via Claude Desktop (drop a photo) or Claude Code (terminal Q&A).

Vinyl records — particularly from the 1960s and 70s — were pressed many times across different countries, labels, and years. This tool narrows down hundreds of candidate pressings to the specific one you have.

## Two ways to use it

| | Claude Desktop | Claude Code |
|---|---|---|
| **Start** | Drop a photo of the label | `/identify-vinyl` |
| **Input** | Photo + conversation | Terminal Q&A |
| **Requires** | Claude Desktop + `wtd` on PATH | Claude Code CLI + repo |
| **Best for** | Any collector | Terminal users |

---

## Claude Desktop setup

### 1. Install `wtd`

Download a pre-built binary from the [releases page](https://github.com/richardthe3rd/what-the-discogs/releases) and put it somewhere on your PATH, or build from source:

```bash
mise run install   # adds wtd to ~/go/bin
```

### 2. Get a Discogs token

Visit [discogs.com/settings/developers](https://www.discogs.com/settings/developers) and generate a personal access token.

### 3. Register the MCP server

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (Mac) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "what-the-discogs": {
      "command": "wtd",
      "args": ["mcp"],
      "env": {
        "DISCOGS_TOKEN": "your_token_here"
      }
    }
  }
}
```

Restart Claude Desktop. You'll see the vinyl tools available in the tool panel.

### 4. Identify a record

Just chat. Attach a photo of the label or describe what you have:

> "Help me identify this record" *(attach photo)*

or

> "I have a copy of Abbey Road — it's on Apple Records, pressed in the UK. Can you help me figure out which pressing it is?"

Claude will use the Discogs tools, ask any clarifying questions, and show you the exact pressing with its Discogs URL and cover art.

---

## Claude Code setup

```bash
git clone https://github.com/richardthe3rd/what-the-discogs
cd what-the-discogs

mise install                    # installs Go + goreleaser
cp .env.example .env            # add DISCOGS_TOKEN
mise run build                  # builds ./wtd
```

Then in Claude Code:

```
/identify-vinyl
/identify-vinyl /path/to/photo.jpg
```

---

## Matrix etchings

Matrix etchings (run-out etchings) are text scratched into the dead wax — the shiny ring between the last groove and the label. They look like `YEX 749-1` or `ABCD 123-A`. Look closely at an angle under a bright light. They're the most reliable way to distinguish between otherwise identical pressings.

The tool explains this in context when it needs to ask.

---

## `wtd` subcommand reference

```
wtd search-master  --artist STR --album STR      → []MasterResult
wtd search-release --artist STR --album STR      → []Version
wtd versions       --master INT                  → []Version
wtd release        --id INT                      → ReleaseDetail
wtd identity                                     → Identity
wtd list-folders   [--username STR]              → []Folder
wtd add-to-collection \
       --release-id INT \
       --folder-id INT \
       [--username STR] \
       [--notes STR]                             → CollectionInstance
wtd mcp                                          start MCP server (stdio)
```

All subcommands output JSON to stdout. Errors go to stderr with non-zero exit.

## Build tasks

```bash
mise run build            # build ./wtd
mise run install          # go install → adds wtd to ~/go/bin
mise run vet              # go vet ./...
mise run test             # go test ./...
mise run release-snapshot # local goreleaser snapshot
```

## Release

Tag with a semver tag to trigger GoReleaser:

```bash
git tag v1.0.0 && git push --tags
```

## Contributing

See `docs/design.md` for the full architecture. Key decisions are in `docs/adr/`.
