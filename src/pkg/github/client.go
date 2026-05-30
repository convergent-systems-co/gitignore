// Package github provides functionality to interact with GitHub repositories
// for fetching gitignore templates.
//
// Requests are authenticated when a token is available (GITHUB_TOKEN env var,
// else `gh auth token`), raising the rate limit from 60/hr to 5000/hr. Fetched
// templates and the template listing are cached on disk with a TTL so repeat
// reads avoid the network. On a rate-limit (403) or network failure the client
// degrades gracefully to any cached copy before failing hard.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// defaultCacheTTL is how long a cached entry is considered fresh. GitHub's
	// gitignore templates change rarely, so a week keeps reads cheap without
	// going meaningfully stale.
	defaultCacheTTL = 7 * 24 * time.Hour

	// cacheSchemaVersion namespaces the on-disk layout; bump it if the cache
	// file format changes so stale layouts are ignored rather than misread.
	cacheSchemaVersion = "v1"

	defaultAPIBaseURL = "https://api.github.com"
	defaultRawBaseURL = "https://raw.githubusercontent.com"

	ghAuthTokenTimeout = 5 * time.Second
)

// Client is a GitHub API client for fetching gitignore templates.
type Client struct {
	httpClient *http.Client
	repoURL    string
	owner      string
	repo       string
	branch     string

	// apiBaseURL and rawBaseURL are injectable for testing; they default to
	// the real GitHub hosts.
	apiBaseURL string
	rawBaseURL string

	// token authorizes requests when non-empty. It is never logged or printed.
	token string
	// tokenSet records that a caller pinned the token via WithToken, so an
	// explicit WithToken("") disables auth instead of triggering env/gh lookup.
	tokenSet bool

	cacheDir string
	cacheTTL time.Duration

	// warn receives human-readable degradation notices (e.g. cache fallback).
	// Defaults to stderr; never receives the token.
	warn io.Writer
}

// GitignoreFile represents a gitignore template file.
type GitignoreFile struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Category string `json:"category"`
}

// TreeResponse represents the GitHub API tree response.
type TreeResponse struct {
	SHA  string     `json:"sha"`
	URL  string     `json:"url"`
	Tree []TreeItem `json:"tree"`
}

// TreeItem represents an item in the GitHub tree.
type TreeItem struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
	Size int    `json:"size,omitempty"`
	URL  string `json:"url,omitempty"`
}

// Option configures a Client. Options are applied in order in NewClient.
type Option func(*Client)

// WithBaseURL overrides the GitHub API base URL (for testing).
func WithBaseURL(u string) Option {
	return func(c *Client) { c.apiBaseURL = strings.TrimSuffix(u, "/") }
}

// WithRawBaseURL overrides the raw content base URL (for testing).
func WithRawBaseURL(u string) Option {
	return func(c *Client) { c.rawBaseURL = strings.TrimSuffix(u, "/") }
}

// WithCacheDir sets the on-disk cache directory, bypassing the default
// resolution. Tests inject a temp dir here so they never touch the real cache.
func WithCacheDir(dir string) Option {
	return func(c *Client) { c.cacheDir = dir }
}

// WithToken sets the auth token explicitly, bypassing env/gh resolution. An
// empty string disables authentication (used by tests to avoid ambient creds).
func WithToken(tok string) Option {
	return func(c *Client) {
		c.token = tok
		c.tokenSet = true
	}
}

// WithWarnWriter overrides where degradation notices are written.
func WithWarnWriter(w io.Writer) Option {
	return func(c *Client) { c.warn = w }
}

// NewClient creates a new GitHub client from a repository URL.
func NewClient(repoURL string, opts ...Option) (*Client, error) {
	owner, repo, err := parseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}
	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		repoURL:    repoURL,
		owner:      owner,
		repo:       repo,
		branch:     "main",
		apiBaseURL: defaultAPIBaseURL,
		rawBaseURL: defaultRawBaseURL,
		cacheTTL:   defaultCacheTTL,
		warn:       os.Stderr,
	}

	for _, opt := range opts {
		opt(c)
	}
	// Resolve a token from env/gh only when the caller did not pin one. An
	// explicit WithToken("") leaves auth disabled (tests rely on this).
	if !c.tokenSet {
		c.token = resolveToken()
	}

	if c.cacheDir == "" {
		c.cacheDir = defaultCacheDir()
	}
	return c, nil
}

func parseRepoURL(repoURL string) (owner, repo string, err error) {
	repoURL = strings.TrimSuffix(repoURL, ".git")
	if strings.Contains(repoURL, "github.com/") {
		parts := strings.Split(repoURL, "github.com/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub URL: %s", repoURL)
		}
		pathParts := strings.Split(strings.Trim(parts[1], "/"), "/")
		if len(pathParts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub URL: %s", repoURL)
		}
		return pathParts[0], pathParts[1], nil
	}
	if strings.HasPrefix(repoURL, "git@github.com:") {
		path := strings.TrimPrefix(repoURL, "git@github.com:")
		pathParts := strings.Split(path, "/")
		if len(pathParts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub SSH URL: %s", repoURL)
		}
		return pathParts[0], pathParts[1], nil
	}
	return "", "", fmt.Errorf("unsupported URL format: %s", repoURL)
}

// resolveToken returns an auth token from GITHUB_TOKEN, falling back to
// `gh auth token`. The token is never logged. Returns "" when none is found.
func resolveToken() string {
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		return tok
	}
	if tok := strings.TrimSpace(os.Getenv("GH_TOKEN")); tok != "" {
		return tok
	}
	return ghAuthToken()
}

// ghAuthToken shells out to the GitHub CLI for a token. Any failure (gh not
// installed, not logged in) yields "" — authentication is best-effort.
func ghAuthToken() string {
	if _, err := exec.LookPath("gh"); err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), ghAuthTokenTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// get issues an authenticated GET. The Authorization header is set only when a
// token is present, so we never send an empty Bearer.
func (c *Client) get(rawURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	return c.httpClient.Do(req)
}

// isRateLimited reports whether a response represents a rate-limit rejection.
func isRateLimited(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return false
	}
	// A 403 with remaining==0 is the canonical rate-limit signal, but GitHub
	// also returns plain 403/429 under secondary limits; treat both as such.
	return true
}

// ListGitignoreFiles returns all gitignore files in the repository, preferring
// a fresh cache entry and degrading to a stale one on rate-limit/network error.
func (c *Client) ListGitignoreFiles() ([]GitignoreFile, error) {
	if files, ok := c.readListCache(); ok {
		return files, nil
	}

	files, err := c.fetchListFromAPI()
	if err != nil {
		if stale, ok := c.readListCacheStale(); ok {
			c.warnf("github: %v; using cached template list", err)
			return stale, nil
		}
		return nil, err
	}
	c.writeListCache(files)
	return files, nil
}

// fetchListFromAPI fetches the repository tree and extracts gitignore files. It
// retries on the alternate default branch (master) when main 404s.
func (c *Client) fetchListFromAPI() ([]GitignoreFile, error) {
	resp, err := c.get(c.treeURL(c.branch))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository tree: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		c.branch = "master"
		resp, err = c.get(c.treeURL(c.branch))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch repository tree: %w", err)
		}
		defer resp.Body.Close()
	}

	if isRateLimited(resp) {
		return nil, fmt.Errorf("GitHub API rate limit reached (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(body))
	}

	var tree TreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var files []GitignoreFile
	gitignoreRegex := regexp.MustCompile(`(?i)\.gitignore$`)
	for _, item := range tree.Tree {
		if item.Type != "blob" || !gitignoreRegex.MatchString(item.Path) {
			continue
		}
		files = append(files, parseGitignorePath(item.Path))
	}
	return files, nil
}

func (c *Client) treeURL(branch string) string {
	return fmt.Sprintf("%s/repos/%s/%s/git/trees/%s?recursive=1",
		c.apiBaseURL, url.PathEscape(c.owner), url.PathEscape(c.repo), url.PathEscape(branch))
}

func parseGitignorePath(path string) GitignoreFile {
	parts := strings.Split(path, "/")
	filename := parts[len(parts)-1]
	name := strings.TrimSuffix(filename, ".gitignore")
	category := ""
	if len(parts) > 1 {
		category = strings.Join(parts[:len(parts)-1], "/")
	}
	return GitignoreFile{Name: name, Path: path, Category: category}
}

// GetGitignoreContent fetches the content of a specific gitignore file,
// preferring a fresh cache entry and degrading to a stale one on failure.
func (c *Client) GetGitignoreContent(file GitignoreFile) (string, error) {
	if content, ok := c.readContentCache(file.Path); ok {
		return content, nil
	}

	content, err := c.fetchContentFromAPI(file)
	if err != nil {
		if stale, ok := c.readContentCacheStale(file.Path); ok {
			c.warnf("github: %v; using cached copy of %s", err, file.Path)
			return stale, nil
		}
		return "", err
	}
	c.writeContentCache(file.Path, content)
	return content, nil
}

func (c *Client) fetchContentFromAPI(file GitignoreFile) (string, error) {
	rawURL := fmt.Sprintf("%s/%s/%s/%s/%s",
		c.rawBaseURL, url.PathEscape(c.owner), url.PathEscape(c.repo),
		url.PathEscape(c.branch), file.Path)
	resp, err := c.get(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch gitignore content: %w", err)
	}
	defer resp.Body.Close()

	if isRateLimited(resp) {
		return "", fmt.Errorf("GitHub rate limit reached fetching %s (status %d)", file.Path, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch gitignore content (status %d)", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read gitignore content: %w", err)
	}
	return string(content), nil
}

// FindGitignoreFile finds a gitignore file by name (case-insensitive).
func (c *Client) FindGitignoreFile(name string) (*GitignoreFile, error) {
	files, err := c.ListGitignoreFiles()
	if err != nil {
		return nil, err
	}

	nameLower := strings.ToLower(name)
	for _, file := range files {
		if strings.ToLower(file.Name) == nameLower {
			return &file, nil
		}
	}
	for _, file := range files {
		fullName := file.Name
		if file.Category != "" {
			fullName = file.Category + "/" + file.Name
		}
		if strings.ToLower(fullName) == nameLower {
			return &file, nil
		}
	}
	return nil, fmt.Errorf("gitignore template '%s' not found", name)
}

// Owner returns the repository owner.
func (c *Client) Owner() string { return c.owner }

// Repo returns the repository name.
func (c *Client) Repo() string { return c.repo }

func (c *Client) warnf(format string, args ...any) {
	if c.warn == nil {
		return
	}
	fmt.Fprintf(c.warn, format+"\n", args...)
}

// defaultCacheDir resolves the on-disk cache root, preferring os.UserCacheDir
// (which honours XDG_CACHE_HOME on Linux). Falls back to a temp dir if the
// home directory is unresolvable so the client still functions.
func defaultCacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "gitignore")
}
