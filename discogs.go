package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	baseURL   = "https://api.discogs.com"
	userAgent = "WhatTheDiscogs/1.0 +https://github.com/richardthe3rd/what-the-discogs"
)

type Client struct {
	token   string
	http    *http.Client
	cache   map[string][]byte
	cacheMu sync.RWMutex
	limiter *rate.Limiter
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
		cache: make(map[string][]byte),
		// 1 token/sec refill, burst of 3 — allows the first few calls in a
		// session to fire immediately before throttling to 1/sec. Well within
		// the 25/sec authenticated Discogs limit.
		limiter: rate.NewLimiter(rate.Every(time.Second), 3),
	}
}

func (c *Client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	u := baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	c.cacheMu.RLock()
	if cached, ok := c.cache[u]; ok {
		out := make([]byte, len(cached))
		copy(out, cached)
		c.cacheMu.RUnlock()
		return out, nil
	}
	c.cacheMu.RUnlock()

	backoff := 2 * time.Second
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, err // context cancelled
		}
		body, err := c.doGet(ctx, u)
		if err == nil {
			out := make([]byte, len(body))
			copy(out, body)
			c.cacheMu.Lock()
			c.cache[u] = body
			c.cacheMu.Unlock()
			return out, nil
		}
		lastErr = err
		if !isRateLimit(err) || attempt == 3 {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}
	return nil, lastErr
}

func (c *Client) doGet(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Discogs token="+c.token)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", u, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == 429 {
		return nil, &rateLimitError{}
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, u, extractMessage(body))
	}
	return body, nil
}

type rateLimitError struct{}

func (e *rateLimitError) Error() string { return "rate limited (429)" }

func isRateLimit(err error) bool {
	_, ok := err.(*rateLimitError)
	return ok
}

func extractMessage(body []byte) string {
	var m struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &m) == nil && m.Message != "" {
		return m.Message
	}
	return string(body)
}

// SearchMasters searches for master releases matching artist and title.
func (c *Client) SearchMasters(ctx context.Context, artist, title string) ([]MasterResult, error) {
	params := url.Values{
		"artist": {artist},
		"q":      {title},
		"type":   {"master"},
	}
	body, err := c.get(ctx, "/database/search", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Results []struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
			Year  string `json:"year"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}

	limit := len(resp.Results)
	if limit > 10 {
		limit = 10
	}
	results := make([]MasterResult, 0, limit)
	for _, r := range resp.Results[:limit] {
		year, _ := strconv.Atoi(r.Year)
		results = append(results, MasterResult{
			ID:    r.ID,
			Title: r.Title,
			Year:  year,
			URL:   fmt.Sprintf("https://www.discogs.com/master/%d", r.ID),
		})
	}
	return results, nil
}

// SearchReleases searches for releases (fallback when no master exists).
func (c *Client) SearchReleases(ctx context.Context, artist, title string) ([]Version, error) {
	params := url.Values{
		"artist": {artist},
		"q":      {title},
		"type":   {"release"},
	}
	body, err := c.get(ctx, "/database/search", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Results []struct {
			ID      int      `json:"id"`
			Title   string   `json:"title"`
			Label   []string `json:"label"`
			Country string   `json:"country"`
			Year    string   `json:"year"`
			CatNo   string   `json:"catno"`
			Format  []string `json:"format"`
			Thumb   string   `json:"thumb"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}

	limit := len(resp.Results)
	if limit > 10 {
		limit = 10
	}
	versions := make([]Version, 0, limit)
	for _, r := range resp.Results[:limit] {
		label := ""
		if len(r.Label) > 0 {
			label = r.Label[0]
		}
		format, descs := splitFormat(r.Format)
		versions = append(versions, Version{
			ID:          r.ID,
			Title:       r.Title,
			Label:       label,
			Country:     r.Country,
			Year:        r.Year,
			CatNo:       r.CatNo,
			Format:      format,
			FormatDescs: descs,
			Thumb:       r.Thumb,
			ResourceURL: fmt.Sprintf("https://api.discogs.com/releases/%d", r.ID),
		})
	}
	return versions, nil
}

// GetVersions returns all versions of a master release, paginated.
func (c *Client) GetVersions(ctx context.Context, masterID int) ([]Version, error) {
	var all []Version
	page := 1

	for {
		params := url.Values{
			"page":     {strconv.Itoa(page)},
			"per_page": {"500"},
		}
		body, err := c.get(ctx, fmt.Sprintf("/masters/%d/versions", masterID), params)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Versions []struct {
				ID           int      `json:"id"`
				Title        string   `json:"title"`
				Label        string   `json:"label"`
				Country      string   `json:"country"`
				Released     string   `json:"released"`
				CatNo        string   `json:"catno"`
				Format       string   `json:"format"`
				MajorFormats []string `json:"major_formats"`
				Thumb        string   `json:"thumb"`
				ResourceURL  string   `json:"resource_url"`
			} `json:"versions"`
			Pagination Pagination `json:"pagination"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parsing versions page %d: %w", page, err)
		}

		for _, v := range resp.Versions {
			all = append(all, Version{
				ID:          v.ID,
				Title:       v.Title,
				Label:       v.Label,
				Country:     v.Country,
				Year:        v.Released,
				CatNo:       v.CatNo,
				Format:      primaryFormat(v.MajorFormats, v.Format),
				FormatDescs: parseFormatDescs(v.Format, v.MajorFormats),
				Thumb:       v.Thumb,
				ResourceURL: v.ResourceURL,
			})
		}

		if len(all) > 500 && len(all) <= 500+len(resp.Versions) {
			// Emit a warning once when we cross 500 — the caller (Claude) will
			// see this in stderr and can mention it to the user.
			fmt.Fprintf(os.Stderr, "warning: %d versions found for master %d; large sets may take longer to process\n", resp.Pagination.Items, masterID)
		}
		if page >= resp.Pagination.Pages || len(all) >= 1500 {
			if len(all) >= 1500 {
				fmt.Fprintf(os.Stderr, "warning: version list truncated at %d; master %d has %d total\n", len(all), masterID, resp.Pagination.Items)
			}
			break
		}
		page++
	}
	return all, nil
}

// GetRelease returns full release detail for a single release.
func (c *Client) GetRelease(ctx context.Context, releaseID int) (*ReleaseDetail, error) {
	body, err := c.get(ctx, fmt.Sprintf("/releases/%d", releaseID), nil)
	if err != nil {
		return nil, err
	}

	var raw struct {
		ID      int    `json:"id"`
		Title   string `json:"title"`
		Year    int    `json:"year"`
		Country string `json:"country"`
		Labels  []struct {
			Name  string `json:"name"`
			CatNo string `json:"catno"`
			ID    int    `json:"id"`
		} `json:"labels"`
		Formats []struct {
			Name         string   `json:"name"`
			Qty          string   `json:"qty"`
			Descriptions []string `json:"descriptions"`
			Text         string   `json:"text"`
		} `json:"formats"`
		Identifiers []Identifier `json:"identifiers"`
		Companies   []struct {
			Name           string `json:"name"`
			EntityTypeName string `json:"entity_type_name"`
		} `json:"companies"`
		Images []struct {
			URI    string `json:"uri"`
			Type   string `json:"type"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"images"`
		Notes string `json:"notes"`
		URI   string `json:"uri"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing release: %w", err)
	}

	labels := make([]Label, len(raw.Labels))
	for i, l := range raw.Labels {
		labels[i] = Label{Name: l.Name, CatNo: l.CatNo, EntityID: l.ID}
	}
	formats := make([]Format, len(raw.Formats))
	for i, f := range raw.Formats {
		formats[i] = Format{Name: f.Name, Qty: f.Qty, Descriptions: f.Descriptions, Text: f.Text}
	}
	companies := make([]Company, len(raw.Companies))
	for i, co := range raw.Companies {
		companies[i] = Company{Name: co.Name, EntityTypeName: co.EntityTypeName}
	}
	images := make([]Image, len(raw.Images))
	for i, img := range raw.Images {
		images[i] = Image{URI: img.URI, Type: img.Type, Width: img.Width, Height: img.Height}
	}

	return &ReleaseDetail{
		ID:          raw.ID,
		Title:       raw.Title,
		Year:        raw.Year,
		Country:     raw.Country,
		Labels:      labels,
		Formats:     formats,
		Identifiers: raw.Identifiers,
		Companies:   companies,
		Images:      images,
		Notes:       raw.Notes,
		URL:         raw.URI,
	}, nil
}

// GetIdentity returns the authenticated user's identity.
func (c *Client) GetIdentity(ctx context.Context) (*Identity, error) {
	body, err := c.get(ctx, "/oauth/identity", nil)
	if err != nil {
		return nil, err
	}
	var id Identity
	if err := json.Unmarshal(body, &id); err != nil {
		return nil, fmt.Errorf("parsing identity: %w", err)
	}
	return &id, nil
}

// GetFolders returns the user's collection folders.
func (c *Client) GetFolders(ctx context.Context, username string) ([]Folder, error) {
	body, err := c.get(ctx, "/users/"+url.PathEscape(username)+"/collection/folders", nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Folders []Folder `json:"folders"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing folders: %w", err)
	}
	return resp.Folders, nil
}

// AddToCollection adds a release to the user's collection.
func (c *Client) AddToCollection(ctx context.Context, username string, folderID, releaseID int) (*CollectionInstance, error) {
	u := fmt.Sprintf("%s/users/%s/collection/folders/%d/releases/%d",
		baseURL, url.PathEscape(username), folderID, releaseID)

	backoff := 2 * time.Second
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, "POST", u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Discogs token="+c.token)
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("POST %s: %w", u, err)
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("reading response: %w", readErr)
		}

		if resp.StatusCode == 429 {
			lastErr = &rateLimitError{}
			if attempt == 3 {
				break
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			continue
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, extractMessage(body))
		}

		var instance CollectionInstance
		if err := json.Unmarshal(body, &instance); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}
		return &instance, nil
	}
	return nil, lastErr
}

// GetCollectionFields returns the user's collection fields (Media Condition,
// Sleeve Condition, Notes, and any custom fields).
func (c *Client) GetCollectionFields(ctx context.Context, username string) ([]CollectionField, error) {
	body, err := c.get(ctx, "/users/"+url.PathEscape(username)+"/collection/fields", nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Fields []CollectionField `json:"fields"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing collection fields: %w", err)
	}
	return resp.Fields, nil
}

// SetInstanceNote stores a text note on a collection instance. It resolves the
// Notes field ID dynamically via the collection fields endpoint so it works
// regardless of whether the user has customised their field ordering.
func (c *Client) SetInstanceNote(ctx context.Context, username string, folderID, releaseID, instanceID int, note string) error {
	fields, err := c.GetCollectionFields(ctx, username)
	if err != nil {
		return fmt.Errorf("resolving Notes field: %w", err)
	}
	fieldID := 0
	for _, f := range fields {
		if f.Type == "textarea" && f.Name == "Notes" {
			fieldID = f.ID
			break
		}
	}
	if fieldID == 0 {
		return fmt.Errorf("Notes field not found in collection fields")
	}

	u := fmt.Sprintf("%s/users/%s/collection/folders/%d/releases/%d/instances/%d/fields/%d",
		baseURL, url.PathEscape(username), folderID, releaseID, instanceID, fieldID)

	payload, err := json.Marshal(map[string]string{"value": note})
	if err != nil {
		return err
	}

	if err := c.limiter.Wait(ctx); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Discogs token="+c.token)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", u, err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body) // drain

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

// FetchImageBase64 downloads an image and returns its base64-encoded data and
// MIME type. Discogs CDN images require the same auth token as API calls.
func (c *Client) FetchImageBase64(ctx context.Context, imageURL string) (string, string, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Discogs token="+c.token)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("fetching image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("reading image: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}

	return base64.StdEncoding.EncodeToString(body), ct, nil
}

func primaryFormat(majorFormats []string, formatStr string) string {
	if len(majorFormats) > 0 {
		return majorFormats[0]
	}
	if i := strings.Index(formatStr, ","); i >= 0 {
		return strings.TrimSpace(formatStr[:i])
	}
	return formatStr
}

func parseFormatDescs(formatStr string, majorFormats []string) []string {
	parts := strings.Split(formatStr, ",")
	skip := make(map[string]bool)
	for _, mf := range majorFormats {
		skip[strings.TrimSpace(mf)] = true
	}
	var descs []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && !skip[p] {
			descs = append(descs, p)
		}
	}
	return descs
}

func splitFormat(formats []string) (primary string, descs []string) {
	if len(formats) == 0 {
		return "", nil
	}
	primary = formats[0]
	if len(formats) > 1 {
		descs = formats[1:]
	}
	return
}
