package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "search-master":
		fmt.Fprintln(os.Stderr, "not implemented yet")
		os.Exit(1)
	case "search-release":
		fmt.Fprintln(os.Stderr, "not implemented yet")
		os.Exit(1)
	case "versions":
		fmt.Fprintln(os.Stderr, "not implemented yet")
		os.Exit(1)
	case "release":
		fmt.Fprintln(os.Stderr, "not implemented yet")
		os.Exit(1)
	case "identity":
		fmt.Fprintln(os.Stderr, "not implemented yet")
		os.Exit(1)
	case "list-folders":
		fmt.Fprintln(os.Stderr, "not implemented yet")
		os.Exit(1)
	case "add-to-collection":
		fmt.Fprintln(os.Stderr, "not implemented yet")
		os.Exit(1)
	case "version":
		fmt.Println(version)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `wtd - Discogs data tool for vinyl identification

Usage: wtd <subcommand> [flags]

Subcommands:
  search-master   --artist STR --album STR
  search-release  --artist STR --album STR
  versions        --master INT
  release         --id INT
  identity
  list-folders    --username STR
  add-to-collection --username STR --release-id INT --folder-id INT [--notes STR]
  version

Environment:
  DISCOGS_TOKEN   Required. Personal access token from discogs.com/settings/developers
`)
}
