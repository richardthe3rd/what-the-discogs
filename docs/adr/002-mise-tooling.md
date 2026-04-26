# ADR-002: Mise for tool and environment management

## Status

Accepted

## Context

The project needs to:
- Pin a specific Go version for reproducible builds
- Load `DISCOGS_TOKEN` from a local `.env` file without committing it
- Provide simple task aliases (`build`, `install`, `clean`, `test`)
- Keep CI and local development in sync on tooling versions

Options considered: plain `Makefile`, `direnv` + `Makefile`, or `mise`.

## Decision

Use [mise](https://mise.jdx.dev/) (`mise.toml`).

- Pins Go version in `[tools]` — same version used locally and in CI via `jdx/mise-action`.
- `[env] _.file = ".env"` loads `DISCOGS_TOKEN` automatically when entering the project directory.
- `[tasks]` replaces a Makefile with a cleaner syntax and built-in dependency support.
- `jdx/mise-action@v2` in GitHub Actions installs the exact same toolchain as specified in `mise.toml`, eliminating version drift between local and CI.
- Also manages `goreleaser` version, avoiding CI/local mismatch on the release tool.

## Consequences

- Contributors need `mise` installed (`curl https://mise.jdx.dev/install.sh | sh`).
- The `.env` file is gitignored; `.env.example` is committed as a template.
- `mise run build` is the canonical way to build — the README and CLAUDE.md document this.
- CI uses `jdx/mise-action` with `experimental: true` for task support.
