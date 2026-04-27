package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func runMCP(c *Client) error {
	s := server.NewMCPServer(
		"what-the-discogs",
		version,
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(false),
	)

	s.AddTool(toolSearchMasters(), handleSearchMasters(c))
	s.AddTool(toolSearchReleases(), handleSearchReleases(c))
	s.AddTool(toolSearchByMatrix(), handleSearchByMatrix(c))
	s.AddTool(toolGetVersions(), handleGetVersions(c))
	s.AddTool(toolGetRelease(), handleGetRelease(c))
	s.AddTool(toolGetIdentity(), handleGetIdentity(c))
	s.AddTool(toolListFolders(), handleListFolders(c))
	s.AddTool(toolAddToCollection(), handleAddToCollection(c))

	s.AddPrompt(promptIdentifyVinyl(), handleIdentifyVinyl())

	return server.ServeStdio(s)
}

// --- Tool definitions ---

func toolSearchMasters() mcp.Tool {
	return mcp.NewTool("search_masters",
		mcp.WithDescription("Search Discogs for master releases matching an artist and album name. Returns a list of masters (id, title, year, url). Use this first when identifying a record."),
		mcp.WithString("artist", mcp.Required(), mcp.Description("Artist or band name")),
		mcp.WithString("album", mcp.Required(), mcp.Description("Album or release title")),
	)
}

func toolSearchReleases() mcp.Tool {
	return mcp.NewTool("search_releases",
		mcp.WithDescription("Search Discogs for individual releases. Use as a fallback when search_masters returns no results — some one-off pressings have no master."),
		mcp.WithString("artist", mcp.Required(), mcp.Description("Artist or band name")),
		mcp.WithString("album", mcp.Required(), mcp.Description("Album or release title")),
	)
}

func toolSearchByMatrix() mcp.Tool {
	return mcp.NewTool("search_by_matrix",
		mcp.WithDescription("Search Discogs for releases by matrix/runout etching string. Use when the user can read matrix markings from the dead wax — this can identify the exact pressing in one step without iterating through get_release calls."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Matrix or runout etching string, e.g. \"XARL-7503\" or \"YEX 749\"")),
	)
}

func toolGetVersions() mcp.Tool {
	return mcp.NewTool("get_versions",
		mcp.WithDescription("Get all known pressings and versions of a master release. Returns country, year, label, catalogue number, and format for each. Pass country/year/format to pre-filter results and reduce the candidate set immediately."),
		mcp.WithNumber("master_id", mcp.Required(), mcp.Description("Master release ID from search_masters")),
		mcp.WithString("country", mcp.Description("Filter to a specific country, e.g. \"US\" or \"UK\"")),
		mcp.WithString("year", mcp.Description("Filter to a specific release year, e.g. \"1969\"")),
		mcp.WithString("format", mcp.Description("Filter to a specific format, e.g. \"Vinyl\" or \"LP\"")),
	)
}

func toolGetRelease() mcp.Tool {
	return mcp.NewTool("get_release",
		mcp.WithDescription("Get full detail for a specific release, including matrix/runout etchings, barcodes, pressing plant, and cover art. Use this when you need to distinguish between pressings that look identical in the versions list."),
		mcp.WithNumber("release_id", mcp.Required(), mcp.Description("Release ID")),
	)
}

func toolGetIdentity() mcp.Tool {
	return mcp.NewTool("get_identity",
		mcp.WithDescription("Get the Discogs username for the authenticated token. Use this before list_folders or add_to_collection when no username is known."),
	)
}

func toolListFolders() mcp.Tool {
	return mcp.NewTool("list_folders",
		mcp.WithDescription("List the user's Discogs collection folders. Call get_identity first if you don't have the username."),
		mcp.WithString("username", mcp.Description("Discogs username. If omitted, fetched automatically via get_identity.")),
	)
}

func toolAddToCollection() mcp.Tool {
	return mcp.NewTool("add_to_collection",
		mcp.WithDescription("Add an identified release to the user's Discogs collection. Only call this after the user has confirmed they want to add it."),
		mcp.WithNumber("release_id", mcp.Required(), mcp.Description("Release ID to add")),
		mcp.WithNumber("folder_id", mcp.Description("Collection folder ID. If omitted, defaults to 1 (Uncategorized)")),
		mcp.WithString("username", mcp.Description("Discogs username. If omitted, fetched automatically.")),
		mcp.WithString("notes", mcp.Description("Optional notes to save to the collection instance (stored in the Notes field).")),
	)
}

// --- Handlers ---

func handleSearchMasters(c *Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		artist, _ := args["artist"].(string)
		album, _ := args["album"].(string)

		results, err := c.SearchMasters(ctx, artist, album)
		if err != nil {
			return toolErr("search failed: %v", err), nil
		}
		if len(results) == 0 {
			return mcp.NewToolResultText("No master releases found. Try search_releases as a fallback."), nil
		}
		return toolJSON(results), nil
	}
}

func handleSearchReleases(c *Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		artist, _ := args["artist"].(string)
		album, _ := args["album"].(string)

		results, err := c.SearchReleases(ctx, artist, album)
		if err != nil {
			return toolErr("search failed: %v", err), nil
		}
		return toolJSON(results), nil
	}
}

func handleSearchByMatrix(c *Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		query, _ := args["query"].(string)
		if query == "" {
			return toolErr("query is required"), nil
		}
		results, err := c.SearchByMatrix(ctx, query)
		if err != nil {
			return toolErr("search failed: %v", err), nil
		}
		if len(results) == 0 {
			return mcp.NewToolResultText("No releases found matching that matrix string."), nil
		}
		return toolJSON(results), nil
	}
}

func handleGetVersions(c *Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		masterID := int(numArg(req, "master_id"))
		if masterID == 0 {
			return toolErr("master_id is required and must be non-zero"), nil
		}

		args := getArgs(req)
		filterCountry, _ := args["country"].(string)
		filterYear, _ := args["year"].(string)
		filterFormat, _ := args["format"].(string)

		versions, err := c.GetVersions(ctx, masterID)
		if err != nil {
			return toolErr("fetching versions: %v", err), nil
		}
		if filterCountry != "" || filterYear != "" || filterFormat != "" {
			versions = filterVersions(versions, filterCountry, filterYear, filterFormat)
		}
		return toolJSON(versions), nil
	}
}

func handleGetRelease(c *Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		releaseID := int(numArg(req, "release_id"))
		if releaseID == 0 {
			return toolErr("release_id is required and must be non-zero"), nil
		}

		detail, err := c.GetRelease(ctx, releaseID)
		if err != nil {
			return toolErr("fetching release: %v", err), nil
		}

		b, err := json.MarshalIndent(detail, "", "  ")
		if err != nil {
			return toolErr("marshaling release details: %v", err), nil
		}
		text := string(b)

		img := primaryImage(detail.Images)
		if img != "" {
			imgData, mimeType, err := c.FetchImageBase64(ctx, img)
			if err == nil {
				return mcp.NewToolResultImage(text, imgData, mimeType), nil
			}
			// Image fetch failed — fall through to text-only with the URL noted.
			text = fmt.Sprintf("Cover art: %s\n\n%s", img, text)
		}

		return mcp.NewToolResultText(text), nil
	}
}

func handleGetIdentity(c *Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := c.GetIdentity(ctx)
		if err != nil {
			return toolErr("fetching identity: %v", err), nil
		}
		return toolJSON(id), nil
	}
}

func handleListFolders(c *Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		username, _ := args["username"].(string)
		if username == "" {
			id, err := c.GetIdentity(ctx)
			if err != nil {
				return toolErr("fetching identity: %v", err), nil
			}
			username = id.Username
		}

		folders, err := c.GetFolders(ctx, username)
		if err != nil {
			return toolErr("fetching folders: %v", err), nil
		}
		return toolJSON(folders), nil
	}
}

func handleAddToCollection(c *Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		releaseID := int(numArg(req, "release_id"))
		folderID := int(numArg(req, "folder_id"))
		username, _ := args["username"].(string)
		notes, _ := args["notes"].(string)

		if releaseID == 0 {
			return toolErr("release_id is required and must be non-zero"), nil
		}
		if folderID == 0 {
			folderID = 1 // default: Uncategorized
		}

		if username == "" {
			id, err := c.GetIdentity(ctx)
			if err != nil {
				return toolErr("fetching identity: %v", err), nil
			}
			username = id.Username
		}

		instance, err := c.AddToCollection(ctx, username, folderID, releaseID)
		if err != nil {
			return toolErr("adding to collection: %v", err), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "Added to collection.\n")
		fmt.Fprintf(&sb, "Instance ID: %d\n", instance.InstanceID)
		fmt.Fprintf(&sb, "URL: https://www.discogs.com/user/%s/collection\n", username)
		if notes != "" {
			if err := c.SetInstanceNote(ctx, username, folderID, releaseID, instance.InstanceID, notes); err != nil {
				fmt.Fprintf(&sb, "Notes: %s (warning: could not save to Discogs: %v)\n", notes, err)
			} else {
				fmt.Fprintf(&sb, "Notes: %s\n", notes)
			}
		}
		return mcp.NewToolResultText(sb.String()), nil
	}
}

// --- Helpers ---

func toolJSON(v any) *mcp.CallToolResult {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return toolErr("encoding result: %v", err)
	}
	return mcp.NewToolResultText(string(b))
}

func toolErr(format string, args ...any) *mcp.CallToolResult {
	return mcp.NewToolResultError(fmt.Sprintf(format, args...))
}

func getArgs(req mcp.CallToolRequest) map[string]any {
	if args, ok := req.Params.Arguments.(map[string]any); ok {
		return args
	}
	return map[string]any{}
}

func numArg(req mcp.CallToolRequest, key string) float64 {
	if v, ok := getArgs(req)[key].(float64); ok {
		return v
	}
	return 0
}

func primaryImage(images []Image) string {
	for _, img := range images {
		if img.Type == "primary" {
			return img.URI
		}
	}
	if len(images) > 0 {
		return images[0].URI
	}
	return ""
}

// --- Prompt ---

func promptIdentifyVinyl() mcp.Prompt {
	return mcp.NewPrompt("identify_vinyl",
		mcp.WithPromptDescription("Identify the exact Discogs pressing of a vinyl record via guided Q&A."),
		mcp.WithArgument("hint",
			mcp.ArgumentDescription("Anything already known: artist, album, country, year, label, catalogue number, or matrix etchings. Leave blank to start from scratch."),
		),
	)
}

const identifyVinylWorkflow = `You are helping identify the exact Discogs pressing of a vinyl record.

## Step 1: Gather initial details

Ask the user for:
- Artist and album name (required to search)
- Anything else they know: country, year, label, catalogue number
- Whether they can read any text etched into the dead wax (the shiny ring between the last groove and the label) — these are matrix etchings, e.g. "XARL-7503" or "YEX 749-1"

If a hint was provided, use it as the starting point and skip questions already answered.

**If matrix etchings are available, go directly to Step 2b.**

## Step 2: Find master release

Call search_masters with the artist and album.
- 0 results → try search_releases as fallback; if still 0, ask the user to check spelling
- 1 result → auto-select and tell the user
- Multiple → show title + year for each, ask user to pick

If you used search_releases (no master found), skip to Step 4 using those results as candidates.

## Step 2b: Matrix search shortcut

If matrix etchings are known, call search_by_matrix with the etching string (e.g. "XARL-7503").
- Matches found → show results (title, country, year, label) and confirm with user, then jump to Step 6
- No matches → fall back to Step 2

## Step 3: Load versions

Call get_versions with the master ID. Pass any known country/year/format as filter parameters to pre-reduce the list:
- get_versions(master_id=X, country="US", year="1969")

Tell the user: "Found N versions — let me narrow these down."

## Step 4: Structured narrowing

Examine the versions. For each field that has multiple distinct non-empty values across candidates, ask the user about it:
- format (LP, Single, EP…)
- country
- label
- year (the released field)
- catno
- format_descriptions (Stereo, Mono, Reissue, Promo…)

Ask in order of fewest distinct values first (binary choices are easiest). For each question, list the actual options from the data and include "Not sure / skip". Filter candidates after each answer; show remaining count. If an answer produces 0 matches, skip that filter. Stop when ≤ 3 candidates remain or no more fields discriminate.

## Step 5: Matrix etching detail

For each remaining candidate (max 10), call get_release to get the identifiers array. Focus on entries with type "Matrix / Runout", "Matrix", or "Runout".

If matrix strings vary between candidates, explain to the user:
> Matrix etchings are text scratched into the shiny dead wax — the area between the last groove and the label. Look closely at an angle under a light. They typically look like XYZ-1A or ABCD 123-A.

Ask for Side A (and Side B if still needed). Match their input generously against known strings — spacing differences, minor typos, and extra handwritten characters are all fine.

If barcodes vary and the record is likely post-1985, ask for the barcode number.

## Step 6: Present result

Single match:
  Identified: ARTIST – ALBUM
  Year: YEAR  Country: COUNTRY  Label: LABEL  Cat#: CATNO
  Format: FORMAT DESCRIPTIONS
  Matrix A: ...  Matrix B: ...
  https://www.discogs.com/release/ID

2–5 matches: show a comparison table with key differentiating fields and a URL for each.
0 matches: explain what happened and offer to retry with relaxed constraints.

## Step 7: Add to collection (optional, single match only)

Ask if they want to add it to their Discogs collection. If yes:
1. Call get_identity to get the username
2. Call list_folders to show their folders
3. Ask which folder and whether to add notes
4. Call add_to_collection with release_id, folder_id, and notes

Confirm success and show the collection URL.

---
You are the intelligence here. The tools only fetch data. You decide which questions to ask and in what order. The user should never need to understand Discogs data structures.`

func handleIdentifyVinyl() server.PromptHandlerFunc {
	return func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		text := identifyVinylWorkflow
		if hint := req.Params.Arguments["hint"]; hint != "" {
			text = "Starting hint from the user: " + hint + "\n\n" + text
		}
		return &mcp.GetPromptResult{
			Description: "Identify the exact Discogs pressing of a vinyl record",
			Messages: []mcp.PromptMessage{
				{
					Role:    mcp.RoleUser,
					Content: mcp.TextContent{Type: "text", Text: text},
				},
			},
		}, nil
	}
}
