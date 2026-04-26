package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
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

	switch os.Args[1] {
	case "search-master":
		cmdSearchMaster(client)
	case "search-release":
		cmdSearchRelease(client)
	case "versions":
		cmdVersions(client)
	case "release":
		cmdRelease(client)
	case "identity":
		cmdIdentity(client)
	case "list-folders":
		cmdListFolders(client)
	case "add-to-collection":
		cmdAddToCollection(client)
	case "version":
		fmt.Println(version)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func cmdSearchMaster(c *Client) {
	fs := flag.NewFlagSet("search-master", flag.ExitOnError)
	artist := fs.String("artist", "", "Artist name")
	album := fs.String("album", "", "Album/release title")
	fs.Parse(os.Args[2:])
	requireFlag("search-master", "artist", *artist)
	requireFlag("search-master", "album", *album)

	results, err := c.SearchMasters(*artist, *album)
	dieOnErr(err)
	writeJSON(results)
}

func cmdSearchRelease(c *Client) {
	fs := flag.NewFlagSet("search-release", flag.ExitOnError)
	artist := fs.String("artist", "", "Artist name")
	album := fs.String("album", "", "Album/release title")
	fs.Parse(os.Args[2:])
	requireFlag("search-release", "artist", *artist)
	requireFlag("search-release", "album", *album)

	results, err := c.SearchReleases(*artist, *album)
	dieOnErr(err)
	writeJSON(results)
}

func cmdVersions(c *Client) {
	fs := flag.NewFlagSet("versions", flag.ExitOnError)
	masterID := fs.Int("master", 0, "Master release ID")
	fs.Parse(os.Args[2:])
	if *masterID == 0 {
		fmt.Fprintln(os.Stderr, "error: --master is required")
		os.Exit(1)
	}

	versions, err := c.GetVersions(*masterID)
	dieOnErr(err)
	writeJSON(versions)
}

func cmdRelease(c *Client) {
	fs := flag.NewFlagSet("release", flag.ExitOnError)
	id := fs.Int("id", 0, "Release ID")
	fs.Parse(os.Args[2:])
	if *id == 0 {
		fmt.Fprintln(os.Stderr, "error: --id is required")
		os.Exit(1)
	}

	detail, err := c.GetRelease(*id)
	dieOnErr(err)
	writeJSON(detail)
}

func cmdIdentity(c *Client) {
	flag.CommandLine.Parse(os.Args[2:])
	id, err := c.GetIdentity()
	dieOnErr(err)
	writeJSON(id)
}

func cmdListFolders(c *Client) {
	fs := flag.NewFlagSet("list-folders", flag.ExitOnError)
	username := fs.String("username", "", "Discogs username")
	fs.Parse(os.Args[2:])

	if *username == "" {
		// Auto-detect from identity.
		id, err := c.GetIdentity()
		dieOnErr(err)
		*username = id.Username
	}

	folders, err := c.GetFolders(*username)
	dieOnErr(err)
	writeJSON(folders)
}

func cmdAddToCollection(c *Client) {
	fs := flag.NewFlagSet("add-to-collection", flag.ExitOnError)
	username := fs.String("username", "", "Discogs username (optional; auto-detected)")
	releaseID := fs.Int("release-id", 0, "Release ID to add")
	folderID := fs.Int("folder-id", 1, "Collection folder ID (default: 1 = Uncategorized)")
	notes := fs.String("notes", "", "Notes to attach (informational; stored locally)")
	fs.Parse(os.Args[2:])

	if *releaseID == 0 {
		fmt.Fprintln(os.Stderr, "error: --release-id is required")
		os.Exit(1)
	}

	if *username == "" {
		id, err := c.GetIdentity()
		dieOnErr(err)
		*username = id.Username
	}

	instance, err := c.AddToCollection(*username, *folderID, *releaseID)
	dieOnErr(err)

	// Notes aren't supported by the Discogs collection API at add time;
	// surface them in the output so the skill can show them to the user.
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

// intArg is a helper used by tests.
func intArg(s string) int {
	n, _ := strconv.Atoi(s)
	return n
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
  version

Environment:
  DISCOGS_TOKEN   Personal access token from discogs.com/settings/developers
`)
}
