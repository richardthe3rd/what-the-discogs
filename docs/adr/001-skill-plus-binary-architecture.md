# ADR-001: Claude Code skill + Go binary architecture

## Status

Accepted

## Context

The tool needs to:
1. Query the Discogs REST API (authenticated, rate-limited, paginated)
2. Reason about which fields discriminate between hundreds of candidate pressings
3. Interpret free-text data (matrix etchings) that users transcribe imprecisely
4. Guide an interactive Q&A session to narrow candidates

Two broad approaches were considered:

**Option A**: Standalone CLI binary (Go or Python) that calls the Anthropic API directly for AI reasoning. Requires `ANTHROPIC_API_KEY` in addition to `DISCOGS_TOKEN`.

**Option B**: Claude Code skill backed by a Go data binary. Claude (running inside Claude Code) provides the reasoning. The Go binary handles only data fetching.

## Decision

Option B: Claude Code skill + Go binary.

- The user runs Claude Code already. Claude's reasoning is available at no extra API cost and without an additional key.
- Claude's language understanding handles matrix etching interpretation naturally — no fuzzy matching code, no thresholds to tune.
- Claude reasons about which fields actually discriminate (e.g., "all candidates are Stereo — skip that question") without hardcoded logic.
- The Go binary is a focused, testable data layer: it fetches JSON, handles rate limiting and pagination, and nothing else.
- The skill can be updated by editing markdown. The binary can be updated independently.

## Consequences

- Users need Claude Code installed (they already have it to run this skill).
- The Go binary is straightforward to compile and distribute via GoReleaser. It is not stdlib-only: it depends on `golang.org/x/time` (rate limiting) and `github.com/mark3labs/mcp-go` (MCP server), plus their transitives — all resolved at build time and statically linked into the binary.
- Identification quality depends on Claude's reasoning. This is a feature: it improves as Claude models improve, without code changes.
- The binary is invoked via `Bash` tool calls within the skill. JSON parsing happens in Claude's context window.
- For very large version sets (500+ versions), the JSON may be large. Claude handles this but it consumes context.
