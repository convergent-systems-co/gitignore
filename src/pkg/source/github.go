// Package source provides abstraction for different gitignore template sources
package source

import (
	"github.com/polliard/gitignore/src/pkg/github"
)

// GitHubSource wraps the github.Client to implement the Source interface
type GitHubSource struct {
	client *github.Client
	url    string
}

// NewGitHubSource creates a new GitHub source from a repository URL
func NewGitHubSource(repoURL string) (*GitHubSource, error) {
	client, err := github.NewClient(repoURL)
	if err != nil {
		return nil, err
	}

	return &GitHubSource{
		client: client,
		url:    repoURL,
	}, nil
}

// Name returns the source name
func (g *GitHubSource) Name() string {
	return "github"
}

// URL returns the GitHub repository URL
func (g *GitHubSource) URL() string {
	return g.url
}

// List returns all available templates from GitHub
func (g *GitHubSource) List() ([]TemplateFile, error) {
	files, err := g.client.ListGitignoreFiles()
	if err != nil {
		return nil, err
	}

	var result []TemplateFile
	for _, f := range files {
		result = append(result, TemplateFile{
			Name:     f.Name,
			Path:     f.Path,
			Category: f.Category,
			Source:   "github",
		})
	}

	return result, nil
}

// Get returns the content of a template by name
func (g *GitHubSource) Get(name string) (*TemplateFile, string, error) {
	file, err := g.client.FindGitignoreFile(name)
	if err != nil {
		return nil, "", err
	}

	content, err := g.client.GetGitignoreContent(*file)
	if err != nil {
		return nil, "", err
	}

	return &TemplateFile{
		Name:     file.Name,
		Path:     file.Path,
		Category: file.Category,
		Source:   "github",
	}, content, nil
}

// Find finds a template by name (case-insensitive)
func (g *GitHubSource) Find(name string) (*TemplateFile, error) {
	file, err := g.client.FindGitignoreFile(name)
	if err != nil {
		return nil, err
	}

	return &TemplateFile{
		Name:     file.Name,
		Path:     file.Path,
		Category: file.Category,
		Source:   "github",
	}, nil
}
