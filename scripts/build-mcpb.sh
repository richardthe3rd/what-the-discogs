#!/usr/bin/env bash
# build-mcpb.sh: Assemble and pack the what-the-discogs MCP Bundle.
#
# Usage: bash scripts/build-mcpb.sh <version>
#   version  Semver string without leading 'v', e.g. "1.2.3"
#
# Expects goreleaser to have already built binaries into dist/.
# Produces what-the-discogs.mcpb in the repo root.
set -euo pipefail

VERSION="${1:-0.0.0}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BUNDLE_STAGE="$(mktemp -d)"
OUTPUT="$ROOT_DIR/what-the-discogs.mcpb"

echo "Building MCPB bundle v${VERSION}..."

trap 'rm -rf "$BUNDLE_STAGE"' EXIT

# Copy manifest and any other bundle-level files
cp -r "$ROOT_DIR/mcpb/." "$BUNDLE_STAGE/"
mkdir -p "$BUNDLE_STAGE/server"

# ── macOS ──────────────────────────────────────────────────────────────────
# Universal binary (arm64 + amd64 fat binary produced by goreleaser)
cp "$ROOT_DIR/dist/wtd_darwin_all/wtd"           "$BUNDLE_STAGE/server/wtd-darwin-universal"

# ── Linux ──────────────────────────────────────────────────────────────────
cp "$ROOT_DIR/dist/wtd_linux_amd64_v1/wtd"       "$BUNDLE_STAGE/server/wtd-linux-amd64"
cp "$ROOT_DIR/dist/wtd_linux_arm64/wtd"           "$BUNDLE_STAGE/server/wtd-linux-arm64"

# ── Windows ────────────────────────────────────────────────────────────────
cp "$ROOT_DIR/dist/wtd_windows_amd64_v1/wtd.exe"  "$BUNDLE_STAGE/server/wtd-windows-amd64.exe"
cp "$ROOT_DIR/dist/wtd_windows_arm64/wtd.exe"     "$BUNDLE_STAGE/server/wtd-windows-arm64.exe"

# Make non-Windows binaries executable
chmod +x \
  "$BUNDLE_STAGE/server/wtd-darwin-universal" \
  "$BUNDLE_STAGE/server/wtd-linux-amd64" \
  "$BUNDLE_STAGE/server/wtd-linux-arm64"

# Stamp the version into manifest.json (placeholder is "0.0.0")
sed -i.bak "s/\"0\.0\.0\"/\"${VERSION}\"/" "$BUNDLE_STAGE/manifest.json"
rm -f "$BUNDLE_STAGE/manifest.json.bak"

# Pack the bundle
echo "Packing bundle..."
npx --yes @anthropic-ai/mcpb pack "$BUNDLE_STAGE" "$OUTPUT"

echo "Created: $OUTPUT"
