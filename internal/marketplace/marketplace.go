// Package marketplace provides agent template discovery and installation from the Volra marketplace.
package marketplace

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// DefaultIndexURL points to the marketplace index on GitHub.
	DefaultIndexURL = "https://raw.githubusercontent.com/romerox3/volra-marketplace/main/index.json"

	// CacheTTL defines how long the cached index stays valid.
	CacheTTL = 1 * time.Hour
)

// Template represents a single marketplace template entry.
type Template struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Framework   string   `json:"framework"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
	RepoURL     string   `json:"repo_url"`
	Version     string   `json:"version,omitempty"`
}

// Index represents the full marketplace index.
type Index struct {
	Templates []Template `json:"templates"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Client handles marketplace operations.
type Client struct {
	IndexURL   string
	CacheDir   string
	HTTPClient *http.Client
}

// NewClient creates a marketplace client with defaults.
func NewClient() *Client {
	home, _ := os.UserHomeDir()
	return &Client{
		IndexURL:   DefaultIndexURL,
		CacheDir:   filepath.Join(home, ".volra"),
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// cachePath returns the local cache file path.
func (c *Client) cachePath() string {
	return filepath.Join(c.CacheDir, "marketplace-cache.json")
}

// FetchIndex retrieves the marketplace index, using cache if valid.
func (c *Client) FetchIndex() (*Index, error) {
	// Try cache first.
	if idx, err := c.loadCache(); err == nil {
		return idx, nil
	}

	// Fetch from remote.
	resp, err := c.HTTPClient.Get(c.IndexURL)
	if err != nil {
		return nil, fmt.Errorf("fetching marketplace index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace index returned HTTP %d", resp.StatusCode)
	}

	// Limit index size to 10MB to prevent OOM from malicious servers.
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("reading marketplace index: %w", err)
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing marketplace index: %w", err)
	}

	// Write cache.
	_ = c.writeCache(data)

	return &idx, nil
}

// loadCache reads and validates the local cache.
func (c *Client) loadCache() (*Index, error) {
	info, err := os.Stat(c.cachePath())
	if err != nil {
		return nil, err
	}

	if time.Since(info.ModTime()) > CacheTTL {
		return nil, fmt.Errorf("cache expired")
	}

	data, err := os.ReadFile(c.cachePath())
	if err != nil {
		return nil, err
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

// writeCache stores the index locally.
func (c *Client) writeCache(data []byte) error {
	if err := os.MkdirAll(c.CacheDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(c.cachePath(), data, 0o644)
}

// Search returns templates matching the query (case-insensitive, matches name/description/tags).
func Search(idx *Index, query string) []Template {
	if query == "" {
		return idx.Templates
	}

	q := strings.ToLower(query)
	var results []Template

	for _, t := range idx.Templates {
		if matches(t, q) {
			results = append(results, t)
		}
	}
	return results
}

// Lookup finds a template by exact name (case-insensitive).
func Lookup(idx *Index, name string) (*Template, bool) {
	n := strings.ToLower(name)
	for _, t := range idx.Templates {
		if strings.ToLower(t.Name) == n {
			return &t, true
		}
	}
	return nil, false
}

// matches checks if a template matches the search query.
func matches(t Template, query string) bool {
	if strings.Contains(strings.ToLower(t.Name), query) {
		return true
	}
	if strings.Contains(strings.ToLower(t.Description), query) {
		return true
	}
	for _, tag := range t.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}
