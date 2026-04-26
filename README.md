# what-the-discogs

Identify the exact Discogs pressing of a vinyl record — via Claude Desktop (drop a photo) or Claude Code (terminal Q&A).

Vinyl records from the 1960s–70s were pressed dozens or hundreds of times across different countries, labels, and years. This tool narrows the field from hundreds of candidates to the specific pressing in your hands.

---

## Claude Desktop

Requires a [Discogs token](https://www.discogs.com/settings/developers). Choose either install method:

### Option A — MCP Bundle (no mise required)

1. [Download `what-the-discogs.mcpb`](https://github.com/richardthe3rd/what-the-discogs/releases/latest/download/what-the-discogs.mcpb) from the latest release.
2. Double-click the downloaded file — the installer opens, prompts for your Discogs token, and configures Claude Desktop automatically.
3. Restart Claude Desktop.

### Option B — mise

Requires [mise](https://mise.jdx.dev).

**1. Add to `claude_desktop_config.json`**

Mac: `~/Library/Application Support/Claude/claude_desktop_config.json`  
Windows: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "what-the-discogs": {
      "command": "mise",
      "args": ["x", "github:richardthe3rd/what-the-discogs@latest", "--", "wtd", "mcp"],
      "env": {
        "DISCOGS_TOKEN": "your_token_here"
      }
    }
  }
}
```

**2. Restart Claude Desktop.** That's it.

`mise x` downloads the right `wtd` binary for your OS on first use and caches it — no Go required. To pin a specific version replace `@latest` with e.g. `@v1.2.0`.

### Using it

Drop a photo of the label or dead wax into the chat, or just describe what you have:

> *"Help me identify this record"* + attach photo

or

> *"I have Abbey Road on Apple Records, UK pressing. Can you identify which one?"*

Claude searches Discogs, asks only the questions needed to narrow it down, and returns the exact pressing with its URL and cover art.

---

## Claude Code

Requires [mise](https://mise.jdx.dev) and a [Discogs token](https://www.discogs.com/settings/developers).

### Install via the plugin system

Run these two commands inside Claude Code (no terminal, no git clone needed):

```
/plugin marketplace add richardthe3rd/what-the-discogs
/plugin install what-the-discogs@what-the-discogs
```

Then set your Discogs token (add to `~/.zshrc` or `~/.bashrc`):

```bash
export DISCOGS_TOKEN=your_token_here
```

Run `/reload-plugins`, then use it in any Claude Code session:

```
/what-the-discogs:identify-vinyl
/what-the-discogs:identify-vinyl /path/to/photo.jpg
```

The plugin's `bin/wtd` is a [mise tool stub](https://mise.jdx.dev/dev-tools/tool-stubs.html) — it downloads and caches the `wtd` binary on first use. No separate binary installation needed.

### Alternative: standalone install (git clone)

```bash
git clone https://github.com/richardthe3rd/what-the-discogs
cd what-the-discogs

cp .env.example .env   # add DISCOGS_TOKEN
mise run setup         # installs wtd globally + /identify-vinyl skill
```

After setup:

```
/identify-vinyl
/identify-vinyl /path/to/photo.jpg
```

---

## Matrix etchings

Matrix etchings are text scratched into the dead wax — the shiny ring between the last groove and the label. They look like `YEX 749-1` or `ABCD 123-A`. Look at an angle under a bright light. They're the most reliable way to distinguish between otherwise identical pressings.

The tool explains this in context when it needs to ask.

---

## `wtd` CLI reference

The binary can also be used directly. All subcommands output JSON to stdout; errors go to stderr with non-zero exit.

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

Key JSON fields for identification:
- `Version`: `id`, `label`, `country`, `released`, `catno`, `format`, `format_descriptions`
- `ReleaseDetail.Identifiers`: type `"Matrix / Runout"` contains etching strings
- `ReleaseDetail.Images`: first `"primary"` image URI is the cover art

## Development

```bash
git clone https://github.com/richardthe3rd/what-the-discogs
cd what-the-discogs
mise install          # installs Go + goreleaser
cp .env.example .env  # add DISCOGS_TOKEN
mise run build        # builds ./wtd locally
mise run test         # go test ./...
```

See `docs/design.md` for the full architecture and `docs/adr/` for key decisions.

### Release

Tag with a semver tag to trigger GoReleaser (builds cross-platform binaries, publishes GitHub release):

```bash
git tag v1.0.0 && git push --tags
```
