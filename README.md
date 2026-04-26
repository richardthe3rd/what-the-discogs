# what-the-discogs

A Claude Code skill that identifies the exact Discogs pressing of a vinyl record in front of you.

Vinyl records — particularly from the 1960s and 70s — were pressed many times across different countries, labels, and years. This tool guides you through a targeted Q&A session to narrow down from hundreds of candidate pressings to the specific one you have.

## How it works

1. You invoke `/identify-vinyl` (or `/identify-vinyl /path/to/photo.jpg` with a photo of the record/label)
2. Claude asks only the questions needed to narrow down the candidates
3. You get a Discogs URL for the exact pressing

## Requirements

- [Claude Code](https://claude.ai/code) (the CLI)
- [mise](https://mise.jdx.dev/) for tooling (`curl https://mise.jdx.dev/install.sh | sh`)
- A [Discogs personal access token](https://www.discogs.com/settings/developers)
- Go 1.23+ (managed by mise — you don't need to install it separately)

## Setup

```bash
git clone https://github.com/richardthe3rd/what-the-discogs
cd what-the-discogs

# Install tools (Go + goreleaser)
mise install

# Add your Discogs token
cp .env.example .env
# Edit .env and set DISCOGS_TOKEN=your_token_here

# Build the wtd data binary
mise run build
```

## Usage

Open Claude Code in the `what-the-discogs` directory, then:

```
/identify-vinyl
```

Or start with a photo of the record label or dead wax:

```
/identify-vinyl /path/to/my-record.jpg
```

Claude will:
- Search Discogs for the master release
- Ask targeted questions about what you can read on the record (country, year, label, catalogue number)
- If needed, ask about the matrix etchings etched into the dead wax
- Identify the pressing and give you its Discogs URL

At the end, you can optionally add the identified release to your Discogs collection.

## Matrix etchings

Matrix etchings (also called run-out etchings) are text scratched into the dead wax — the shiny area between the last groove and the label. They typically look like `XYZ-1A` or `ABCD 123-A`. You need to look closely, often at an angle under a light.

The tool will explain this when it needs to ask.

## Installing `wtd` globally

If you want the `wtd` binary available system-wide:

```bash
mise run install
```

Or download a pre-built binary from the [releases page](https://github.com/richardthe3rd/what-the-discogs/releases).

## Contributing

See `docs/design.md` for the full architecture. Key decisions are in `docs/adr/`.

```bash
mise run build    # build
mise run vet      # lint
mise run test     # test
```
