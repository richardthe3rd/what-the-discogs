# ADR-003: GoReleaser for binary distribution

## Status

Accepted

## Context

Users who want to use the `wtd` binary without cloning the repo or having Go installed need a way to download pre-built binaries. Options:

- Manual `go install` from source (requires Go)
- Hand-crafted GitHub Actions matrix build (complex, no changelog)
- GoReleaser (automated cross-platform release pipeline)

## Decision

Use [GoReleaser](https://goreleaser.com/) triggered on `v*` tag pushes via GitHub Actions.

- Builds for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64` in one workflow.
- Publishes a GitHub release with tarball/zip archives and a `checksums.txt`.
- Auto-generates a changelog from commit messages (excluding `docs:`, `test:`, `chore:` prefixes).
- Configuration lives in `.goreleaser.yaml` — easy to extend (e.g., Homebrew tap, Scoop manifest) later.
- `CGO_ENABLED=0` ensures fully static binaries with no system library dependencies.

## Consequences

- Releases are triggered by pushing a semver tag (`git tag v1.0.0 && git push --tags`).
- The `mise.toml` pins the `goreleaser` version so local snapshot builds match CI exactly.
- `GITHUB_TOKEN` (automatically available in Actions) is the only credential required.
- Users without Go can install `wtd` by downloading the appropriate archive from the GitHub releases page and adding it to their PATH.
- The skill's binary detection will eventually support `gh release download` as a fourth fallback for users who have `gh` but not Go.
