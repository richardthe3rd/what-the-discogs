package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	token := os.Getenv("DISCOGS_TOKEN")
	if token == "" && os.Args[1] != "version" {
		fmt.Fprintln(os.Stderr, "error: DISCOGS_TOKEN environment variable not set")
		fmt.Fprintln(os.Stderr, "Get a token at: https://www.discogs.com/settings/developers")
		os.Exit(1)
	}

	client := NewClient(token)
	ctx := context.Background()

	switch os.Args[1] {
	case "search-master":
		cmdSearchMaster(ctx, client)
	case "search-release":
		cmdSearchRelease(ctx, client)
	case "versions":
		cmdVersions(ctx, client)
	case "release":
		cmdRelease(ctx, client)
	case "identity":
		cmdIdentity(ctx, client)
	case "list-folders":
		cmdListFolders(ctx, client)
	case "add-to-collection":
		cmdAddToCollection(ctx, client)
	case "mcp":
		if err := runMCP(client); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "version":
		fmt.Println(version)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func cmdSearchMaster(ctx context.Context, c *Client) {
	fs := flag.NewFlagSet("search-master", flag.ExitOnError)
	artist := fs.String("artist", "", "Artist name")
	album := fs.String("album", "", "Album/release title")
	fs.Parse(os.Args[2:])
	requireFlag("search-master", "artist", *artist)
	requireFlag("search-master", "album", *album)

	results, err := c.SearchMasters(ctx, *artist, *album)
	dieOnErr(err)
	writeJSON(results)
}

func cmdSearchRelease(ctx context.Context, c *Client) {
	fs := flag.NewFlagSet("search-release", flag.ExitOnError)
	artist := fs.String("artist", "", "Artist name")
	album := fs.String("album", "", "Album/release title")
	fs.Parse(os.Args[2:])
	requireFlag("search-release", "artist", *artist)
	requireFlag("search-release", "album", *album)

	results, err := c.SearchReleases(ctx, *artist, *album)
	dieOnErr(err)
	writeJSON(results)
}

func cmdVersions(ctx context.Context, c *Client) {
	fs := flag.NewFlagSet("versions", flag.ExitOnError)
	masterID := fs.Int("master", 0, "Master release ID")
	fs.Parse(os.Args[2:])
	if *masterID == 0 {
		fmt.Fprintln(os.Stderr, "error: --master is required")
		os.Exit(1)
	}

	versions, err := c.GetVersions(ctx, *masterID)
	dieOnErr(err)
	writeJSON(versions)
}

func cmdRelease(ctx context.Context, c *Client) {
	fs := flag.NewFlagSet("release", flag.ExitOnError)
	id := fs.Int("id", 0, "Release ID")
	fs.Parse(os.Args[2:])
	if *id == 0 {
		fmt.Fprintln(os.Stderr, "error: --id is required")
		os.Exit(1)
	}

	detail, err := c.GetRelease(ctx, *id)
	dieOnErr(err)
	writeJSON(detail)
}

func cmdIdentity(ctx context.Context, c *Client) {
	flag.CommandLine.Parse(os.Args[2:])
	id, err := c.GetIdentity(ctx)
	dieOnErr(err)
	writeJSON(id)
}

func cmdListFolders(ctx context.Context, c *Client) {
	fs := flag.NewFlagSet("list-folders", flag.ExitOnError)
	username := fs.String("username", "", "Discogs username")
	fs.Parse(os.Args[2:])

	if *username == "" {
		id, err := c.GetIdentity(ctx)
		dieOnErr(err)
		*username = id.Username
	}

	folders, err := c.GetFolders(ctx, *username)
	dieOnErr(err)
	writeJSON(folders)
}

func cmdAddToCollection(ctx context.Context, c *Client) {
	fs := flag.NewFlagSet("add-to-collection", flag.ExitOnError)
	username := fs.String("username", "", "Discogs username (auto-detected if omitted)")
	releaseID := fs.Int("release-id", 0, "Release ID to add")
	folderID := fs.Int("folder-id", 1, "Collection folder ID (default: 1 = Uncategorized)")
	notes := fs.String("notes", "", "Notes to save to the collection instance")
	fs.Parse(os.Args[2:])

	if *releaseID == 0 {
		fmt.Fprintln(os.Stderr, "error: --release-id is required")
		os.Exit(1)
	}

	if *username == "" {
		id, err := c.GetIdentity(ctx)
		dieOnErr(err)
		*username = id.Username
	}

	instance, err := c.AddToCollection(ctx, *username, *folderID, *releaseID)
	dieOnErr(err)

	if *notes != "" {
		if err := c.SetInstanceNote(ctx, *username, *folderID, *releaseID, instance.InstanceID, *notes); err != nil {
			fmt.Fprintf(os.Stderr, "warning: note not saved: %v\n", err)
		}
	}

	type result struct {
		*CollectionInstance
		Notes string `json:"notes,omitempty"`
	}
	writeJSON(result{instance, *notes})
}

func requireFlag(cmd, name, val string) {
	if val == "" {
		fmt.Fprintf(os.Stderr, "error: --%s is required for %s\n", name, cmd)
		os.Exit(1)
	}
}

func dieOnErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "error encoding JSON:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `wtd — Discogs data tool for vinyl identification

Usage: wtd <subcommand> [flags]

Subcommands:
  search-master   --artist STR --album STR
  search-release  --artist STR --album STR
  versions        --master INT
  release         --id INT
  identity
  list-folders    [--username STR]
  add-to-collection --release-id INT [--folder-id INT] [--username STR] [--notes STR]
  mcp             Start MCP server (stdio) for Claude Desktop
  version

Environment:
  DISCOGS_TOKEN   Personal access token from discogs.com/settings/developers
`)
}
