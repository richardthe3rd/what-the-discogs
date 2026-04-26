# what-the-discogs

Vinyl pressing identification tool. A Claude Code skill backed by a Go data binary.

- **Full architecture**: `docs/design.md`
- **Key decisions**: `docs/adr/`

## Setup

```bash
cp .env.example .env    # add DISCOGS_TOKEN
mise run build          # builds ./wtd binary
```

## Invoke the skill

```
/identify-vinyl
/identify-vinyl /path/to/photo.jpg
```

## `wtd` subcommand reference

All subcommands output JSON to stdout. Errors go to stderr with non-zero exit.

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

- `Version`: `id`, `label`, `country`, `year` (string), `catno`, `format`, `format_descriptions`
- `ReleaseDetail.Identifiers`: type `"Matrix / Runout"` contains etching strings

## Build tasks (mise)

```bash
mise run build            # build ./wtd
mise run install          # go install → adds wtd to GOPATH/bin
mise run vet              # go vet ./...
mise run test             # go test ./...
mise run release-snapshot # local goreleaser snapshot (no publish)
```

## Release

Tag with a semver tag to trigger GoReleaser:

```bash
git tag v1.0.0 && git push --tags
```

## Test against live API

```bash
mise run build
./wtd search-master --artist "Beatles" --album "Abbey Road"
```

## Rate limiting

`discogs.go` uses a token-bucket limiter (`golang.org/x/time/rate`): 1 token/sec refill, burst of 3. The first few calls in a session fire immediately; after the burst it throttles to 1/sec. The Discogs authenticated limit is 25/sec — this is deliberately conservative. Context cancellation propagates correctly (MCP tool call cancelled → in-flight wait aborts).

## Claude Desktop MCP

`wtd mcp` starts an MCP server over stdio. Register it in `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "what-the-discogs": {
      "command": "wtd",
      "args": ["mcp"],
      "env": { "DISCOGS_TOKEN": "..." }
    }
  }
}
```

See `docs/adr/004-mcp-server-claude-desktop.md` for full rationale and setup.
