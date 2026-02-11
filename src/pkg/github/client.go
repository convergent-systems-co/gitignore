// Package github provides functionality to interact with GitHub repositories
// for fetching gitignore templates
package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Client is a GitHub API client for fetching gitignore templates
type Client struct {
	httpClient *http.Client
	repoURL    string
	owner      string
	repo       string
	branch     string
}

// GitignoreFile represents a gitignore template file
type GitignoreFile struct {
	Name     string
	Path     string
	Category string
}

// TreeResponse represents the GitHub API tree response
type TreeResponse struct {
	SHA  string     `json:"sha"`
	URL  string     `json:"url"`
	Tree []TreeItem `json:"tree"`
}

// TreeItem represents an item in the GitHub tree
type TreeItem struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
	Size int    `json:"size,omitempty"`
	URL  string `json:"url,omitempty"`
}

// NewClient creates a new GitHub client from a repository URL
func NewClient(repoURL string) (*Client, error) {
	owner, repo, err := parseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		repoURL:    repoURL,
		owner:      owner,
		repo:       repo,
		branch:     "main",
	}, nil
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

// ListGitignoreFiles returns all gitignore files in the repository
func (c *Client) ListGitignoreFiles() ([]GitignoreFile, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1",
		url.PathEscape(c.owner), url.PathEscape(c.repo), url.PathEscape(c.branch))
	resp, err := c.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository tree: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		c.branch = "master"
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1",
			url.PathEscape(c.owner), url.PathEscape(c.repo), url.PathEscape(c.branch))
		resp2, err := c.httpClient.Get(apiURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch repository tree: %w", err)
		}
		defer resp2.Body.Close()
		resp = resp2
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
		file := parseGitignorePath(item.Path)
		files = append(files, file)
	}
	return files, nil
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

// GetGitignoreContent fetches the content of a specific gitignore file
func (c *Client) GetGitignoreContent(file GitignoreFile) (string, error) {
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
		url.PathEscape(c.owner), url.PathEscape(c.repo), url.PathEscape(c.branch), file.Path)
	resp, err := c.httpClient.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch gitignore content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch gitignore content (status %d)", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read gitignore content: %w", err)
	}
	return string(content), nil
}

// FindGitignoreFile finds a gitignore file by name (case-insensitive)
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

// Owner returns the repository owner
func (c *Client) Owner() string {
	return c.owner
}

// Repo returns the repository name
func (c *Client) Repo() string {
	return c.repo
}
