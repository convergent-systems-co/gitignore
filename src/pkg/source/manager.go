// Package source provides abstraction for different gitignore template sources
package source

import (
	"fmt"
	"strings"
)

// SourceManager manages multiple template sources with priority ordering
type SourceManager struct {
	local   *LocalSource
	remote  []Source
	sources []Source // all sources in order (local first, then remote)
}

// NewSourceManager creates a new source manager
// Priority order: local -> GitHub -> Toptal (if enabled)
func NewSourceManager(localPath, templateURL string, enableToptal bool) (*SourceManager, error) {
	local := NewLocalSourceWithDir(localPath)

	sm := &SourceManager{
		local:  local,
		remote: []Source{},
	}

	// Local source is always first
	sm.sources = append(sm.sources, local)

	// Add GitHub source
	githubSource, err := NewGitHubSource(templateURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub source: %w", err)
	}
	sm.remote = append(sm.remote, githubSource)
	sm.sources = append(sm.sources, githubSource)

	// Add Toptal source if enabled
	if enableToptal {
		toptalSource := NewToptalSource()
		sm.remote = append(sm.remote, toptalSource)
		sm.sources = append(sm.sources, toptalSource)
	}

	return sm, nil
}

// List returns all templates from all sources, local templates first
func (sm *SourceManager) List() ([]TemplateFile, error) {
	var allFiles []TemplateFile
	localNames := make(map[string]bool)

	// First, get local templates
	localFiles, err := sm.local.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list local templates: %w", err)
	}
	for _, f := range localFiles {
		allFiles = append(allFiles, f)
		localNames[strings.ToLower(f.Name)] = true
	}

	// Then get remote templates (mark duplicates)
	for _, source := range sm.remote {
		files, err := source.List()
		if err != nil {
			// Log warning but continue with other sources
			continue
		}
		for _, f := range files {
			// Don't add if already exists locally (local takes precedence)
			if !localNames[strings.ToLower(f.Name)] {
				allFiles = append(allFiles, f)
			}
		}
	}

	return allFiles, nil
}

// SourceResult contains the list result for a source
type SourceResult struct {
	Files []TemplateFile
	Error error
}

// ListBySource returns templates grouped by source
// Sources that fail to list are included with an empty slice (graceful degradation)
func (sm *SourceManager) ListBySource() (map[string]SourceResult, error) {
	result := make(map[string]SourceResult)

	for _, source := range sm.sources {
		files, err := source.List()
		if err != nil {
			// Include the source with error to indicate what went wrong
			result[source.Name()] = SourceResult{Files: []TemplateFile{}, Error: err}
			continue
		}
		result[source.Name()] = SourceResult{Files: files, Error: nil}
	}

	return result, nil
}

// Get retrieves a template by name, checking local first then remote sources
func (sm *SourceManager) Get(name string) (*TemplateFile, string, error) {
	// Always try local first
	file, content, err := sm.local.Get(name)
	if err == nil {
		return file, content, nil
	}

	// Try remote sources in order
	var lastErr error
	for _, source := range sm.remote {
		file, content, err := source.Get(name)
		if err == nil {
			return file, content, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, "", fmt.Errorf("template '%s' not found in any source", name)
	}

	return nil, "", fmt.Errorf("template '%s' not found", name)
}

// GetFromSource retrieves a template from a specific source
// sourceName should be "local", "github", or "toptal"
func (sm *SourceManager) GetFromSource(sourceName, templateName string) (*TemplateFile, string, error) {
	for _, source := range sm.sources {
		if source.Name() == sourceName {
			return source.Get(templateName)
		}
	}
	return nil, "", fmt.Errorf("unknown source: %s", sourceName)
}

// Find finds a template by name, checking local first
func (sm *SourceManager) Find(name string) (*TemplateFile, error) {
	// Always try local first
	file, err := sm.local.Find(name)
	if err == nil {
		return file, nil
	}

	// Try remote sources in order
	for _, source := range sm.remote {
		file, err := source.Find(name)
		if err == nil {
			return file, nil
		}
	}

	return nil, fmt.Errorf("template '%s' not found in any source", name)
}

// LocalSource returns the local source
func (sm *SourceManager) LocalSource() *LocalSource {
	return sm.local
}

// RemoteSources returns the remote sources
func (sm *SourceManager) RemoteSources() []Source {
	return sm.remote
}

// AllSources returns all sources in priority order
func (sm *SourceManager) AllSources() []Source {
	return sm.sources
}
