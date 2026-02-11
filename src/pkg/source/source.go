// Package source provides abstraction for different gitignore template sources
package source

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TemplateFile represents a gitignore template
type TemplateFile struct {
	Name     string
	Path     string
	Category string
	Source   string // identifies which source this came from (local, github, toptal)
}

// Source is the interface that all template sources must implement
type Source interface {
	// Name returns the name/identifier of this source
	Name() string
	// List returns all available templates from this source
	List() ([]TemplateFile, error)
	// Get returns the content of a specific template by name
	Get(name string) (*TemplateFile, string, error)
	// Find finds a template by name (case-insensitive)
	Find(name string) (*TemplateFile, error)
}

// LocalSource handles templates from ~/.config/gitignore/
type LocalSource struct {
	dir string
}

// NewLocalSource creates a new local source
func NewLocalSource() (*LocalSource, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dir := filepath.Join(home, ".config", "gitignore")
	return &LocalSource{dir: dir}, nil
}

// NewLocalSourceWithDir creates a local source with a specific directory
func NewLocalSourceWithDir(dir string) *LocalSource {
	return &LocalSource{dir: dir}
}

// Name returns the source name
func (l *LocalSource) Name() string {
	return "local"
}

// Dir returns the local templates directory path
func (l *LocalSource) Dir() string {
	return l.dir
}

// List returns all local templates
func (l *LocalSource) List() ([]TemplateFile, error) {
	var files []TemplateFile

	entries, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return files, nil // Return empty list if directory doesn't exist
		}
		return nil, fmt.Errorf("failed to read local templates directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".gitignore") {
			continue
		}

		templateName := strings.TrimSuffix(name, ".gitignore")
		files = append(files, TemplateFile{
			Name:     templateName,
			Path:     filepath.Join(l.dir, name),
			Category: "",
			Source:   "local",
		})
	}

	return files, nil
}

// Get returns the content of a template by name
func (l *LocalSource) Get(name string) (*TemplateFile, string, error) {
	file, err := l.Find(name)
	if err != nil {
		return nil, "", err
	}

	content, err := os.ReadFile(file.Path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read local template: %w", err)
	}

	return file, string(content), nil
}

// Find finds a template by name (case-insensitive)
func (l *LocalSource) Find(name string) (*TemplateFile, error) {
	files, err := l.List()
	if err != nil {
		return nil, err
	}

	nameLower := strings.ToLower(name)
	for _, file := range files {
		if strings.ToLower(file.Name) == nameLower {
			return &file, nil
		}
	}

	return nil, fmt.Errorf("local template '%s' not found", name)
}

// Exists checks if the local templates directory exists
func (l *LocalSource) Exists() bool {
	_, err := os.Stat(l.dir)
	return err == nil
}

// EnsureDir creates the local templates directory if it doesn't exist
func (l *LocalSource) EnsureDir() error {
	return os.MkdirAll(l.dir, 0755)
}
