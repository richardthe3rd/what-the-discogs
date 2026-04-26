# ADR-004: MCP server for Claude Desktop integration

## Status

Accepted

## Context

The Claude Code skill works well for terminal users but has two hard constraints:
1. Requires Claude Code (the CLI) — not accessible to non-technical users.
2. No image input — the user must type the artist and album name manually.

Claude Desktop is the primary interface for most users. It offers:
- Direct image attachment — users can photograph a record label or dead wax and drop it straight into the chat.
- Natural conversation — no slash commands or terminal knowledge required.
- Cover art rendering — tool results containing image URLs render inline.

## Decision

Expose the existing `discogs.go` client as an MCP (Model Context Protocol) server via a new `wtd mcp` subcommand. Claude Desktop spawns `wtd mcp` as a subprocess over stdio; Claude then calls the Discogs tools natively during conversation without any orchestration skill file.

**Library**: `github.com/mark3labs/mcp-go` — widely used, stable API, stdio transport out of the box.

**Tools registered** (mirrors the CLI subcommands 1:1):

| Tool | Purpose |
|---|---|
| `search_masters` | Find master releases by artist + album |
| `search_releases` | Fallback when no master exists |
| `get_versions` | All pressings for a master |
| `get_release` | Full detail: matrix etchings, barcodes, cover art |
| `get_identity` | Authenticated user's username |
| `list_folders` | User's collection folders |
| `add_to_collection` | Add identified release to collection |

**Rate limiting**: the same `rate.Limiter` (1/sec, burst 3) used by the CLI is shared — MCP tool calls are naturally serialised by the limiter and context-cancellation is propagated correctly.

**Cover art**: `get_release` prepends the primary image URL to its text output so Claude Desktop renders it inline in the conversation.

## Claude Desktop setup

`~/Library/Application Support/Claude/claude_desktop_config.json` (Mac):

```json
{
  "mcpServers": {
    "what-the-discogs": {
      "command": "/path/to/wtd",
      "args": ["mcp"],
      "env": {
        "DISCOGS_TOKEN": "your_token_here"
      }
    }
  }
}
```

## Consequences

- `github.com/mark3labs/mcp-go` becomes the first non-`golang.org/x` external dependency.
- The `discogs.go` client is unchanged — `mcp.go` is purely an adapter layer over the same methods.
- Users with Claude Desktop get image-first identification (photograph → automatic artist/album extraction) without any terminal interaction.
- The skill (`.claude/skills/identify-vinyl/SKILL.md`) remains the right path for Claude Code users; the MCP server is additive.
- GoReleaser already produces pre-built binaries, so Desktop users can install `wtd` without Go.
