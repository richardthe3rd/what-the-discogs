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
	)

	s.AddTool(toolSearchMasters(), handleSearchMasters(c))
	s.AddTool(toolSearchReleases(), handleSearchReleases(c))
	s.AddTool(toolSearchByMatrix(), handleSearchByMatrix(c))
	s.AddTool(toolGetVersions(), handleGetVersions(c))
	s.AddTool(toolGetRelease(), handleGetRelease(c))
	s.AddTool(toolGetIdentity(), handleGetIdentity(c))
	s.AddTool(toolListFolders(), handleListFolders(c))
	s.AddTool(toolAddToCollection(), handleAddToCollection(c))

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
