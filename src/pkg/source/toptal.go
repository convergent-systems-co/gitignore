// Package source provides abstraction for different gitignore template sources
package source

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ToptalSource handles templates from the Toptal gitignore API
type ToptalSource struct {
	httpClient *http.Client
	baseURL    string
}

// NewToptalSource creates a new Toptal source with the default URL
func NewToptalSource() *ToptalSource {
	return &ToptalSource{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://www.toptal.com/developers/gitignore/api",
	}
}

// NewToptalSourceWithURL creates a Toptal source with a custom base URL
func NewToptalSourceWithURL(baseURL string) *ToptalSource {
	return &ToptalSource{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
	}
}

// Name returns the source name
func (t *ToptalSource) Name() string {
	return "toptal"
}

// BaseURL returns the base URL for the Toptal API
func (t *ToptalSource) BaseURL() string {
	return t.baseURL
}

// List returns all available templates from Toptal
func (t *ToptalSource) List() ([]TemplateFile, error) {
	listURL := fmt.Sprintf("%s/list", t.baseURL)
	resp, err := t.httpClient.Get(listURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Toptal template list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Toptal API error (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Toptal response: %w", err)
	}

	// Toptal returns a comma-separated or newline-separated list
	content := string(body)
	// Replace newlines with commas for consistent parsing
	content = strings.ReplaceAll(content, "\n", ",")
	content = strings.ReplaceAll(content, "\r", "")

	var files []TemplateFile
	for _, name := range strings.Split(content, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		files = append(files, TemplateFile{
			Name:     name,
			Path:     name,
			Category: "",
			Source:   "toptal",
		})
	}

	return files, nil
}

// Get returns the content of a template by name
func (t *ToptalSource) Get(name string) (*TemplateFile, string, error) {
	file, err := t.Find(name)
	if err != nil {
		return nil, "", err
	}

	contentURL := fmt.Sprintf("%s/%s", t.baseURL, url.PathEscape(name))
	resp, err := t.httpClient.Get(contentURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch Toptal template content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", fmt.Errorf("Toptal template '%s' not found", name)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Toptal API error (status %d)", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read Toptal template: %w", err)
	}

	return file, string(content), nil
}

// Find finds a template by name (case-insensitive)
func (t *ToptalSource) Find(name string) (*TemplateFile, error) {
	files, err := t.List()
	if err != nil {
		return nil, err
	}

	nameLower := strings.ToLower(name)
	for _, file := range files {
		if strings.ToLower(file.Name) == nameLower {
			return &file, nil
		}
	}

	return nil, fmt.Errorf("Toptal template '%s' not found", name)
}
